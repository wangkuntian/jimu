package templatesource

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"jimu/tools/generator/manifest"
)

func Materialize(bundle fs.FS) (string, func() error, error) {
	root, err := os.MkdirTemp("", "jimu-template-")
	if err != nil {
		return "", nil, fmt.Errorf("create template bundle directory: %w", err)
	}
	cleanup := func() error {
		return os.RemoveAll(root)
	}

	err = fs.WalkDir(bundle, ".", func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if path == "." {
			return nil
		}
		if entry.Type()&fs.ModeSymlink != 0 {
			return fmt.Errorf("template bundle contains symlink %q", path)
		}
		target := filepath.Join(root, filepath.FromSlash(path))
		if entry.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		content, err := fs.ReadFile(bundle, path)
		if err != nil {
			return fmt.Errorf("read template bundle file %s: %w", path, err)
		}
		mode := os.FileMode(0o644)
		if strings.HasSuffix(path, ".sh") {
			mode = 0o755
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(target, content, mode); err != nil {
			return fmt.Errorf("write template bundle file %s: %w", path, err)
		}
		return nil
	})
	if err != nil {
		_ = cleanup()
		return "", nil, err
	}
	return root, cleanup, nil
}

func Embedded(bundle fs.FS, info Info) (Source, func() error, error) {
	root, cleanup, err := Materialize(bundle)
	if err != nil {
		return Source{}, nil, err
	}
	version := info.BundleVersion
	if version == "" {
		version = "dev"
	}
	commit := info.Commit
	if commit == "" {
		commit = "dev"
	}
	return Source{
		Root: root,
		Framework: &manifest.Framework{
			Module:  "jimu",
			Version: version,
			Commit:  commit,
			Dirty:   info.Dirty,
		},
	}, cleanup, nil
}
