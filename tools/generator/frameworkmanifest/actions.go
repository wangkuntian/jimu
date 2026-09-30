package frameworkmanifest

import (
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strconv"
	"strings"

	"jimu/internal/capabilities/catalog"
	"jimu/tools/generator/manifest"
)

var kernelDirs = []string{
	"internal/kernel",
	"internal/app",
	"internal/assembly",
	"internal/capability",
	"internal/contract",
	"internal/config",
	"internal/shared",
	"internal/e2e",
	"conf",
	"cmd/server",
	"cmd/cli",
}

var kernelFiles = []string{
	"internal/profiles/profiles.go",
	"go.mod",
	"go.sum",
	".golangci.yml",
	".gitignore",
}

var copiedTools = []string{
	"tools/internal/profileoverlay",
	"tools/internal/profileassets",
	"tools/internal/heavydeps",
	"tools/internal/projectmetrics",
	"tools/profileoverlay",
	"tools/profileassets",
	"tools/checkcapabilities",
	"tools/composereport",
	"tools/logcheck",
}

var kernelExcludes = []string{
	"cmd/cli/main.go",
	"cmd/cli/new.go",
	"cmd/cli/new_test.go",
	"cmd/cli/capability.go",
	"cmd/cli/capability_test.go",
	"cmd/cli/activecaps_test.go",
}

func copyActions(root string, selected selection) []manifest.CopyAction {
	actions := make([]manifest.CopyAction, 0, len(kernelDirs)+len(kernelFiles)+len(selected.copy)+len(copiedTools)+len(selected.migration)+len(selected.domain))
	for _, source := range kernelDirs {
		actions = append(actions, manifest.CopyAction{Source: source, Destination: source, Include: []string{}, Exclude: kernelExcludesFor(source)})
	}
	for _, source := range kernelFiles {
		actions = append(actions, manifest.CopyAction{Source: source, Destination: source, Include: []string{}, Exclude: []string{}})
	}
	descriptors, _ := allDescriptors()
	byName := descriptorByName(descriptors)
	for _, name := range selected.copy {
		if contains(selected.migration, name) || contains(selected.domain, name) {
			continue
		}
		exclude := []string{}
		if descriptor, ok := byName[name]; ok {
			chosen := selected.drivers[name]
			for _, driver := range descriptor.Drivers {
				if !contains(chosen, driver) {
					exclude = append(exclude, driver)
				}
			}
		}
		actions = append(actions, manifest.CopyAction{
			Source:      filepath.ToSlash(filepath.Join("internal/capabilities", name)),
			Destination: filepath.ToSlash(filepath.Join("internal/capabilities", name)),
			Include:     []string{},
			Exclude:     sortedStrings(exclude),
		})
	}
	for _, name := range selected.migration {
		base := filepath.ToSlash(filepath.Join("internal/capabilities", name))
		actions = append(actions, manifest.CopyAction{Source: base + "/migrations", Destination: base + "/migrations", Include: []string{}, Exclude: []string{}})
		if directoryExists(root, filepath.FromSlash(base+"/domain")) {
			actions = append(actions, manifest.CopyAction{Source: base + "/domain", Destination: base + "/domain", Include: []string{}, Exclude: []string{}})
		}
	}
	for _, name := range selected.domain {
		base := filepath.ToSlash(filepath.Join("internal/capabilities", name, "domain"))
		actions = append(actions, manifest.CopyAction{Source: base, Destination: base, Include: []string{}, Exclude: []string{}})
	}
	for _, source := range copiedTools {
		actions = append(actions, manifest.CopyAction{Source: source, Destination: source, Include: []string{}, Exclude: []string{"*_test.go"}})
	}
	return actions
}

func kernelExcludesFor(root string) []string {
	var out []string
	for _, value := range kernelExcludes {
		if value == root || strings.HasPrefix(value, root+"/") {
			out = append(out, strings.TrimPrefix(value, root+"/"))
		}
	}
	return sortedStrings(out)
}

