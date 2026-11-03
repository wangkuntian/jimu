package main

import (
	"jimu/internal/cli/scaffold"
	"jimu/tools/generator"

	"github.com/spf13/cobra"
)

var capabilityCmd = newCapabilityCmd()

func newCapabilityCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "capability",
		Short: "Create framework capabilities and manage generated projects",
	}
	cmd.AddCommand(newCapabilityCreateCmd())
	cmd.AddCommand(scaffold.NewAddCommand(resolveFromCheckout))
	return cmd
}

func newCapabilityCreateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "create <name>",
		Short: "Create a new capability skeleton in the framework repository",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return generator.GenerateModule(args[0])
		},
	}
}
