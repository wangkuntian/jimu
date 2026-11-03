package workspace

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

func replaceDirectory(target, staging string) error {
	backup := target + ".old-"
	if _, err := os.Lstat(target); err == nil {
		var renameErr error
		backup, renameErr = os.MkdirTemp(filepath.Dir(target), filepath.Base(target)+".old-")
		if renameErr != nil {
			return fmt.Errorf("create backup directory: %w", renameErr)
		}
		_ = os.Remove(backup)
		if err := os.Rename(target, backup); err != nil {
			return fmt.Errorf("backup existing target: %w", err)
		}
	}
	if err := os.Rename(staging, target); err != nil {
		if backup != target+".old-" {
			_ = os.Rename(backup, target)
		}
		return fmt.Errorf("install generated project: %w", err)
	}
	if backup != target+".old-" {
		if err := os.RemoveAll(backup); err != nil {
			return fmt.Errorf("remove backup: %w", err)
		}
	}
	return nil
}

// InstallFile is the per-file installation hook used by workspace updates.
// Package-level callers may replace it briefly to verify rollback behavior.
var InstallFile = copyStagedFile

func copyStagedFile(sourceRoot, targetRoot, rel string) error {
	source := filepath.Join(sourceRoot, filepath.FromSlash(rel))
	target := filepath.Join(targetRoot, filepath.FromSlash(rel))
	info, err := os.Stat(source)
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("staged path is not a regular file")
	}
	content, err := os.ReadFile(source)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}
	return os.WriteFile(target, content, info.Mode().Perm())
}

func diffGenerated(root, staging string, previous, next []string) (changed, removed []string, err error) {
	nextSet := make(map[string]bool, len(next))
	for _, rel := range next {
		nextSet[rel] = true
		stagePath := filepath.Join(staging, filepath.FromSlash(rel))
		projectPath := filepath.Join(root, filepath.FromSlash(rel))
		want, readErr := os.ReadFile(stagePath)
		if readErr != nil {
			return nil, nil, fmt.Errorf("read staged generated file %s: %w", rel, readErr)
		}
		got, readErr := os.ReadFile(projectPath)
		if os.IsNotExist(readErr) {
			changed = append(changed, rel)
			continue
		}
		if readErr != nil {
			return nil, nil, fmt.Errorf("read existing generated file %s: %w", rel, readErr)
		}
		if string(got) != string(want) {
			changed = append(changed, rel)
		}
	}
	for _, rel := range previous {
		if nextSet[rel] {
			continue
		}
		if _, statErr := os.Lstat(filepath.Join(root, filepath.FromSlash(rel))); statErr == nil {
			removed = append(removed, rel)
		} else if !os.IsNotExist(statErr) {
			return nil, nil, statErr
		}
	}
	sort.Strings(changed)
	sort.Strings(removed)
	return changed, removed, nil
}
