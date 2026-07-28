package main

import (
	"fmt"
	"os"

	"jimu/tools/generator"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "jimu",
	Short: "Jimu backend framework CLI",
}

var moduleCmd = &cobra.Command{
	Use:   "module",
	Short: "Module management commands",
}

var moduleCreateCmd = &cobra.Command{
	Use:   "create [name]",
	Short: "Create a new module skeleton",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		return generator.GenerateModule(name)
	},
}

func init() {
	moduleCmd.AddCommand(moduleCreateCmd)
	rootCmd.AddCommand(moduleCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
