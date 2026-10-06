package workspace

import (
	"fmt"
	"os"
	"path/filepath"

	"jimu/tools/generator/manifest"
)

func LoadProjectManifest(root string) (manifest.Document, error) {
	path := filepath.Join(root, manifestRel)
	doc, err := manifest.Load(path)
	if err != nil {
		if os.IsNotExist(err) {
			return manifest.Document{}, fmt.Errorf("%s is not a jimu project: missing %s (run jimu new first)", root, manifestRel)
		}
		return manifest.Document{}, err
	}
	return doc, nil
}
