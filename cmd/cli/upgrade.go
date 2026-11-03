package main

import (
	"jimu/internal/cli/scaffold"

	"github.com/spf13/cobra"
)

var upgradeCmd = newUpgradeCmd()

func newUpgradeCmd() *cobra.Command {
	return scaffold.NewUpgradeCommand(resolveFromCheckout)
}

func init() {
	rootCmd.AddCommand(upgradeCmd)
}
