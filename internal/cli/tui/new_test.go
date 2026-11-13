package tui

import (
	"context"
	"strings"
	"testing"

	"jimu/internal/cli/actions"
	"jimu/tools/generator"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/stretchr/testify/require"
)

func enterNewDetails(t *testing.T, wizard *newWizard) *newWizard {
	t.Helper()
	updated, _, _ := wizard.Update(tea.KeyMsg{Type: tea.KeyEnter})
	require.Equal(t, newStepForm, updated.step)
	return &updated
}

func enterNewSummary(t *testing.T, wizard *newWizard, dir string) *newWizard {
	t.Helper()
	wizard.inputs[newFieldDir].SetValue(dir)
	updated, _, _ := wizard.Update(tea.KeyMsg{Type: tea.KeyEnter})
	wizard = &updated
	require.Equal(t, 1, wizard.focus)
	updated, _, _ = wizard.Update(tea.KeyMsg{Type: tea.KeyEnter})
	wizard = &updated
	require.Equal(t, newStepSummary, wizard.step)
	return wizard
}

func TestNewWizardPresetSelectionAndStepBar(t *testing.T) {
	wizard := newNewWizard(Options{}, context.Background(), nil)
	require.Equal(t, newStepPreset, wizard.step)
	require.Contains(t, wizard.View(), "1 Template")
	require.Contains(t, wizard.View(), "Minimal")

	updated, _, _ := wizard.Update(tea.KeyMsg{Type: tea.KeyDown})
	require.Equal(t, 1, updated.presetCursor)
	require.Contains(t, updated.View(), "Auth")

	updated, _, _ = updated.Update(tea.KeyMsg{Type: tea.KeyEnter})
	require.Equal(t, newStepForm, updated.step)
	require.Equal(t, "auth", updated.inputs[newFieldProfile].Value())
	require.Contains(t, updated.View(), "❯ Directory:")
	require.Contains(t, updated.View(), "the path is created atomically")
}

func TestNewWizardPresetSearchFiltersAndSelects(t *testing.T) {
	wizard := newNewWizard(Options{}, context.Background(), nil)
	for _, ch := range "auth" {
		updated, _, _ := wizard.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{ch}})
		wizard = &updated
	}
	require.Equal(t, "auth", wizard.presetQuery)
	require.Equal(t, 1, wizard.presetCursor)
	require.Contains(t, wizard.View(), "Auth")
	require.NotContains(t, wizard.View(), "Minimal  lean")

	updated, _, _ := wizard.Update(tea.KeyMsg{Type: tea.KeyEnter})
	require.Equal(t, newStepForm, updated.step)
	require.Equal(t, "auth", updated.inputs[newFieldProfile].Value())
}

func TestNewWizardPresetColumnsAlign(t *testing.T) {
	configureStyles(true)
	t.Cleanup(func() { configureStyles(false) })

	lines := strings.Split(newNewWizard(Options{}, context.Background(), nil).View(), "\n")
	var columns []int
	for _, badge := range []string{"lean", "ready", "complete", "expert"} {
		found := false
		for _, line := range lines {
			if index := strings.Index(line, " "+badge); index >= 0 {
				columns = append(columns, lipgloss.Width(line[:index+1]))
				found = true
				break
			}
		}
		require.True(t, found, "missing badge %q", badge)
	}
	for _, column := range columns[1:] {
		require.Equal(t, columns[0], column)
	}
}

func TestNewWizardFormRequiresDirectory(t *testing.T) {
	wizard := enterNewDetails(t, newNewWizard(Options{}, context.Background(), nil))
	updated, _, _ := wizard.Update(tea.KeyMsg{Type: tea.KeyEnter})
	require.Equal(t, newStepForm, updated.step)
	require.Contains(t, updated.err, "directory")
}

func TestNewWizardAdvancedOptionsAreCollapsedByDefault(t *testing.T) {
	wizard := enterNewDetails(t, newNewWizard(Options{}, context.Background(), nil))
	require.NotContains(t, wizard.View(), "Database:")
	require.Contains(t, wizard.View(), "Advanced settings [ ]")

	updated, _, _ := wizard.Update(tea.KeyMsg{Type: tea.KeyTab})
	updated, _, _ = updated.Update(tea.KeyMsg{Type: tea.KeyTab})
	updated, _, _ = updated.Update(tea.KeyMsg{Type: tea.KeyEnter})
	require.True(t, updated.advanced)
	require.Contains(t, updated.View(), "Database:")
	require.Contains(t, updated.View(), "Advanced settings [x]")
}

