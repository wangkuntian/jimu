package workspace

import (
	"fmt"
	"os"
	"path/filepath"

	"jimu/tools/generator/manifest"
	"jimu/tools/generator/render"
)

func mergeConfigs(projectRoot, staging string, actions []manifest.MergeAction) error {
	for _, action := range actions {
		if action.Strategy != "sections" {
			continue
		}
		destination := filepath.Join(projectRoot, filepath.FromSlash(action.Destination))
		generated := filepath.Join(staging, filepath.FromSlash(action.Destination))
		generatedContent, err := os.ReadFile(generated)
		if err != nil {
			return fmt.Errorf("read staged config %s: %w", action.Destination, err)
		}
		existingContent, err := os.ReadFile(destination)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return fmt.Errorf("read existing config %s: %w", action.Destination, err)
		}
		merged := render.MergeSections(existingContent, generatedContent)
		if err := os.WriteFile(generated, merged, 0o644); err != nil {
			return fmt.Errorf("write staged config %s: %w", action.Destination, err)
		}
	}
	return nil
}
