package templatesource

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"os"
	"path"
	"path/filepath"
	"strings"
)

const (
	ManifestName          = "template-bundle.json"
	manifestSchemaVersion = 1
)

type Info struct {
	CLIVersion      string
	BundleVersion   string
	Commit          string
	ContractVersion string
	Dirty           bool
}

type Manifest struct {
	SchemaVersion   int               `json:"schema_version"`
	BundleVersion   string            `json:"bundle_version"`
	Commit          string            `json:"commit"`
	ContractVersion string            `json:"contract_version"`
	Files           map[string]string `json:"files"`
}

func BuildManifest(bundle fs.FS, roots []string, info Info) (Manifest, error) {
	files := map[string]string{}
	for _, root := range roots {
		root = path.Clean(root)
		if root == "." || root == "" {
			return Manifest{}, fmt.Errorf("template bundle root must not be empty")
		}
		entry, err := fs.Stat(bundle, root)
		if err != nil {
			return Manifest{}, fmt.Errorf("inspect template bundle root %s: %w", root, err)
		}
		if !entry.IsDir() {
			if err := addManifestFile(bundle, root, files); err != nil {
				return Manifest{}, err
			}
			continue
		}
		if err := fs.WalkDir(bundle, root, func(file string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if entry.IsDir() {
				return nil
			}
			return addManifestFile(bundle, file, files)
		}); err != nil {
			return Manifest{}, fmt.Errorf("walk template bundle root %s: %w", root, err)
		}
	}
	return Manifest{
		SchemaVersion:   manifestSchemaVersion,
		BundleVersion:   info.BundleVersion,
		Commit:          info.Commit,
		ContractVersion: info.ContractVersion,
		Files:           files,
	}, nil
}

func addManifestFile(bundle fs.FS, rel string, files map[string]string) error {
	clean := path.Clean(rel)
	if clean == ManifestName || isTransient(clean) {
		return nil
	}
	content, err := fs.ReadFile(bundle, clean)
	if err != nil {
		return fmt.Errorf("read template bundle file %s: %w", clean, err)
	}
	sum := sha256.Sum256(content)
	files[clean] = "sha256:" + hex.EncodeToString(sum[:])
	return nil
}

func LoadManifest(root string) (Manifest, error) {
	content, err := os.ReadFile(filepath.Join(root, ManifestName))
	if err != nil {
		return Manifest{}, fmt.Errorf("read %s: %w", ManifestName, err)
	}
	var manifest Manifest
	if err := json.Unmarshal(content, &manifest); err != nil {
		return Manifest{}, fmt.Errorf("decode %s: %w", ManifestName, err)
	}
	if manifest.SchemaVersion != manifestSchemaVersion {
		return Manifest{}, fmt.Errorf("unsupported %s schema version %d", ManifestName, manifest.SchemaVersion)
	}
	return manifest, nil
}

func WriteManifest(root string, manifest Manifest) error {
	return WriteManifestFile(filepath.Join(root, ManifestName), manifest)
}

func WriteManifestFile(filename string, manifest Manifest) error {
	content, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("encode %s: %w", ManifestName, err)
	}
	content = append(content, '\n')
	if err := os.MkdirAll(filepath.Dir(filename), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(filename, content, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", ManifestName, err)
	}
	return nil
}

func Validate(bundle fs.FS, manifest Manifest) error {
	if manifest.SchemaVersion != manifestSchemaVersion {
		return fmt.Errorf("unsupported %s schema version %d", ManifestName, manifest.SchemaVersion)
	}
	for rel, want := range manifest.Files {
		content, err := fs.ReadFile(bundle, path.Clean(rel))
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				return fmt.Errorf("template bundle file %s is missing", rel)
			}
			return fmt.Errorf("read template bundle file %s: %w", rel, err)
		}
		sum := sha256.Sum256(content)
		got := "sha256:" + hex.EncodeToString(sum[:])
		if got != want {
			return fmt.Errorf("template bundle hash mismatch for %s", rel)
		}
	}
	return nil
}

func isTransient(rel string) bool {
	base := path.Base(rel)
	return base == ".DS_Store" ||
		strings.Contains(rel, "/.idea/") || strings.Contains(rel, "/.vscode/") ||
		strings.HasSuffix(rel, ".tmp-") || strings.Contains(rel, "/.tmp-")
}

func manifestsMatch(a, b Manifest) bool {
	return a.SchemaVersion == b.SchemaVersion &&
		a.BundleVersion == b.BundleVersion &&
		a.Commit == b.Commit &&
		a.ContractVersion == b.ContractVersion &&
		maps.Equal(a.Files, b.Files)
}
