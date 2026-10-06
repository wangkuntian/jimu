package render

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

func copyTree(sourceRoot, destinationRoot, source, destination string, includes, excludes []string) error {
	sourcePath, err := safeJoin(sourceRoot, source)
	if err != nil {
		return fmt.Errorf("copy source: %w", err)
	}
	destinationPath, err := safeJoin(destinationRoot, destination)
	if err != nil {
		return fmt.Errorf("copy destination: %w", err)
	}
	info, err := os.Stat(sourcePath)
	if err != nil {
		return fmt.Errorf("stat %s: %w", source, err)
	}
	if !info.IsDir() {
		if len(includes) > 0 && !matchesInclude(".", filepath.Base(sourcePath), includes) {
			return nil
		}
		return copyFile(sourcePath, destinationPath, info.Mode().Perm())
	}
	return filepath.WalkDir(sourcePath, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(sourcePath, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if rel != "." && isTransientEntry(entry.Name()) {
			if entry.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		if rel != "." && excluded(rel, entry, excludes) {
			if entry.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		if rel != "." && len(includes) > 0 && !matchesInclude(rel, entry.Name(), includes) {
			if entry.IsDir() && !hasIncludedDescendant(rel, includes) {
				return fs.SkipDir
			}
			if !entry.IsDir() {
				return nil
			}
		}
		if entry.Type()&fs.ModeSymlink != 0 {
			return fmt.Errorf("refusing symlink %s", path)
		}
		if entry.IsDir() {
			return os.MkdirAll(filepath.Join(destinationPath, filepath.FromSlash(rel)), 0o755)
		}
		fileInfo, err := entry.Info()
		if err != nil {
			return err
		}
		return copyFile(path, filepath.Join(destinationPath, filepath.FromSlash(rel)), fileInfo.Mode().Perm())
	})
}

func isTransientEntry(name string) bool {
	switch name {
	case ".git", ".DS_Store", ".idea", ".vscode":
		return true
	}
	return strings.HasSuffix(name, ".tmp-") || (strings.HasPrefix(name, ".") && strings.Contains(name, ".tmp-"))
}

func copyFile(source, destination string, mode os.FileMode) error {
	content, err := os.ReadFile(source)
	if err != nil {
		return fmt.Errorf("read %s: %w", source, err)
	}
	if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(destination, content, mode); err != nil {
		return fmt.Errorf("write %s: %w", destination, err)
	}
	return nil
}

func excluded(rel string, entry fs.DirEntry, values []string) bool {
	for _, value := range values {
		if value == "*_test.go" && strings.HasSuffix(entry.Name(), "_test.go") {
			return true
		}
		if rel == value || strings.HasPrefix(rel, value+"/") {
			return true
		}
	}
	return false
}

func matchesInclude(rel, name string, values []string) bool {
	for _, value := range values {
		if value == rel || value == name || strings.HasPrefix(rel, value+"/") {
			return true
		}
		if matched, err := filepath.Match(value, name); err == nil && matched {
			return true
		}
	}
	return false
}

func hasIncludedDescendant(rel string, values []string) bool {
	for _, value := range values {
		if strings.HasPrefix(value, rel+"/") {
			return true
		}
	}
	return false
}

func safeJoin(root, rel string) (string, error) {
	if rel == "" || filepath.IsAbs(rel) || strings.Contains(rel, "\\") {
		return "", fmt.Errorf("path %q must be a relative slash-separated path", rel)
	}
	for _, part := range strings.Split(rel, "/") {
		if part == ".." {
			return "", fmt.Errorf("path %q contains parent traversal", rel)
		}
	}
	return filepath.Join(root, filepath.FromSlash(rel)), nil
}
