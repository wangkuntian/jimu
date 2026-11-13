package tui

import (
	"context"
	"testing"

	"jimu/internal/cli/actions"
	"jimu/tools/generator"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/require"
)

func TestNewWizardFormRequiresDirectory(t *testing.T) {
	wizard := newNewWizard(Options{}, context.Background(), nil)
	updated, _, _ := wizard.Update(tea.KeyMsg{Type: tea.KeyEnter})
	require.Equal(t, newStepForm, updated.step)
	require.Contains(t, updated.err, "directory")
}

func TestNewWizardSummaryValidationReturnsToForm(t *testing.T) {
	wizard := newNewWizard(Options{}, context.Background(), nil)
	wizard.inputs[0].SetValue("demo")

	updated, _, _ := wizard.Update(tea.KeyMsg{Type: tea.KeyEnter})
	wizard = &updated
	require.Equal(t, newStepSummary, wizard.step)

	wizard.inputs[3].SetValue("user")
	updated, _, _ = wizard.Update(tea.KeyMsg{Type: tea.KeyEnter})
	wizard = &updated
	require.Equal(t, newStepForm, wizard.step)
	require.Contains(t, wizard.err, "mutually exclusive")
}

func TestNewWizardBackNavigation(t *testing.T) {
	wizard := newNewWizard(Options{}, context.Background(), nil)
	wizard.inputs[0].SetValue("demo")

	_, _, back := wizard.Update(tea.KeyMsg{Type: tea.KeyEsc})
	require.True(t, back)

	updated, _, _ := wizard.Update(tea.KeyMsg{Type: tea.KeyEnter})
	wizard = &updated
	require.Equal(t, newStepSummary, wizard.step)
	updated, _, back = updated.Update(tea.KeyMsg{Type: tea.KeyEsc})
	require.False(t, back)
	require.Equal(t, newStepForm, updated.step)
}

func TestNewWizardSubmitsOnceAndShowsResult(t *testing.T) {
	calls := 0
	run := func(context.Context, actions.NewInput) (*generator.Result, error) {
		calls++
		return &generator.Result{Dir: "demo", Template: "minimal", Support: "certified", FileCount: 3}, nil
	}
	wizard := newNewWizard(Options{}, context.Background(), run)
	wizard.inputs[0].SetValue("demo")

	updated, _, _ := wizard.Update(tea.KeyMsg{Type: tea.KeyEnter})
	wizard = &updated
	updated, cmd, _ := wizard.Update(tea.KeyMsg{Type: tea.KeyEnter})
	wizard = &updated
	require.Equal(t, newStepRunning, wizard.step)
	require.NotNil(t, cmd)

	updated, duplicate, _ := wizard.Update(tea.KeyMsg{Type: tea.KeyEnter})
	require.Nil(t, duplicate)
	require.Equal(t, newStepRunning, updated.step)

	msg := cmd()
	updated, _, _ = updated.Update(msg)
	require.Equal(t, newStepDone, updated.step)
	require.Equal(t, 1, calls)
	require.NotNil(t, updated.result)
	require.Contains(t, updated.View(), "Project created")
}
