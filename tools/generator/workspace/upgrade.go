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

// TransactionOptions describes one manifest-backed write transaction. Install
// and Remove are project-relative paths; Backup lists paths that may be changed
// as a side effect (for example go.mod when verification runs) and therefore
// must be restored on failure.
type TransactionOptions struct {
	ProjectRoot string
	StagingRoot string
	Install     []string
	Remove      []string
	Backup      []string
	Manifest    *manifest.Document
	Verify      func(dir string) error
}

// TransactionResult reports what was committed.
type TransactionResult struct {
	Changed         []string
	Removed         []string
	ManifestUpdated bool
	Verified        bool
}

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

type fileBackup struct {
	content []byte
	mode    fs.FileMode
}

// UpgradeInstallFile is the per-file install hook used by ApplyUpgrade.
// Package-level callers may replace it briefly to verify rollback behavior.
var UpgradeInstallFile = copyStagedFile

// ApplyTransaction installs staged files and updates the manifest as one
// transaction. Every touched file is backed up first, optional verification
// runs before the manifest is written, and any failure restores the exact
// original bytes.
func ApplyTransaction(opts TransactionOptions) (TransactionResult, error) {
	return applyTransaction(opts, InstallFile)
}

func applyTransaction(opts TransactionOptions, installFile func(sourceRoot, targetRoot, rel string) error) (TransactionResult, error) {
	if opts.ProjectRoot == "" {
		return TransactionResult{}, fmt.Errorf("project root is required")
	}
	root, err := filepath.Abs(opts.ProjectRoot)
	if err != nil {
		return TransactionResult{}, fmt.Errorf("resolve project root: %w", err)
	}
	staging := ""
	if len(opts.Install) > 0 {
		if opts.StagingRoot == "" {
			return TransactionResult{}, fmt.Errorf("staging root is required for installs")
		}
		staging, err = filepath.Abs(opts.StagingRoot)
		if err != nil {
			return TransactionResult{}, fmt.Errorf("resolve staging root: %w", err)
		}
	}
	install, err := safeTransactionPaths(opts.Install)
	if err != nil {
		return TransactionResult{}, err
	}
	remove, err := safeTransactionPaths(opts.Remove)
	if err != nil {
		return TransactionResult{}, err
	}
	extra, err := safeTransactionPaths(opts.Backup)
	if err != nil {
		return TransactionResult{}, err
	}
	manifestPath := filepath.Join(root, manifestRel)
	var beforeManifest []byte
	manifestChanged := false
	if opts.Manifest != nil {
		beforeManifest, err = os.ReadFile(manifestPath)
		if err != nil {
			return TransactionResult{}, fmt.Errorf("read project manifest: %w", err)
		}
		manifestChanged, err = manifestWouldChange(beforeManifest, *opts.Manifest)
		if err != nil {
			return TransactionResult{}, err
		}
	}
	if len(install) == 0 && len(remove) == 0 && !manifestChanged {
		return TransactionResult{}, nil
	}

	touched := transactionPaths(install, remove, extra)
	if opts.Manifest != nil {
		touched = transactionPaths(touched, []string{manifestRel})
	}
	backups := make(map[string]fileBackup, len(touched))
	for _, rel := range touched {
		path := filepath.Join(root, filepath.FromSlash(rel))
		info, statErr := os.Lstat(path)
		if os.IsNotExist(statErr) {
			continue
		}
		if statErr != nil {
			return TransactionResult{}, fmt.Errorf("inspect transaction target %s: %w", rel, statErr)
		}
		if !info.Mode().IsRegular() {
			return TransactionResult{}, fmt.Errorf("transaction target %s is not a regular file", rel)
		}
		content, readErr := os.ReadFile(path)
		if readErr != nil {
			return TransactionResult{}, fmt.Errorf("backup transaction target %s: %w", rel, readErr)
		}
		backups[rel] = fileBackup{content: content, mode: info.Mode().Perm()}
	}
	createdDirs := missingParentDirs(root, touched)
	rollback := func() error {
		return restoreTransaction(root, touched, backups, createdDirs)
	}
	fail := func(action string, cause error) (TransactionResult, error) {
		if restoreErr := rollback(); restoreErr != nil {
			return TransactionResult{}, fmt.Errorf("%s: %w (rollback failed: %v)", action, cause, restoreErr)
		}
		return TransactionResult{}, fmt.Errorf("%s: %w", action, cause)
	}

	for _, rel := range install {
		if err := installFile(staging, root, rel); err != nil {
			return fail("install transaction target "+rel, err)
		}
	}
	for _, rel := range remove {
		path := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			return fail("remove transaction target "+rel, err)
		}
	}
	if opts.Verify != nil {
		if err := opts.Verify(root); err != nil {
			return fail("verify upgraded project", err)
		}
	}
	if opts.Manifest != nil && manifestChanged {
		target := *opts.Manifest
		hashes, err := manifest.HashFiles(root, target.GeneratedFiles)
		if err != nil {
			return fail("hash transaction generated files", err)
		}
		target.GeneratedFileSHA256 = hashes
		if err := manifest.Write(manifestPath, target); err != nil {
			return fail("write upgraded manifest", err)
		}
	}
	afterManifest := beforeManifest
	if opts.Manifest != nil {
		afterManifest, err = os.ReadFile(manifestPath)
		if err != nil {
			return TransactionResult{}, fmt.Errorf("read upgraded manifest: %w", err)
		}
	}
	return TransactionResult{
		Changed:         install,
		Removed:         remove,
		ManifestUpdated: opts.Manifest != nil && !bytes.Equal(beforeManifest, afterManifest),
		Verified:        opts.Verify != nil,
	}, nil
}

