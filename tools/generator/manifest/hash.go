package manifest

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// HashFiles returns the content hash recorded for every generated file.
func HashFiles(root string, files []string) (map[string]string, error) {
	hashes := make(map[string]string, len(files))
	for _, rel := range files {
		if err := validateActionPath("generated_files", rel); err != nil {
			return nil, err
		}
		path := filepath.Join(root, filepath.FromSlash(rel))
		file, err := os.Open(path)
		if err != nil {
			return nil, fmt.Errorf("open generated file %s: %w", rel, err)
		}
		hash := sha256.New()
		_, copyErr := io.Copy(hash, file)
		closeErr := file.Close()
		if copyErr != nil {
			return nil, fmt.Errorf("hash generated file %s: %w", rel, copyErr)
		}
		if closeErr != nil {
			return nil, fmt.Errorf("close generated file %s: %w", rel, closeErr)
		}
		hashes[rel] = "sha256:" + hex.EncodeToString(hash.Sum(nil))
	}
	return hashes, nil
}