func templateActions(root string, selected selection) []manifest.TemplateAction {
	shapeData := map[string]any{
		"Shape":        selected.shape,
		"Capabilities": shapeCapabilities(selected),
	}
	buildData := map[string]any{
		"Shape":      selected.shape,
		"HasAPIdocs": contains(selected.copy, "apidocs"),
		"Expected":   strings.Join(selected.roots, " "),
	}
	catalogData := catalogTemplateData(selected)
	cliData := map[string]any{"CLIImports": cliImports(root, selected)}
	actions := []manifest.TemplateAction{
		{Source: "tools/generator/templates/project/Makefile.tmpl", Destination: "Makefile", Kind: "build", Data: buildData},
		{Source: "tools/generator/templates/project/Dockerfile.tmpl", Destination: "Dockerfile", Kind: "build", Data: buildData},
		{Source: "tools/generator/templates/project/check_profiles.sh.tmpl", Destination: "scripts/check_profiles.sh", Kind: "build", Data: buildData},
		{Source: "tools/generator/templates/project/registry.go.tmpl", Destination: "internal/profiles/registry/registry.go", Kind: "shape", Data: shapeData},
		{Source: "tools/generator/templates/project/assembly.go.tmpl", Destination: filepath.ToSlash(filepath.Join("internal/profiles", selected.shape, "assembly.go")), Kind: "shape", Data: shapeData},
		{Source: "tools/generator/templates/project/drivers.go.tmpl", Destination: filepath.ToSlash(filepath.Join("internal/profiles", selected.shape, "drivers.go")), Kind: "shape", Data: shapeData},
		{Source: "tools/generator/templates/project/active.go.tmpl", Destination: "internal/profiles/active/assembly.go", Kind: "shape", Data: map[string]any{"Shape": selected.shape}},
		{Source: "tools/generator/templates/project/catalog.go.tmpl", Destination: "internal/capabilities/catalog/catalog.go", Kind: "catalog", Data: catalogData},
		{Source: "tools/generator/templates/project/catalog_migration.go.tmpl", Destination: "internal/capabilities/catalog/migration.go", Kind: "catalog", Data: catalogData},
		{Source: "cmd/cli/main.go", Destination: "cmd/cli/main.go", Kind: "cli", Data: cliData},
	}
	return actions
}

func shapeCapabilities(selected selection) []map[string]any {
	capabilities := make([]map[string]any, 0, len(selected.declared))
	for _, name := range selected.declared {
		drivers := slices.Clone(selected.drivers[name])
		capabilities = append(capabilities, map[string]any{
			"Alias":      name + "module",
			"Path":       "jimu/internal/capabilities/" + name,
			"Ungated":    selected.ungated[name],
			"Drivers":    drivers,
			"HasDrivers": len(drivers) > 0,
			"Capability": name,
		})
	}
	return capabilities
}

func catalogTemplateData(selected selection) map[string]any {
	selectedNames := make(map[string]bool, len(selected.declared)+len(selected.migration))
	for _, name := range selected.declared {
		selectedNames[name] = true
	}
	for _, name := range selected.migration {
		selectedNames[name] = true
	}
	all, _ := allDescriptors()
	inCatalog := make(map[string]bool, len(catalog.All()))
	for _, descriptor := range catalog.All() {
		inCatalog[descriptor.Name] = true
	}
	entries := make([]map[string]any, 0, len(selectedNames))
	for _, descriptor := range all {
		if !selectedNames[descriptor.Name] || !inCatalog[descriptor.Name] {
			continue
		}
		entries = append(entries, catalogEntry(descriptor.Name))
	}
	for _, name := range selected.declared {
		if !inCatalog[name] {
			entries = append(entries, catalogEntry(name))
		}
	}
	migrationOnly := make([]map[string]any, 0, len(selected.migration))
	for _, name := range selected.migration {
		migrationOnly = append(migrationOnly, catalogEntry(name))
	}
	deps := make([]map[string]any, 0)
	for _, entry := range entries {
		name, ok := entry["Name"].(string)
		if !ok {
			continue
		}
		values := catalog.MigrationSchemaDeps[name]
		if len(values) == 0 {
			continue
		}
		quoted := make([]string, 0, len(values))
		for _, value := range values {
			quoted = append(quoted, strconv.Quote(value))
		}
		deps = append(deps, map[string]any{
			"From": name,
			"To":   "{" + strings.Join(quoted, ", ") + "}",
		})
	}
	return map[string]any{
		"Entries":       entries,
		"MigrationOnly": migrationOnly,
		"KnownNames":    slices.Clone(selected.known),
		"SchemaDeps":    deps,
	}
}

