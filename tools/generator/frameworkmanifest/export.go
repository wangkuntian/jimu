// Package frameworkmanifest 将 Jimu 框架内部的能力事实导出成生成器 manifest。
// 这是生成器中唯一允许直接读取能力、profile、contract 和资产注册表的适配层。
package frameworkmanifest

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"

	"jimu/internal/assembly"
	"jimu/internal/capabilities/catalog"
	"jimu/internal/contract"
	"jimu/internal/profiles"
	"jimu/internal/profiles/full"
	"jimu/tools/generator/manifest"

	"github.com/gin-gonic/gin"
)

type Request struct {
	Root    string
	Profile string
	With    string
	Shape   string
	Module  string
}

type selection struct {
	shape      string
	profile    string
	declared   []string
	known      []string
	copy       []string
	roots      []string
	migration  []string
	domain     []string
	drivers    map[string][]string
	ungated    map[string]bool
	assemblies []assembly.Capability
}

var probeMu sync.Mutex

// Export 根据框架当前事实生成一份可校验、可复现的 manifest。
func Export(req Request) (manifest.Document, error) {
	root, err := filepath.Abs(req.Root)
	if err != nil {
		return manifest.Document{}, fmt.Errorf("resolve framework root: %w", err)
	}
	if req.Module == "" {
		return manifest.Document{}, fmt.Errorf("module is required")
	}
	selected, err := resolveSelection(root, req)
	if err != nil {
		return manifest.Document{}, err
	}
	doc, err := buildDocument(root, req.Module, selected)
	if err != nil {
		return manifest.Document{}, err
	}
	digest, err := manifest.Digest(doc)
	if err != nil {
		return manifest.Document{}, err
	}
	doc.Digest = digest
	return doc, nil
}

// ExportForAdd 在原有声明集上增加一个能力。硬依赖必须已经存在，避免 add 静默改变用户选择。
func ExportForAdd(root string, previous manifest.Document, name string, force bool) (manifest.Document, error) {
	if err := manifest.Validate(previous); err != nil {
		return manifest.Document{}, fmt.Errorf("validate previous manifest: %w", err)
	}
	for _, capability := range previous.Capabilities {
		if capability.Name != name || !capability.MigrationOnly || force {
			continue
		}
		return manifest.Document{}, fmt.Errorf("capability %q is already present only as a migration carry; pass --force to promote it to a full declared capability", name)
	}
	if contains(previous.Selection.Capabilities, name) && !force {
		return manifest.Document{}, fmt.Errorf("capability %q is already present as a declared capability (already selected)", name)
	}
	selected := slices.Clone(previous.Selection.Capabilities)
	if !contains(selected, name) {
		selected = append(selected, name)
	}
	for _, descriptor := range full.Assembly().Capabilities {
		if descriptor.Descriptor.Name != name {
			continue
		}
		for _, required := range descriptor.Descriptor.Requires {
			if !contains(selected, required) {
				return manifest.Document{}, fmt.Errorf("capability %q requires %q; add it first (add the dependency first)", name, required)
			}
		}
		break
	}
	with := strings.Join(selected, ",")
	module := previous.Framework.Module
	for _, rewrite := range previous.Rewrites {
		if rewrite.Kind == "module" && rewrite.To != "" {
			module = rewrite.To
			break
		}
	}
	profile := ""
	if force && contains(previous.Selection.Capabilities, name) {
		// Rebuilding an existing profile selection must preserve its profile identity;
		// otherwise an idempotent --force run would rewrite only metadata.
		profile = previous.Selection.Profile
	}
	if profile != "" {
		return Export(Request{Root: root, Profile: profile, Module: module})
	}
	return Export(Request{Root: root, Profile: profile, With: with, Shape: previous.Selection.Shape, Module: module})
}

func buildDocument(root, module string, selected selection) (manifest.Document, error) {
	pruneReason := "test-import"
	if catalogCoversAll(selected) {
		pruneReason = "catalog-complete"
	}
	doc := manifest.Document{
		SchemaVersion:  manifest.CurrentSchemaVersion,
		Framework:      manifest.Framework{Module: "jimu", Version: profiles.Version, Commit: frameworkCommit(root)},
		Selection:      manifest.Selection{Shape: selected.shape, Profile: selected.profile, Capabilities: slices.Clone(selected.declared), Drivers: cloneDrivers(selected.drivers)},
		Capabilities:   capabilityFacts(root, selected),
		Copy:           copyActions(root, selected),
		Templates:      templateActions(root, selected),
		Merges:         mergeActions(root, selected),
		Rewrites:       rewriteActions(module),
		Assets:         assetActions(root, selected),
		Prune:          []manifest.PruneRule{{Kind: "test-import", Path: "**/*_test.go", Reason: pruneReason}},
		GeneratedFiles: generatedFiles(root, selected),
	}
	routes, err := routeCount(root, selected.assemblies)
	if err != nil {
		return manifest.Document{}, err
	}
	doc.Report = reportSpec(root, selected, routes)
	if err := manifest.Validate(doc); err != nil {
		return manifest.Document{}, err
	}
	return doc, nil
}

