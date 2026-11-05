package main

import (
	"jimu/internal/cli/scaffold"

	"github.com/spf13/cobra"
)

var checkCmd = newCheckCmd()

func newCheckCmd() *cobra.Command {
	return scaffold.NewCheckCommand(resolveFromCheckout)
}

func init() {
	rootCmd.AddCommand(checkCmd)
}
