package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	jimuassets "jimu"
	"jimu/internal/templatesource"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string) error {
	flags := flag.NewFlagSet("bundlemeta", flag.ContinueOnError)
	root := flags.String("root", ".", "template bundle source root")
	version := flags.String("version", "dev", "template bundle version")
	commit := flags.String("commit", "", "source commit")
	contract := flags.String("contract", "v0.1", "template contract version")
	output := flags.String("output", "", "output path (default: <root>/template-bundle.json)")
	if err := flags.Parse(args); err != nil {
		return err
	}
	absolute, err := filepath.Abs(*root)
	if err != nil {
		return fmt.Errorf("resolve bundle root: %w", err)
	}
	manifest, err := templatesource.BuildManifest(os.DirFS(absolute), jimuassets.BundleRoots, templatesource.Info{
		CLIVersion:      *version,
		BundleVersion:   *version,
		Commit:          *commit,
		ContractVersion: *contract,
		Dirty:           false,
	})
	if err != nil {
		return err
	}
	filename := *output
	if filename == "" {
		filename = filepath.Join(absolute, templatesource.ManifestName)
	}
	return templatesource.WriteManifestFile(filename, manifest)
}
