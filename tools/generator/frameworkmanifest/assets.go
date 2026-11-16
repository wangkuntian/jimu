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
	add := func(value string) {
		if canonical := profileassets.Canonical(value); canonical != "" {
			paths[canonical] = true
		}
	}
	deploy := selected.generation.Deploy
	if HasDeploy(deploy, manifest.DeployCompose) {
		add("deploy/scripts")
	}
	if HasDeploy(deploy, manifest.DeployK8s) {
		add("deploy/k8s")
		add("deploy/scripts")
	}
	if HasDeploy(deploy, manifest.DeployHelm) {
		add("deploy/helm")
		add("deploy/scripts")
	}
	if HasDeploy(deploy, manifest.DeployObservability) {
		add("deploy/observability")
		if HasDeploy(deploy, manifest.DeployK8s) {
			add("deploy/k8s/observability")
		}
		if HasDeploy(deploy, manifest.DeployHelm) {
			add("deploy/helm/templates/observability")
		}
	}
	if includeBackup(selected) && (HasDeploy(deploy, manifest.DeployCompose) || HasDeploy(deploy, manifest.DeployK8s) || HasDeploy(deploy, manifest.DeployHelm)) {
		add("deploy/backup")
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
				exclude = append(exclude, "templates/observability")
			}
			if !includeBackup(selected) {
				exclude = append(exclude, "templates/backup-cronjob.yaml")
				exclude = append(exclude, "templates/secret.yaml")
			}
		case "deploy/k8s":
			exclude = append(exclude, "config-files.yaml")
			if !includeObservability(selected) {
				exclude = append(exclude, "observability")
			}
			if !includeBackup(selected) {
				exclude = append(exclude, "backup-cronjob.yaml")
				exclude = append(exclude, "secret.yaml")
			}
		case "deploy/scripts":
			if !includeAuthSecrets(selected) && !includeBackup(selected) {
				exclude = append(exclude, "init-secrets.sh")
			}
			if !includeBackup(selected) {
				exclude = append(exclude,
					"backup.sh",
					"restore.sh",
					"db_common.sh",
					"install_db_clients.sh",
				)
			}
			if !includeObservability(selected) {
				exclude = append(exclude, "observability.sh")
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
	deploy := selected.generation.Deploy
	if HasDeploy(deploy, manifest.DeployCompose) || HasDeploy(deploy, manifest.DeployK8s) || HasDeploy(deploy, manifest.DeployHelm) {
		seen["deploy/docker/Dockerfile"] = true
	}
	if HasDeploy(deploy, manifest.DeployCompose) {
		seen["deploy/compose/docker-compose.yml"] = true
		if HasDeploy(deploy, manifest.DeployObservability) {
			seen["deploy/compose/compose.observability.yml"] = true
		}
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

// includeAuthSecrets 只在有认证类能力时生成认证 secret；0 能力项目没有消费者。
func includeAuthSecrets(selected selection) bool {
	for _, name := range selected.declared {
		switch name {
		case "auth", "user", "access", "mfa", "passkey", "oauth", "captcha":
			return true
		}
	}
	return false
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
