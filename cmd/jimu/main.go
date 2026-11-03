package main

import (
	"fmt"
	"os"
	"strconv"

	jimuassets "jimu"
	"jimu/internal/cli/scaffold"
	"jimu/internal/templatesource"

	"github.com/spf13/cobra"
)

var version = "dev"

func newRootCommand() (*cobra.Command, func() error) {
	root := &cobra.Command{
		Use:           "jimu",
		Short:         "Generate and upgrade Jimu projects",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.AddCommand(
		scaffold.NewCommand(resolveRelease),
		scaffold.NewCapabilityCommand(resolveRelease),
		scaffold.NewCheckCommand(),
		scaffold.NewUpgradeCommand(resolveRelease),
		newVersionCommand(),
	)
	return root, func() error { return nil }
}

func newVersionCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print CLI and template bundle versions",
		Run: func(cmd *cobra.Command, args []string) {
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "jimu %s\ntemplate bundle %s\ntemplate contract %s\ncommit %s\n",
				version, jimuassets.BundleVersion, jimuassets.ContractVersion, jimuassets.BundleCommit)
		},
	}
}

func resolveRelease(explicitRoot string) (templatesource.Source, func() error, error) {
	info := templatesource.Info{
		CLIVersion:      version,
		BundleVersion:   jimuassets.BundleVersion,
		Commit:          jimuassets.BundleCommit,
		ContractVersion: jimuassets.ContractVersion,
		Dirty:           false,
	}
	if value, err := strconv.ParseBool(jimuassets.BundleDirty); err == nil {
		info.Dirty = value
	}
	if explicitRoot != "" {
		source, err := templatesource.External(explicitRoot, info)
		return source, func() error { return nil }, err
	}
	return templatesource.Embedded(jimuassets.TemplateBundle, info)
}

func main() {
	cmd, cleanup := newRootCommand()
	defer func() { _ = cleanup() }()
	if err := cmd.Execute(); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
