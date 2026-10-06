package frameworkmanifest

import (
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"sort"
	"strconv"
	"strings"

	"jimu/internal/capabilities/catalog"
	"jimu/internal/config"
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
		exclude := []string{"*_test.go"}
		if source == "tools/composereport" {
			// The framework-only adapter imports frameworkmanifest. Generated projects
			// read their persisted manifest through the generated spec instead.
			exclude = append(exclude, "framework_spec.go")
		}
		actions = append(actions, manifest.CopyAction{Source: source, Destination: source, Include: []string{}, Exclude: exclude})
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
		{Source: "tools/generator/render/templates/project/Makefile.tmpl", Destination: "Makefile", Kind: "build", Data: buildData},
		{Source: "tools/generator/render/templates/project/Dockerfile.tmpl", Destination: "Dockerfile", Kind: "build", Data: buildData},
		{Source: "tools/generator/render/templates/project/check_profiles.sh.tmpl", Destination: "scripts/check_profiles.sh", Kind: "build", Data: buildData},
		{Source: "tools/generator/render/templates/project/registry.go.tmpl", Destination: "internal/profiles/registry/registry.go", Kind: "shape", Data: shapeData},
		{Source: "tools/generator/render/templates/project/assembly.go.tmpl", Destination: filepath.ToSlash(filepath.Join("internal/profiles", selected.shape, "assembly.go")), Kind: "shape", Data: shapeData},
		{Source: "tools/generator/render/templates/project/drivers.go.tmpl", Destination: filepath.ToSlash(filepath.Join("internal/profiles", selected.shape, "drivers.go")), Kind: "shape", Data: shapeData},
		{Source: "tools/generator/render/templates/project/active.go.tmpl", Destination: "internal/profiles/active/assembly.go", Kind: "shape", Data: map[string]any{"Shape": selected.shape}},
		{Source: "tools/generator/render/templates/project/catalog.go.tmpl", Destination: "internal/capabilities/catalog/catalog.go", Kind: "catalog", Data: catalogData},
		{Source: "tools/generator/render/templates/project/catalog_migration.go.tmpl", Destination: "internal/capabilities/catalog/migration.go", Kind: "catalog", Data: catalogData},
		{Source: "tools/generator/render/templates/project/compose_spec.go.tmpl", Destination: "tools/composereport/manifest_spec.go", Kind: "report"},
		{Source: "cmd/cli/main.go", Destination: "cmd/cli/main.go", Kind: "cli", Data: cliData},
	}
	all, _ := allDescriptors()
	byName := descriptorByName(all)
	for _, name := range selected.migration {
		descriptor, ok := byName[name]
		if !ok {
			continue
		}
		actions = append(actions, manifest.TemplateAction{
			Source:      "tools/generator/render/templates/project/migration_module.go.tmpl",
			Destination: filepath.ToSlash(filepath.Join("internal/capabilities", name, "module.go")),
			Kind:        "migration-module",
			Data: map[string]any{
				"Name": descriptor.Name,
				"Owns": slices.Clone(descriptor.Owns),
			},
		})
	}
	return actions
}

func configSections(selected selection) []string {
	keep := map[string]bool{}
	typ := reflect.TypeOf(config.Config{})
	for i := 0; i < typ.NumField(); i++ {
		name, _, _ := strings.Cut(typ.Field(i).Tag.Get("mapstructure"), ",")
		if name != "" && name != "-" {
			keep[name] = true
		}
	}
	all, _ := allDescriptors()
	selectedNames := make(map[string]bool, len(selected.declared))
	for _, name := range selected.declared {
		selectedNames[name] = true
	}
	for _, descriptor := range all {
		if !selectedNames[descriptor.Name] {
			continue
		}
		for _, spec := range descriptor.Configs {
			keep[spec.Section] = true
		}
	}
	for name, sections := range map[string][]string{
		"storage":      {"storage"},
		"retention":    {"retention"},
		"notification": {"email", "sms", "notification"},
	} {
		if !selectedNames[name] {
			continue
		}
		for _, section := range sections {
			keep[section] = true
		}
	}
	return sortedStrings(mapsKeys(keep))
}

