package tui

import (
	"context"
	"testing"

	"jimu/internal/cli/actions"
	"jimu/tools/generator"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/require"
)

func TestAddWizardMapsFlagsAndRendersChanges(t *testing.T) {
	var got actions.AddInput
	run := func(_ context.Context, in actions.AddInput) (*generator.Result, error) {
		got = in
		return &generator.Result{Dir: in.Dir, Changed: []string{"internal/capabilities/user/module.go"}}, nil
	}
	wizard := newAddWizard(Options{}, context.Background(), run)
	wizard.inputs[0].SetValue("demo")
	wizard.inputs[1].SetValue("user")
	wizard.inputs[2].SetValue("memory")
	for range 2 {
		_, _ = wizard.Update(tea.KeyMsg{Type: tea.KeyTab})
	}
	_, _ = wizard.Update(tea.KeyMsg{Type: tea.KeyTab})
	_, _ = wizard.Update(tea.KeyMsg{Type: tea.KeySpace})
	_, _ = wizard.Update(tea.KeyMsg{Type: tea.KeyTab})
	_, _ = wizard.Update(tea.KeyMsg{Type: tea.KeySpace})

	cmd, _ := wizard.Update(tea.KeyMsg{Type: tea.KeyEnter})
	require.NotNil(t, cmd)
	_, _ = wizard.Update(cmd())
	require.Equal(t, "demo", got.Dir)
	require.Equal(t, "user", got.Name)
	require.Equal(t, "memory", got.Driver)
	require.True(t, got.Force)
	require.True(t, got.DryRun)
	require.Contains(t, wizard.View(), "internal/capabilities/user/module.go")
}

func TestAddWizardRequiresCapabilityName(t *testing.T) {
	wizard := newAddWizard(Options{}, context.Background(), nil)
	cmd, _ := wizard.Update(tea.KeyMsg{Type: tea.KeyEnter})
	require.Nil(t, cmd)
	require.Equal(t, addStepForm, wizard.step)
	require.Contains(t, wizard.err, "capability name")
}
