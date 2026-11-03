package generator

import (
	"fmt"
	"path/filepath"
)

func resolveSourceRoot(sourceRoot string) (string, error) {
	if sourceRoot == "" {
		return frameworkRoot()
	}
	root, err := filepath.Abs(sourceRoot)
	if err != nil {
		return "", fmt.Errorf("resolve source root: %w", err)
	}
	return root, nil
}
