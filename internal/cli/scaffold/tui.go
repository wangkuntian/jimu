package scaffold

import (
	"jimu/internal/cli/tui"
	"jimu/tools/generator"

	"github.com/spf13/cobra"
)

func tuiAvailable(cmd *cobra.Command) bool {
	flag := cmd.Flag("no-tui")
	disabled := flag != nil && flag.Value.String() == "true"
	return tui.Available(cmd.InOrStdin(), cmd.OutOrStdout(), disabled)
}

func runTUI(cmd *cobra.Command, initial tui.Screen) error {
	return tui.Run(cmd.Context(), tui.Options{
		Version:    cmd.Root().Version,
		SourceRoot: generator.FrameworkRoot(),
		Initial:    initial,
		NoColor:    tui.ColorDisabled(noColorFlag(cmd)),
		In:         cmd.InOrStdin(),
		Out:        cmd.OutOrStdout(),
	})
}

func noColorFlag(cmd *cobra.Command) bool {
	flag := cmd.Flag("no-color")
	return flag != nil && flag.Value.String() == "true"
}