// ApplyUpgrade installs a clean upgrade plan as one transaction. Every touched
// file is backed up first, the upgraded project is verified in place, and any
// failure restores the exact original bytes. The manifest is written only after
// verification succeeds, so a failed upgrade never leaves a new baseline behind.
func ApplyUpgrade(opts UpgradeApplyOptions) (UpgradeApplyResult, error) {
	if opts.ProjectRoot == "" || opts.StagingRoot == "" {
		return UpgradeApplyResult{}, fmt.Errorf("project root and staging root are required")
	}
	verify := opts.Verify
	if verify == nil {
		verify = Verify
	}
	result, err := applyTransaction(TransactionOptions{
		ProjectRoot: opts.ProjectRoot,
		StagingRoot: opts.StagingRoot,
		Install:     opts.Install,
		Backup:      []string{"go.mod", "go.sum"},
		Manifest:    &opts.Target,
		Verify:      verify,
	}, UpgradeInstallFile)
	if err != nil {
		return UpgradeApplyResult{}, err
	}
	return UpgradeApplyResult{
		Changed:         result.Changed,
		ManifestUpdated: result.ManifestUpdated,
		Verified:        result.Verified,
	}, nil
}

func manifestWouldChange(before []byte, target manifest.Document) (bool, error) {
	target.Digest = ""
	digest, err := manifest.Digest(target)
	if err != nil {
		return false, fmt.Errorf("encode target manifest: %w", err)
	}
	target.Digest = digest
	canonical, err := manifest.CanonicalJSON(target)
	if err != nil {
		return false, fmt.Errorf("encode target manifest: %w", err)
	}
	return !bytes.Equal(before, append(canonical, '\n')), nil
}

func safeTransactionPaths(paths []string) ([]string, error) {
	seen := make(map[string]bool, len(paths))
	out := make([]string, 0, len(paths))
	for _, rel := range paths {
		clean := filepath.Clean(filepath.FromSlash(rel))
		if rel == "" || filepath.IsAbs(clean) || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
			return nil, fmt.Errorf("invalid transaction path %q", rel)
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

func transactionPaths(groups ...[]string) []string {
	seen := make(map[string]bool)
	for _, group := range groups {
		for _, rel := range group {
			seen[rel] = true
		}
	}
	out := make([]string, 0, len(seen))
	for rel := range seen {
		out = append(out, rel)
	}
	sort.Strings(out)
	return out
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

func restoreTransaction(root string, touched []string, backups map[string]fileBackup, createdDirs []string) error {
	var errs []error
	for _, rel := range touched {
		path := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			errs = append(errs, fmt.Errorf("remove transaction file %s: %w", rel, err))
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
