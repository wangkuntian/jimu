package workspace

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"jimu/tools/generator/manifest"
)

// UpgradeApplyOptions describes the write transaction that follows a clean
// upgrade plan. Install holds the project-relative files whose staged content
// should replace the local copy.
type UpgradeApplyOptions struct {
	ProjectRoot string
	StagingRoot string
	Install     []string
	Target      manifest.Document
	Verify      func(dir string) error
}

// UpgradeApplyResult reports what the transaction committed.
type UpgradeApplyResult struct {
	Changed         []string
	ManifestUpdated bool
	Verified        bool
}

// Verify runs the post-upgrade build gate. It intentionally uses the project's
// own go.mod and test tree, so user-added packages are verified as well.
func Verify(dir string) error {
	if err := Tidy(dir); err != nil {
		return err
	}
	if err := runGo(dir, []string{"GOWORK=off"}, "build", "./..."); err != nil {
		return err
	}
	return runGo(dir, []string{"GOWORK=off"}, "test", "./...")
}

type upgradeBackup struct {
	content []byte
	mode    fs.FileMode
}

// UpgradeInstallFile is the per-file install hook used by ApplyUpgrade.
// Package-level callers may replace it briefly to verify rollback behavior.
var UpgradeInstallFile = copyStagedFile

// ApplyUpgrade installs a clean upgrade plan as one transaction. Every touched
// file is backed up first, the upgraded project is verified in place, and any
// failure restores the exact original bytes. The manifest is written only after
// verification succeeds, so a failed upgrade never leaves a new baseline behind.
func ApplyUpgrade(opts UpgradeApplyOptions) (UpgradeApplyResult, error) {
	if opts.ProjectRoot == "" || opts.StagingRoot == "" {
		return UpgradeApplyResult{}, fmt.Errorf("project root and staging root are required")
	}
	root, err := filepath.Abs(opts.ProjectRoot)
	if err != nil {
		return UpgradeApplyResult{}, fmt.Errorf("resolve project root: %w", err)
	}
	staging, err := filepath.Abs(opts.StagingRoot)
	if err != nil {
		return UpgradeApplyResult{}, fmt.Errorf("resolve staging root: %w", err)
	}
	install, err := safeUpgradePaths(opts.Install)
	if err != nil {
		return UpgradeApplyResult{}, err
	}
	manifestPath := filepath.Join(root, manifestRel)
	beforeManifest, err := os.ReadFile(manifestPath)
	if err != nil {
		return UpgradeApplyResult{}, fmt.Errorf("read project manifest: %w", err)
	}
	manifestChanged, err := manifestWouldChange(beforeManifest, opts.Target)
	if err != nil {
		return UpgradeApplyResult{}, err
	}
	if len(install) == 0 && !manifestChanged {
		return UpgradeApplyResult{}, nil
	}

	touched, err := upgradeTouchedPaths(root, install)
	if err != nil {
		return UpgradeApplyResult{}, err
	}
	backupDir, err := os.MkdirTemp(filepath.Dir(root), filepath.Base(root)+".upgrade-backup-")
	if err != nil {
		return UpgradeApplyResult{}, fmt.Errorf("create upgrade backup directory: %w", err)
	}
	defer func() { _ = os.RemoveAll(backupDir) }()
	backups := make(map[string]upgradeBackup, len(touched))
	for _, rel := range touched {
		path := filepath.Join(root, filepath.FromSlash(rel))
		info, statErr := os.Lstat(path)
		if os.IsNotExist(statErr) {
			continue
		}
		if statErr != nil {
			return UpgradeApplyResult{}, fmt.Errorf("inspect upgrade target %s: %w", rel, statErr)
		}
		if !info.Mode().IsRegular() {
			return UpgradeApplyResult{}, fmt.Errorf("upgrade target %s is not a regular file", rel)
		}
		content, readErr := os.ReadFile(path)
		if readErr != nil {
			return UpgradeApplyResult{}, fmt.Errorf("backup upgrade target %s: %w", rel, readErr)
		}
		backups[rel] = upgradeBackup{content: content, mode: info.Mode().Perm()}
	}
	createdDirs := missingParentDirs(root, touched)
	rollback := func() error {
		return restoreUpgrade(root, touched, backups, createdDirs)
	}
	for _, rel := range install {
		if err := UpgradeInstallFile(staging, root, rel); err != nil {
			restoreErr := rollback()
			if restoreErr != nil {
				return UpgradeApplyResult{}, fmt.Errorf("install upgrade target %s: %w (rollback failed: %v)", rel, err, restoreErr)
			}
			return UpgradeApplyResult{}, fmt.Errorf("install upgrade target %s: %w", rel, err)
		}
	}
	verify := opts.Verify
	if verify == nil {
		verify = Verify
	}
	if err := verify(root); err != nil {
		restoreErr := rollback()
		if restoreErr != nil {
			return UpgradeApplyResult{}, fmt.Errorf("verify upgraded project: %w (rollback failed: %v)", err, restoreErr)
		}
		return UpgradeApplyResult{}, fmt.Errorf("verify upgraded project: %w", err)
	}
	if err := writeUpgradedManifest(manifestPath, root, opts.Target); err != nil {
		restoreErr := rollback()
		if restoreErr != nil {
			return UpgradeApplyResult{}, fmt.Errorf("write upgraded manifest: %w (rollback failed: %v)", err, restoreErr)
		}
		return UpgradeApplyResult{}, fmt.Errorf("write upgraded manifest: %w", err)
	}
	afterManifest, err := os.ReadFile(manifestPath)
	if err != nil {
		return UpgradeApplyResult{}, fmt.Errorf("read upgraded manifest: %w", err)
	}
	return UpgradeApplyResult{
		Changed:         install,
		ManifestUpdated: !bytes.Equal(beforeManifest, afterManifest),
		Verified:        true,
	}, nil
}

