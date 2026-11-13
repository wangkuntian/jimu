package main

import (
	"errors"

	"jimu/internal/cli/tui"

	"github.com/spf13/cobra"
)

var errTUIRequiresTerminal = errors.New("tui requires an interactive terminal")

func newTUICmd() *cobra.Command {
	return &cobra.Command{
		Use:   "tui",
		Short: "Open the interactive terminal UI",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if !tuiAvailable(cmd) {
				return errTUIRequiresTerminal
			}
			return runTUI(cmd, tui.ScreenMenu)
		},
	}
}

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
	source, cleanup, err := resolveRelease("")
	if err != nil {
		return err
	}
	defer func() { _ = cleanup() }()
	return tui.Run(cmd.Context(), tui.Options{
		Version:    version,
		SourceRoot: source.Root,
		Framework:  source.Framework,
		Initial:    initial,
		In:         cmd.InOrStdin(),
		Out:        cmd.OutOrStdout(),
	})
}