func valuesSections(root string, selected selection) []string {
	path := filepath.Join(root, filepath.FromSlash("deploy/helm/values.yaml"))
	content, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	selectedNames := make(map[string]bool, len(selected.declared))
	for _, name := range selected.declared {
		selectedNames[name] = true
	}
	capabilityKeys := map[string]string{"audit": "audit", "storage": "storage", "auth": "auth"}
	var sections []string
	for _, line := range strings.Split(string(content), "\n") {
		if line == "" || strings.HasPrefix(line, " ") || strings.HasPrefix(line, "\t") || strings.HasPrefix(line, "#") {
			continue
		}
		key, _, ok := strings.Cut(line, ":")
		if !ok || key == "" {
			continue
		}
		if capability, ok := capabilityKeys[strings.TrimSpace(key)]; ok && !selectedNames[capability] {
			continue
		}
		sections = append(sections, strings.TrimSpace(key))
	}
	return sections
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

func mergeActions(root string, selected selection) []manifest.MergeAction {
	sections := configSections(selected)
	return []manifest.MergeAction{
		{Source: "configs/app.yaml", Destination: "configs/app.yaml", Strategy: "sections", Sections: slices.Clone(sections)},
		{Source: "configs/app.prod.yaml", Destination: "configs/app.prod.yaml", Strategy: "sections", Sections: slices.Clone(sections)},
		{Source: "deploy/helm/values.yaml", Destination: "deploy/helm/values.yaml", Strategy: "sections", Sections: valuesSections(root, selected)},
	}
}

func rewriteActions(module string) []manifest.RewriteAction {
	return []manifest.RewriteAction{
		{Kind: "module", From: "jimu", To: module, Files: []string{}},
		{Kind: "file-patch", From: `const modulePath = "jimu"`, To: `const modulePath = "` + module + `"`, Files: []string{"tools/checkcapabilities/drivers.go", "tools/composereport/main.go"}},
		{Kind: "file-patch", From: `"jimu/tools/logcheck"`, To: `"` + module + `/tools/logcheck"`, Files: []string{"tools/logcheck/main.go"}},
		{Kind: "file-patch", From: `"jimu/internal/assembly"`, To: `"` + module + `/internal/assembly"`, Files: []string{"tools/internal/profileoverlay/profileoverlay.go"}},
		{Kind: "file-patch", From: `"jimu/internal/profiles/%s"`, To: `"` + module + `/internal/profiles/%s"`, Files: []string{"tools/internal/profileoverlay/profileoverlay.go"}},
		{Kind: "file-patch", From: "// Command composereport 生成各形态（profile）的「编译面」报告。", To: "// Command composereport 生成本项目唯一形态的「编译面」报告。", Files: []string{"tools/composereport/main.go"}},
		{Kind: "file-patch", From: "报告回答「层②（形态选点）究竟改变了什么」", To: "报告回答「本项目编进了什么」", Files: []string{"tools/composereport/main.go"}},
		{Kind: "file-patch", From: "数在各形态间**完全相同**", To: "数不再随形态变化", Files: []string{"tools/composereport/main.go"}},
		{Kind: "file-patch", From: "（`jimu/...`）", To: "（`" + module + "/...`）", Files: []string{"tools/composereport/main.go"}},
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
	for _, action := range mergeActions(root, selected) {
		seen[action.Destination] = true
	}
	for _, action := range assetActions(root, selected) {
		collectFiles(root, action.Source, action.Exclude, seen)
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
		if rel != source && isTransientEntry(entry.Name()) {
			if entry.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
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

func isTransientEntry(name string) bool {
	switch name {
	case ".git", ".DS_Store", ".idea", ".vscode":
		return true
	}
	return strings.HasSuffix(name, ".tmp-") || (strings.HasPrefix(name, ".") && strings.Contains(name, ".tmp-"))
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
