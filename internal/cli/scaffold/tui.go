package scaffold

import (
	"jimu/internal/cli/tui"
	"jimu/tools/generator"

	"github.com/spf13/cobra"
)

func tuiAvailable(cmd *cobra.Command) bool {
	disabled := false
	if cmd.Flags().Lookup("no-tui") != nil {
		disabled, _ = cmd.Flags().GetBool("no-tui")
	} else if cmd.InheritedFlags().Lookup("no-tui") != nil {
		disabled, _ = cmd.InheritedFlags().GetBool("no-tui")
	}
	return tui.Available(cmd.InOrStdin(), cmd.OutOrStdout(), disabled)
}

func runTUI(cmd *cobra.Command, initial tui.Screen) error {
	return tui.Run(cmd.Context(), tui.Options{
		Version:    cmd.Root().Version,
		SourceRoot: generator.FrameworkRoot(),
		Initial:    initial,
		In:         cmd.InOrStdin(),
		Out:        cmd.OutOrStdout(),
	})
}
