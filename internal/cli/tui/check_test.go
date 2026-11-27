package tui

import (
	"context"
	"testing"

	"jimu/internal/cli/actions"
	"jimu/tools/generator"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/require"
)

func TestCheckWizardMapsInputAndRendersStatus(t *testing.T) {
	var got actions.CheckInput
	run := func(_ context.Context, in actions.CheckInput) (*generator.CheckResult, error) {
		got = in
		return &generator.CheckResult{
			Dir:          in.Dir,
			Capabilities: []string{"user"},
			ManagedFiles: []generator.ManagedFile{{Path: "Makefile", Status: generator.ManagedFileClean}},
		}, nil
	}
	wizard := newCheckWizard(Options{}, context.Background(), run)
	wizard.dir.SetValue("demo")
	_, _ = wizard.Update(tea.KeyMsg{Type: tea.KeyTab})
	_, _ = wizard.Update(tea.KeyMsg{Type: tea.KeySpace})
	cmd, _ := wizard.Update(tea.KeyMsg{Type: tea.KeyEnter})
	require.NotNil(t, cmd)
	_, _ = wizard.Update(cmd())
	require.Equal(t, "demo", got.Dir)
	require.True(t, got.Deep)
	require.Contains(t, wizard.View(), "✓ PASSED")
}

func TestCheckStatusLabelsUseTextAndShape(t *testing.T) {
	tests := []struct {
		status generator.CheckStatus
		text   string
		symbol string
	}{
		{generator.CheckStatusPassed, "PASSED", "✓"},
		{generator.CheckStatusWarnings, "WARNINGS", "!"},
		{generator.CheckStatusFailed, "FAILED", "✗"},
	}
	for _, test := range tests {
		text, symbol, _ := checkStatusLabels(test.status)
		require.Equal(t, test.text, text)
		require.Equal(t, test.symbol, symbol)
	}
}
