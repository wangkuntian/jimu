package frameworkmanifest

import (
	"maps"
	"path/filepath"
	"slices"
	"sort"

	"jimu/tools/generator/manifest"
	"jimu/tools/internal/profileassets"
)

func selectedAssets(selected selection) []string {
	declared := profileassets.Declared()
	paths := map[string]bool{}
	for _, name := range selected.declared {
		for _, value := range declared["cap:"+name] {
			if canonical := profileassets.Canonical(value); canonical != "" {
				paths[canonical] = true
			}
		}
	}
	for _, group := range profileassets.CoreGroups() {
		for _, value := range group {
			if canonical := profileassets.Canonical(value); canonical != "" {
				paths[canonical] = true
			}
		}
	}
	return slices.Sorted(maps.Keys(paths))
}

func assetActions(_ string, selected selection) []manifest.AssetAction {
	paths := selectedAssets(selected)
	actions := make([]manifest.AssetAction, 0, len(paths))
	for _, value := range paths {
		actions = append(actions, manifest.AssetAction{Source: value, Destination: value, Include: []string{}, Exclude: []string{}})
	}
	return actions
}

func assetFiles(root string, selected selection) []string {
	var files []string
	for _, action := range assetActions(root, selected) {
		files = append(files, action.Source)
	}
	sort.Strings(files)
	return files
}

func assetRoot(value string) string {
	return filepath.ToSlash(value)
}
