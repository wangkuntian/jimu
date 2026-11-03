// Package frameworkmanifest 将 Jimu 框架内部的能力事实导出成生成器 manifest。
// 这是生成器中唯一允许直接读取能力、profile、contract 和资产注册表的适配层。
package frameworkmanifest

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"

	"jimu/internal/assembly"
	"jimu/internal/capabilities/catalog"
	"jimu/internal/capability"
	"jimu/internal/contract"
	"jimu/internal/profiles"
	"jimu/tools/generator/manifest"

	"github.com/gin-gonic/gin"
)

type Request struct {
	Root      string
	Template  string
	Profile   string
	With      string
	Shape     string
	Module    string
	Framework *manifest.Framework
	// Drivers 保留既有项目的驱动选择；未列出的能力由默认驱动规则补齐。
	Drivers map[string][]string
}

type selection struct {
	shape      string
	template   string
	profile    string
	support    string
	requested  []string
	autoAdded  []string
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

// Resolution 是 `jimu new --dry-run` 与后续 manifest composition（P0-3）共用的解析摘要。
type Resolution struct {
	Requested       []string            `json:"requested"`
	Resolved        []string            `json:"resolved"`
	AutoAdded       []string            `json:"auto_added"`
	Drivers         map[string][]string `json:"drivers"`
	RuntimeServices []string            `json:"runtime_services"`
	Migrations      []string            `json:"migrations"`
	Validity        string              `json:"validity"`
	Support         string              `json:"support"`
}

// Export 根据框架当前事实生成一份可校验、可复现的 manifest。
func Export(req Request) (manifest.Document, error) {
	doc, _, err := ExportDetailed(req)
	return doc, err
}

// ExportDetailed 与 Export 相同，但额外返回本次解析的 requested/resolved/auto-added 等摘要。
func ExportDetailed(req Request) (manifest.Document, Resolution, error) {
	root, err := filepath.Abs(req.Root)
	if err != nil {
		return manifest.Document{}, Resolution{}, fmt.Errorf("resolve framework root: %w", err)
	}
	if req.Module == "" {
		return manifest.Document{}, Resolution{}, fmt.Errorf("module is required")
	}
	if req.Template == "" && req.Profile == "" && req.With == "" {
		req.Template = "basic-api"
	}
	if req.Template != "" && (req.Profile != "" || req.With != "") {
		return manifest.Document{}, Resolution{}, fmt.Errorf("--template must not be combined with --profile or --with")
	}
	if req.Template != "" && req.Shape != "" {
		return manifest.Document{}, Resolution{}, fmt.Errorf("--shape is only valid with --with")
	}
	var doc manifest.Document
	var selected selection
	if req.Template == "auth-api" {
		doc, err = authAPIDocument(root, req.Module, req.Framework)
	} else if req.Template != "" {
		doc, err = basicAPIDocument(root, req.Module, req.Template, req.Framework)
	} else {
		selected, err = resolveSelection(root, req)
		if err == nil {
			doc, err = buildDocument(root, req.Module, selected, req.Framework)
		}
	}
	if err != nil {
		return manifest.Document{}, Resolution{}, err
	}
	digest, err := manifest.Digest(doc)
	if err != nil {
		return manifest.Document{}, Resolution{}, err
	}
	doc.Digest = digest
	return doc, resolutionFor(doc, selected), nil
}

func resolutionFor(doc manifest.Document, selected selection) Resolution {
	requested := slices.Clone(doc.Selection.Capabilities)
	if selected.requested != nil {
		requested = slices.Clone(selected.requested)
	}
	support := doc.Selection.Support
	if doc.Composition != nil {
		support = doc.Composition.Support
	}
	services := map[string]bool{}
	for _, capability := range doc.Capabilities {
		for _, service := range capability.RuntimeServices {
			services[service] = true
		}
	}
	serviceNames := make([]string, 0, len(services))
	for service := range services {
		serviceNames = append(serviceNames, service)
	}
	slices.Sort(serviceNames)
	return Resolution{
		Requested:       requested,
		Resolved:        slices.Clone(doc.Selection.Capabilities),
		AutoAdded:       slices.Clone(selected.autoAdded),
		Drivers:         cloneDrivers(doc.Selection.Drivers),
		RuntimeServices: serviceNames,
		Migrations:      slices.Clone(doc.Report.Migrations),
		Validity:        string(contract.ValidityValid),
		Support:         support,
	}
}

// ExportForAdd 在原有声明集上增加一个能力。硬依赖必须已经存在，避免 add 静默改变用户选择。
func ExportForAdd(root string, previous manifest.Document, name string, force bool, framework *manifest.Framework) (manifest.Document, error) {
	if err := manifest.Validate(previous); err != nil {
		return manifest.Document{}, fmt.Errorf("validate previous manifest: %w", err)
	}
	if previous.Selection.Template != "" {
		return manifest.Document{}, fmt.Errorf(
			"capability add does not extend official template projects (%s); regenerate with --with to use an experimental composition",
			previous.Selection.Template,
		)
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
	all, _ := allDescriptors()
	// add 的产品规则是不自动补依赖：新增能力的硬依赖必须已在现有装配集中。
	for _, descriptor := range all {
		if descriptor.Name != name {
			continue
		}
		for _, required := range descriptor.Requires {
			if !contains(selected, required) {
				return manifest.Document{}, fmt.Errorf("capability %q requires %q; add it first (add the dependency first)", name, required)
			}
		}
	}
	composed := capability.Compose(all, capability.ComposeRequest{
		Capabilities:   selected,
		Drivers:        previous.Composition.Drivers,
		DefaultDrivers: true,
		// 已生成项目的装配集必须是闭包；严格模式阻止 resolver 自动补依赖。
		RequireResolvedSelection: true,
		Support:                  contract.SupportExperimental,
	})
	if err := composed.Err(); err != nil {
		return manifest.Document{}, fmt.Errorf("capability %q cannot be added: %w", name, err)
	}
	// Rebuild from the requested baseline so capabilities auto-added by an
	// earlier resolution stay auto-added instead of becoming requested.
	requested := slices.Clone(previous.Composition.Requested)
	if !contains(requested, name) {
		requested = append(requested, name)
	}
	with := strings.Join(requested, ",")
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
		return Export(Request{Root: root, Profile: profile, Module: module, Framework: framework})
	}
	return Export(Request{
		Root: root, Profile: profile, With: with, Shape: previous.Selection.Shape, Module: module, Drivers: composed.Drivers,
		Framework: framework,
	})
}

func buildDocument(root, module string, selected selection, frameworkOverride *manifest.Framework) (manifest.Document, error) {
	pruneReason := "test-import"
	if catalogCoversAll(selected) {
		pruneReason = "catalog-complete"
	}
	framework, err := frameworkProvenance(root, frameworkOverride)
	if err != nil {
		return manifest.Document{}, err
	}
	doc := manifest.Document{
		SchemaVersion: manifest.CurrentSchemaVersion,
		Framework:     framework,
		Selection: manifest.Selection{
			Shape: selected.shape, Template: selected.template, TemplateVersion: templateVersion(selected.template),
			Profile: selected.profile, Support: selected.support,
			Capabilities: slices.Clone(selected.declared), Drivers: cloneDrivers(selected.drivers),
		},
		Composition:    compositionFor(selected),
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

func compositionFor(selected selection) *manifest.Composition {
	requested := selected.requested
	if requested == nil {
		requested = selected.declared
	}
	return &manifest.Composition{
		ContractVersion: manifest.CompositionContractVersion,
		Requested:       sortedStrings(requested),
		Resolved:        sortedStrings(selected.declared),
		AutoAdded:       sortedStrings(selected.autoAdded),
		Drivers:         cloneDrivers(selected.drivers),
		Validity:        manifest.CompositionValidityValid,
		Support:         compositionSupport(selected.support),
	}
}

func compositionSupport(legacy string) string {
	switch legacy {
	case "official":
		return manifest.CompositionSupportCertified
	case "validated":
		return manifest.CompositionSupportValidated
	default:
		return manifest.CompositionSupportExperimental
	}
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

// frameworkProvenance resolves Git metadata through the CLI so normal clones and linked worktrees behave identically.
func frameworkProvenance(root string, override *manifest.Framework) (manifest.Framework, error) {
	if override != nil {
		return *override, nil
	}
	commit, err := gitOutput(root, "rev-parse", "--verify", "HEAD^{commit}")
	if err != nil {
		return manifest.Framework{}, fmt.Errorf("resolve framework commit: %w", err)
	}
	if commit == "" {
		return manifest.Framework{}, fmt.Errorf("resolve framework commit: git returned an empty commit")
	}
	status, err := gitOutput(root, "status", "--porcelain=v1", "--untracked-files=normal")
	if err != nil {
		return manifest.Framework{}, fmt.Errorf("read framework worktree status: %w", err)
	}
	dirty := strings.TrimSpace(status) != "" || profiles.Version == "dev" || strings.Contains(profiles.Version, "-dev")
	return manifest.Framework{
		Module: "jimu", Version: profiles.Version, Commit: commit, Dirty: dirty,
	}, nil
}

func gitOutput(root string, args ...string) (string, error) {
	commandArgs := append([]string{"-C", root}, args...)
	output, err := exec.CommandContext(context.Background(), "git", commandArgs...).CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git %s: %w: %s", strings.Join(commandArgs, " "), err, strings.TrimSpace(string(output)))
	}
	return strings.TrimSpace(string(output)), nil
}

func templateVersion(template string) string {
	if template == "" {
		return ""
	}
	return manifest.TemplateContractVersion
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
			Name:                    descriptor.Name,
			Requires:                slices.Clone(descriptor.Requires),
			SoftRequires:            slices.Clone(descriptor.SoftRequires),
			MigrationOnly:           contains(selected.migration, descriptor.Name),
			DomainOnly:              contains(selected.domain, descriptor.Name),
			Owns:                    slices.Clone(descriptor.Owns),
			Mount:                   string(descriptor.Normalized()),
			Drivers:                 slices.Clone(descriptor.Drivers),
			Assets:                  slices.Clone(descriptor.Assets),
			ProtectedHTTPMiddleware: string(descriptor.ProtectedHTTPMiddleware),
			MigrationSchemaDeps:     append([]string{}, descriptor.MigrationSchemaDeps...),
			RuntimeServices:         runtimeServiceNames(effectiveRuntimeServices(descriptor, selected.drivers[descriptor.Name])),
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

func runtimeServiceNames(services []contract.RuntimeService) []string {
	out := make([]string, len(services))
	for i, service := range services {
		out[i] = string(service)
	}
	return out
}

// effectiveRuntimeServices 合并能力自身与所选驱动规格声明的运行时服务。
// 生成项目的 preflight 与 jimu check 都读取这份派生事实。
func effectiveRuntimeServices(descriptor contract.Descriptor, drivers []string) []contract.RuntimeService {
	seen := make(map[contract.RuntimeService]bool, len(descriptor.RuntimeServices))
	var out []contract.RuntimeService
	appendService := func(service contract.RuntimeService) {
		if seen[service] {
			return
		}
		seen[service] = true
		out = append(out, service)
	}
	for _, service := range descriptor.RuntimeServices {
		appendService(service)
	}
	for _, driver := range drivers {
		for _, spec := range descriptor.DriverSpecs {
			if spec.Name != driver {
				continue
			}
			for _, service := range spec.RuntimeServices {
				appendService(service)
			}
		}
	}
	slices.Sort(out)
	return out
}

func reportSpec(root string, selected selection, routes int) manifest.ReportSpec {
	spec := manifest.ReportSpec{Name: selected.shape, Capabilities: assemblyNames(selected.assemblies), Routes: routes}
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

func assemblyNames(values []assembly.Capability) []string {
	names := make([]string, 0, len(values))
	for _, value := range values {
		names = append(names, value.Descriptor.Name)
	}
	return names
}

func routeCount(root string, caps []assembly.Capability) (int, error) {
	gin.SetMode(gin.ReleaseMode)
	result, err := assembly.ProbeAssemblyAt(root, assembly.Assembly{Name: "manifest", Capabilities: caps}, nil)
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
