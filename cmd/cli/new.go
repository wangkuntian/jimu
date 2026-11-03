package main

import (
	"jimu/internal/cli/scaffold"

	"github.com/spf13/cobra"
)

var newCmd = newScaffoldCmd()

func newScaffoldCmd() *cobra.Command {
	return scaffold.NewCommand(resolveFromCheckout)
}
