package tui

import (
	"context"
	"testing"

	"jimu/internal/cli/actions"
	"jimu/tools/generator"
	upgradeplan "jimu/tools/generator/upgrade"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/require"
)

func TestUpgradeWizardRequiresConfirmationForApply(t *testing.T) {
	calls := 0
	var got actions.UpgradeInput
	run := func(_ context.Context, in actions.UpgradeInput) (*generator.UpgradeResult, *generator.UpgradeApplyResult, error) {
		calls++
		got = in
		report := upgradeplan.Report{Counts: map[upgradeplan.Category]int{upgradeplan.Conflict: 1}}
		return nil, &generator.UpgradeApplyResult{
			UpgradeResult: &generator.UpgradeResult{Dir: in.Dir, Report: report},
			Changed:       []string{"Makefile"},
		}, nil
	}
	wizard := newUpgradeWizard(Options{}, context.Background(), run)
	wizard.dir.SetValue("demo")

	cmd, _ := wizard.Update(tea.KeyMsg{Type: tea.KeyEnter})
	require.Nil(t, cmd)
	require.Equal(t, upgradeStepConfirm, wizard.step)
	require.Zero(t, calls)

	cmd, _ = wizard.Update(tea.KeyMsg{Type: tea.KeyEnter})
	require.NotNil(t, cmd)
	_, _ = wizard.Update(cmd())
	require.Equal(t, 1, calls)
	require.False(t, got.DryRun)
	require.False(t, got.Check)
	require.Contains(t, wizard.View(), "CONFLICT")
	require.Contains(t, wizard.View(), "Makefile")
}

func TestUpgradeWizardMapsDryRunAndCheckModes(t *testing.T) {
	for _, test := range []struct {
		steps int
		dry   bool
		check bool
	}{
		{steps: 1, dry: true},
		{steps: 2, check: true},
	} {
		var got actions.UpgradeInput
		run := func(_ context.Context, in actions.UpgradeInput) (*generator.UpgradeResult, *generator.UpgradeApplyResult, error) {
			got = in
			return &generator.UpgradeResult{Dir: in.Dir, Report: upgradeplan.Report{}}, nil, nil
		}
		wizard := newUpgradeWizard(Options{}, context.Background(), run)
		wizard.dir.SetValue("demo")
		_, _ = wizard.Update(tea.KeyMsg{Type: tea.KeyTab})
		for range test.steps {
			_, _ = wizard.Update(tea.KeyMsg{Type: tea.KeyRight})
		}
		cmd, _ := wizard.Update(tea.KeyMsg{Type: tea.KeyEnter})
		require.NotNil(t, cmd)
		_, _ = wizard.Update(cmd())
		require.Equal(t, test.dry, got.DryRun)
		require.Equal(t, test.check, got.Check)
	}
}