func catalogCoversAll(selected selection) bool {
	chosen := make(map[string]bool, len(selected.declared)+len(selected.migration))
	for _, name := range selected.declared {
		chosen[name] = true
	}
	for _, name := range selected.migration {
		chosen[name] = true
	}
	for _, descriptor := range catalog.All() {
		if !chosen[descriptor.Name] {
			return false
		}
	}
	return true
}

func frameworkCommit(root string) string {
	head, err := os.ReadFile(filepath.Join(root, ".git", "HEAD"))
	if err != nil {
		return ""
	}
	value := strings.TrimSpace(string(head))
	if strings.HasPrefix(value, "ref: ") {
		ref, err := os.ReadFile(filepath.Join(root, ".git", filepath.FromSlash(strings.TrimPrefix(value, "ref: "))))
		if err != nil {
			return ""
		}
		return strings.TrimSpace(string(ref))
	}
	return value
}

func capabilityFacts(root string, selected selection) []manifest.Capability {
	all, _ := allDescriptors()
	selectedNames := make(map[string]bool, len(selected.copy))
	for _, name := range selected.copy {
		selectedNames[name] = true
	}
	for _, name := range selected.declared {
		selectedNames[name] = true
	}
	var out []manifest.Capability
	for _, descriptor := range all {
		if !selectedNames[descriptor.Name] {
			continue
		}
		fact := manifest.Capability{
			Name:          descriptor.Name,
			Requires:      slices.Clone(descriptor.Requires),
			SoftRequires:  slices.Clone(descriptor.SoftRequires),
			MigrationOnly: contains(selected.migration, descriptor.Name),
			DomainOnly:    contains(selected.domain, descriptor.Name),
			Owns:          slices.Clone(descriptor.Owns),
			Mount:         string(descriptor.Normalized()),
			Drivers:       slices.Clone(descriptor.Drivers),
			Assets:        slices.Clone(descriptor.Assets),
		}
		for _, config := range descriptor.Configs {
			fact.Configs = append(fact.Configs, config.Section)
		}
		for _, permission := range descriptor.Permissions {
			fact.Permissions = append(fact.Permissions, manifest.Permission{Name: permission.Name, Resource: permission.Resource, Action: permission.Action})
		}
		fact.Migrations = migrationFiles(descriptor.Name, descriptor.Migrations)
		out = append(out, fact)
	}
	_ = root
	return out
}

func reportSpec(root string, selected selection, routes int) manifest.ReportSpec {
	spec := manifest.ReportSpec{Name: selected.shape, Capabilities: slices.Clone(selected.declared), Routes: routes}
	all, _ := allDescriptors()
	byName := make(map[string]contract.Descriptor, len(all))
	for _, descriptor := range all {
		byName[descriptor.Name] = descriptor
	}
	for _, name := range append(slices.Clone(selected.declared), selected.migration...) {
		descriptor, ok := byName[name]
		if !ok {
			continue
		}
		spec.Migrations = append(spec.Migrations, migrationFiles(name, descriptor.Migrations)...)
		spec.Tables = append(spec.Tables, descriptor.Owns...)
	}
	spec.Migrations = uniqueSorted(spec.Migrations)
	spec.Tables = uniqueSorted(spec.Tables)
	_ = root
	return spec
}

func routeCount(root string, caps []assembly.Capability) (int, error) {
	probeMu.Lock()
	defer probeMu.Unlock()
	workingDirectory, err := os.Getwd()
	if err != nil {
		return 0, fmt.Errorf("read working directory: %w", err)
	}
	if err := os.Chdir(root); err != nil {
		return 0, fmt.Errorf("enter framework root: %w", err)
	}
	defer func() { _ = os.Chdir(workingDirectory) }()
	gin.SetMode(gin.ReleaseMode)
	result, err := assembly.ProbeAssembly(assembly.Assembly{Name: "manifest", Capabilities: caps}, nil)
	if err != nil {
		return 0, fmt.Errorf("probe selected assembly: %w", err)
	}
	return countRoutes(result.Modules), nil
}

func countRoutes(modules []contract.Module) int {
	// Route registration is intentionally kept in the adapter. The core manifest package
	// never imports gin or a framework module.
	router := gin.New()
	for _, module := range modules {
		module.RegisterHTTP(router)
	}
	return len(router.Routes())
}

func cloneDrivers(drivers map[string][]string) map[string][]string {
	out := make(map[string][]string, len(drivers))
	for name, selected := range drivers {
		out[name] = slices.Clone(selected)
	}
	return out
}

func contains(values []string, want string) bool {
	return slices.Contains(values, want)
}
