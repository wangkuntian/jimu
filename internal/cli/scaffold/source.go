package scaffold

import (
	"os"

	"jimu/internal/templatesource"

	"github.com/spf13/cobra"
)

func resolveSource(cmd *cobra.Command, resolve templatesource.Resolver) (templatesource.Source, func() error, error) {
	if resolve == nil {
		return templatesource.Source{}, func() error { return nil }, nil
	}
	from, _ := cmd.Flags().GetString("from")
	if from == "" {
		from = os.Getenv("JIMU_TEMPLATE_SOURCE")
	}
	source, cleanup, err := resolve(from)
	if cleanup == nil {
		cleanup = func() error { return nil }
	}
	return source, cleanup, err
}

func addSourceFlag(cmd *cobra.Command) {
	cmd.Flags().String("from", "", "template bundle source root (development/test only)")
	_ = cmd.Flags().MarkHidden("from")
}