func TestNewWizardCustomPresetRequiresCapability(t *testing.T) {
	wizard := newNewWizard(Options{}, context.Background(), nil)
	for range 3 {
		updated, _, _ := wizard.Update(tea.KeyMsg{Type: tea.KeyDown})
		wizard = &updated
	}
	wizard = enterNewDetails(t, wizard)
	require.Equal(t, "", wizard.inputs[newFieldProfile].Value())
	require.True(t, wizard.advanced)
	require.Contains(t, wizard.View(), "Capabilities:")

	wizard.inputs[newFieldDir].SetValue("demo")
	wizard.focus = len(wizard.visibleFields()) - 1
	updated, _, _ := wizard.Update(tea.KeyMsg{Type: tea.KeyEnter})
	require.Equal(t, newStepForm, updated.step)
	require.Contains(t, updated.err, "custom preset")
}

func TestNewWizardSummaryValidationReturnsToForm(t *testing.T) {
	wizard := enterNewDetails(t, newNewWizard(Options{}, context.Background(), nil))
	wizard.inputs[newFieldDir].SetValue("demo")
	wizard.inputs[newFieldProfile].SetValue("minimal")
	wizard.inputs[newFieldWith].SetValue("user")
	wizard.step = newStepSummary

	updated, _, _ := wizard.Update(tea.KeyMsg{Type: tea.KeyEnter})
	wizard = &updated
	require.Equal(t, newStepForm, wizard.step)
	require.Contains(t, wizard.err, "mutually exclusive")
}

func TestNewWizardBackNavigation(t *testing.T) {
	wizard := newNewWizard(Options{}, context.Background(), nil)
	_, _, back := wizard.Update(tea.KeyMsg{Type: tea.KeyEsc})
	require.True(t, back, "preset step returns to the menu")

	wizard = enterNewDetails(t, newNewWizard(Options{}, context.Background(), nil))
	updated, _, back := wizard.Update(tea.KeyMsg{Type: tea.KeyEsc})
	require.False(t, back, "details returns to the preset step")
	require.Equal(t, newStepPreset, updated.step)
}

func TestNewWizardSubmitsOnceAndShowsResult(t *testing.T) {
	calls := 0
	run := func(context.Context, actions.NewInput) (*generator.Result, error) {
		calls++
		return &generator.Result{Dir: "demo", Template: "minimal", Support: "certified", FileCount: 3}, nil
	}
	wizard := enterNewDetails(t, newNewWizard(Options{}, context.Background(), run))
	wizard = enterNewSummary(t, wizard, "demo")

	updated, cmd, _ := wizard.Update(tea.KeyMsg{Type: tea.KeyEnter})
	wizard = &updated
	require.Equal(t, newStepRunning, wizard.step)
	require.NotNil(t, cmd)
	require.NotEmpty(t, wizard.spinner.View())

	updated, duplicate, _ := wizard.Update(tea.KeyMsg{Type: tea.KeyEnter})
	require.Nil(t, duplicate)
	require.Equal(t, newStepRunning, updated.step)

	msg := findNewRunMsg(t, cmd())
	updated, _, _ = updated.Update(msg)
	require.Equal(t, newStepDone, updated.step)
	require.Equal(t, 1, calls)
	require.NotNil(t, updated.result)
	require.Contains(t, updated.View(), "Project created")
}

func findNewRunMsg(t *testing.T, msg tea.Msg) newRunMsg {
	t.Helper()
	if batch, ok := msg.(tea.BatchMsg); ok {
		for _, cmd := range batch {
			if cmd == nil {
				continue
			}
			if run, ok := cmd().(newRunMsg); ok {
				return run
			}
		}
	}
	run, ok := msg.(newRunMsg)
	require.True(t, ok, "expected newRunMsg, got %T", msg)
	return run
}
