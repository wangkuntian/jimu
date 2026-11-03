package templatesource

import (
	"fmt"
	"os"
	"path/filepath"

	jimuassets "jimu"
	"jimu/tools/generator/manifest"
)

func External(root string, info Info) (Source, error) {
	return externalWithRoots(root, jimuassets.BundleRoots, info)
}

func externalWithRoots(root string, roots []string, info Info) (Source, error) {
	absolute, err := filepath.Abs(root)
	if err != nil {
		return Source{}, fmt.Errorf("resolve external bundle root: %w", err)
	}
	actual, err := LoadManifest(absolute)
	if err != nil {
		return Source{}, err
	}
	if info.CLIVersion != "" && actual.BundleVersion != info.CLIVersion {
		return Source{}, fmt.Errorf("template bundle version %q does not match CLI version %q", actual.BundleVersion, info.CLIVersion)
	}
	if actual.BundleVersion != info.BundleVersion {
		return Source{}, fmt.Errorf("template bundle version %q does not match expected version %q", actual.BundleVersion, info.BundleVersion)
	}
	if actual.Commit != info.Commit {
		return Source{}, fmt.Errorf("template bundle commit %q does not match CLI commit %q", actual.Commit, info.Commit)
	}
	if actual.ContractVersion != info.ContractVersion {
		return Source{}, fmt.Errorf("template bundle contract version %q does not match expected %q", actual.ContractVersion, info.ContractVersion)
	}

	bundle := os.DirFS(absolute)
	if err := Validate(bundle, actual); err != nil {
		return Source{}, err
	}
	expected, err := BuildManifest(bundle, roots, info)
	if err != nil {
		return Source{}, err
	}
	if !manifestsMatch(actual, expected) {
		return Source{}, fmt.Errorf("template bundle file set or hashes do not match source root")
	}
	return Source{
		Root: absolute,
		Framework: &manifest.Framework{
			Module:  "jimu",
			Version: info.BundleVersion,
			Commit:  info.Commit,
			Dirty:   info.Dirty,
		},
	}, nil
}