func catalogEntry(name string) map[string]any {
	return map[string]any{
		"Name":  name,
		"Alias": name + "module",
		"Path":  "jimu/internal/capabilities/" + name,
	}
}

func cliImports(root string, selected selection) []map[string]any {
	imports := make([]map[string]any, 0)
	for _, name := range selected.declared {
		info, err := os.Stat(filepath.Join(root, "internal", "capabilities", name, "cli"))
		if err != nil || !info.IsDir() {
			continue
		}
		imports = append(imports, map[string]any{
			"Alias": name + "cli",
			"Path":  "jimu/internal/capabilities/" + name + "/cli",
		})
	}
	return imports
}

func mergeActions() []manifest.MergeAction {
	return []manifest.MergeAction{
		{Source: "configs/app.yaml", Destination: "configs/app.yaml", Strategy: "sections", Sections: []string{}},
		{Source: "configs/app.prod.yaml", Destination: "configs/app.prod.yaml", Strategy: "sections", Sections: []string{}},
	}
}

func rewriteActions(module string) []manifest.RewriteAction {
	return []manifest.RewriteAction{
		{Kind: "module", From: "jimu", To: module, Files: []string{}},
		{Kind: "file-patches", From: "{{module}}", To: module, Files: []string{"tools/checkcapabilities/drivers.go", "tools/composereport/main.go", "tools/logcheck/main.go", "tools/internal/profileoverlay/profileoverlay.go"}},
	}
}

func generatedFiles(root string, selected selection) []string {
	seen := map[string]bool{}
	for _, action := range copyActions(root, selected) {
		collectFiles(root, action.Source, action.Exclude, seen)
	}
	for _, action := range templateActions(root, selected) {
		seen[action.Destination] = true
	}
	for _, action := range mergeActions() {
		seen[action.Destination] = true
	}
	for _, action := range assetActions(root, selected) {
		collectFiles(root, action.Source, nil, seen)
	}
	return sortedStrings(mapsKeys(seen))
}

func collectFiles(root, source string, excluded []string, seen map[string]bool) {
	base := filepath.Join(root, filepath.FromSlash(source))
	info, err := os.Stat(base)
	if err != nil {
		return
	}
	if !info.IsDir() {
		seen[source] = true
		return
	}
	_ = filepath.WalkDir(base, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if rel != source && excludedPath(rel, source, excluded) {
			if entry.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		if !entry.IsDir() && entry.Type()&fs.ModeSymlink == 0 {
			seen[rel] = true
		}
		return nil
	})
}

func excludedPath(rel, source string, excluded []string) bool {
	relative := strings.TrimPrefix(strings.TrimPrefix(rel, source), "/")
	for _, value := range excluded {
		if value == "*_test.go" && strings.HasSuffix(relative, "_test.go") {
			return true
		}
		if relative == value || strings.HasPrefix(relative, value+"/") {
			return true
		}
	}
	return false
}

func directoryExists(root, rel string) bool {
	info, err := os.Stat(filepath.Join(root, rel))
	return err == nil && info.IsDir()
}

func mapsKeys(values map[string]bool) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	return keys
}

func sortedStrings(values []string) []string {
	out := slices.Clone(values)
	sort.Strings(out)
	return out
}
