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

func installStaged(root, staging string, changed, removed []string) error {
	touched := append(append([]string{}, changed...), removed...)
	sort.Strings(touched)
	if len(touched) == 0 {
		return nil
	}
	backup, err := os.MkdirTemp(filepath.Dir(root), filepath.Base(root)+".backup-")
	if err != nil {
		return fmt.Errorf("create update backup: %w", err)
	}
	defer func() { _ = os.RemoveAll(backup) }()
	backed := make([]string, 0, len(touched))
	restore := func() {
		for _, rel := range touched {
			_ = os.RemoveAll(filepath.Join(root, filepath.FromSlash(rel)))
		}
		for index := len(backed) - 1; index >= 0; index-- {
			rel := backed[index]
			from := filepath.Join(backup, filepath.FromSlash(rel))
			to := filepath.Join(root, filepath.FromSlash(rel))
			_ = os.MkdirAll(filepath.Dir(to), 0o755)
			_ = os.Rename(from, to)
		}
	}
	for _, rel := range touched {
		target := filepath.Join(root, filepath.FromSlash(rel))
		if _, err := os.Lstat(target); os.IsNotExist(err) {
			continue
		} else if err != nil {
			return fmt.Errorf("inspect update target %s: %w", rel, err)
		}
		backupPath := filepath.Join(backup, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(backupPath), 0o755); err != nil {
			return err
		}
		if err := os.Rename(target, backupPath); err != nil {
			return fmt.Errorf("backup update target %s: %w", rel, err)
		}
		backed = append(backed, rel)
	}
	for _, rel := range changed {
		source := filepath.Join(staging, filepath.FromSlash(rel))
		target := filepath.Join(root, filepath.FromSlash(rel))
		if err := copyStagedFile(source, target); err != nil {
			restore()
			return fmt.Errorf("install update target %s: %w", rel, err)
		}
	}
	for _, rel := range removed {
		if err := os.RemoveAll(filepath.Join(root, filepath.FromSlash(rel))); err != nil {
			restore()
			return fmt.Errorf("remove obsolete generated file %s: %w", rel, err)
		}
	}
	return nil
}

func copyStagedFile(source, target string) error {
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