func manifestWouldChange(before []byte, target manifest.Document) (bool, error) {
	canonical, err := manifest.CanonicalJSON(target)
	if err != nil {
		return false, fmt.Errorf("encode target manifest: %w", err)
	}
	return !bytes.Equal(before, append(canonical, '\n')), nil
}

func writeUpgradedManifest(path, root string, target manifest.Document) error {
	hashes, err := manifest.HashFiles(root, target.GeneratedFiles)
	if err != nil {
		return fmt.Errorf("hash upgraded generated files: %w", err)
	}
	target.GeneratedFileSHA256 = hashes
	return manifest.Write(path, target)
}

func safeUpgradePaths(paths []string) ([]string, error) {
	seen := make(map[string]bool, len(paths))
	out := make([]string, 0, len(paths))
	for _, rel := range paths {
		clean := filepath.Clean(filepath.FromSlash(rel))
		if rel == "" || filepath.IsAbs(clean) || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
			return nil, fmt.Errorf("invalid upgrade path %q", rel)
		}
		slashed := filepath.ToSlash(clean)
		if slashed == manifestRel {
			continue
		}
		if !seen[slashed] {
			seen[slashed] = true
			out = append(out, slashed)
		}
	}
	sort.Strings(out)
	return out, nil
}

// upgradeTouchedPaths includes the manifest and the module files because the
// post-install tidy may rewrite them even when they are not part of the plan.
func upgradeTouchedPaths(root string, install []string) ([]string, error) {
	seen := make(map[string]bool, len(install)+3)
	for _, rel := range install {
		seen[rel] = true
	}
	seen[manifestRel] = true
	for _, rel := range []string{"go.mod", "go.sum"} {
		if _, err := os.Lstat(filepath.Join(root, rel)); err == nil {
			seen[rel] = true
		} else if !os.IsNotExist(err) {
			return nil, fmt.Errorf("inspect %s: %w", rel, err)
		}
	}
	out := make([]string, 0, len(seen))
	for rel := range seen {
		out = append(out, rel)
	}
	sort.Strings(out)
	return out, nil
}

func missingParentDirs(root string, paths []string) []string {
	seen := map[string]bool{}
	for _, rel := range paths {
		dir := filepath.Dir(filepath.Join(root, filepath.FromSlash(rel)))
		for dir != root && strings.HasPrefix(dir, root+string(filepath.Separator)) {
			if _, err := os.Lstat(dir); os.IsNotExist(err) {
				seen[dir] = true
			}
			dir = filepath.Dir(dir)
		}
	}
	out := make([]string, 0, len(seen))
	for dir := range seen {
		out = append(out, dir)
	}
	sort.Slice(out, func(i, j int) bool {
		if len(out[i]) != len(out[j]) {
			return len(out[i]) > len(out[j])
		}
		return out[i] > out[j]
	})
	return out
}

func restoreUpgrade(root string, touched []string, backups map[string]upgradeBackup, createdDirs []string) error {
	var errs []error
	for _, rel := range touched {
		path := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			errs = append(errs, fmt.Errorf("remove upgraded file %s: %w", rel, err))
		}
	}
	for _, rel := range touched {
		backup, ok := backups[rel]
		if !ok {
			continue
		}
		path := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			errs = append(errs, fmt.Errorf("restore directory for %s: %w", rel, err))
			continue
		}
		if err := os.WriteFile(path, backup.content, backup.mode); err != nil {
			errs = append(errs, fmt.Errorf("restore %s: %w", rel, err))
		}
	}
	for _, dir := range createdDirs {
		// Best effort: a non-empty directory means it existed for another
		// restored file, which is exactly the state we want.
		_ = os.Remove(dir)
	}
	return errors.Join(errs...)
}
