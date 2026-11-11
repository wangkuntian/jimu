package frameworkmanifest

import (
	"maps"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	"jimu/internal/contract"
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
				if canonical == "deploy/backup" && !includeBackup(selected) {
					continue
				}
				paths[canonical] = true
			}
		}
	}
	if !includeObservability(selected) {
		for _, value := range profileassets.CoreGroups()["observability"] {
			delete(paths, profileassets.Canonical(value))
		}
	}
	return slices.Sorted(maps.Keys(paths))
}

func assetActions(_ string, selected selection) []manifest.AssetAction {
	paths := selectedAssets(selected)
	actions := make([]manifest.AssetAction, 0, len(paths))
	for _, value := range paths {
		exclude := []string{}
		switch value {
		case "deploy/helm":
			exclude = append(exclude, "values.yaml")
			if !includeObservability(selected) {
				exclude = append(exclude, "templates/openobserve.yaml", "templates/otel-collector.yaml")
			}
			if !includeBackup(selected) {
				exclude = append(exclude, "templates/backup-cronjob.yaml")
				exclude = append(exclude, "templates/secret.yaml")
			}
		case "deploy/k8s":
			exclude = append(exclude, "config-files.yaml")
			if !includeObservability(selected) {
				exclude = append(exclude, "openobserve.yaml", "otel-collector.yaml")
			}
			if !includeBackup(selected) {
				exclude = append(exclude, "backup-cronjob.yaml")
				exclude = append(exclude, "secret.yaml")
			}
		}
		actions = append(actions, manifest.AssetAction{Source: value, Destination: value, Include: []string{}, Exclude: exclude})
	}
	return actions
}

func selectedAssetFiles(root string, selected selection) []string {
	seen := map[string]bool{}
	for _, action := range assetActions(root, selected) {
		collectFiles(root, action.Source, action.Exclude, seen)
	}
	roots := profileassets.AssetRoots()
	out := make([]string, 0, len(seen))
	for path := range seen {
		for _, root := range roots {
			root = profileassets.Canonical(root)
			if path == root || strings.HasPrefix(path, root+"/") {
				out = append(out, path)
				break
			}
		}
	}
	slices.Sort(out)
	return out
}

func includeBackup(selected selection) bool {
	return selectionRuntimeServices(selected)[contract.RuntimeServiceMySQL]
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
