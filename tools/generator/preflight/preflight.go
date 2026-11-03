// Package preflight validates generator manifest facts before any target file is written.
package preflight

import (
	"fmt"
	"slices"
	"strings"

	"jimu/internal/capability"
	"jimu/tools/generator/manifest"
)

var runtimeServiceSection = map[string]string{
	"mysql":    "db",
	"redis":    "redis",
	"kafka":    "queue",
	"rabbitmq": "queue",
}

// Validate rejects selections that cannot produce a self-consistent generated project.
func Validate(doc manifest.Document) error {
	return ValidateWithOverlay(doc, nil)
}

// ValidateWithOverlay 与 Validate 相同，但允许用框架当前的 Descriptor 事实补全
// manifest schema 尚未持久化的组合字段（conflicts/provides/consumes/driver specs）。
func ValidateWithOverlay(doc manifest.Document, overlay []capability.Descriptor) error {
	return capability.DiagnosticsError(DiagnoseWithOverlay(doc, overlay))
}

// Diagnose returns every composition diagnostic for a manifest, in stable order.
// It covers the full resolution pipeline from input validation through runtime
// services and middleware. Plan validation is appended by the caller.
func Diagnose(doc manifest.Document) []capability.Diagnostic {
	return DiagnoseWithOverlay(doc, nil)
}

// DiagnoseWithOverlay 在 Diagnose 的基础上叠加框架当前 Descriptor 的组合字段。
func DiagnoseWithOverlay(doc manifest.Document, overlay []capability.Descriptor) []capability.Diagnostic {
	descriptors, migrationFiles, carriers := facts(doc, overlay)
	result := capability.Compose(descriptors, capability.ComposeRequest{
		Capabilities:             doc.Selection.Capabilities,
		Drivers:                  doc.Selection.Drivers,
		DefaultDrivers:           false,
		RequireResolvedSelection: true,
		MigrationCarriers:        carriers,
		CheckMigrationCarriers:   true,
		MigrationFiles:           migrationFiles,
	})
	diagnostics := slices.Clone(result.Diagnostics)
	diagnostics = append(diagnostics, runtimeServiceDiagnostics(result, doc)...)
	diagnostics = append(diagnostics, middlewareDiagnostics(result.Resolved)...)
	capability.SortDiagnostics(diagnostics)
	return diagnostics
}

func facts(doc manifest.Document, overlay []capability.Descriptor) ([]capability.Descriptor, map[string][]string, []string) {
	overlayByName := make(map[string]capability.Descriptor, len(overlay))
	for _, descriptor := range overlay {
		overlayByName[descriptor.Name] = descriptor
	}
	descriptors := make([]capability.Descriptor, 0, len(doc.Capabilities))
	migrationFiles := make(map[string][]string, len(doc.Capabilities))
	var carriers []string
	for _, fact := range doc.Capabilities {
		descriptor := capability.Descriptor{
			Name:                    fact.Name,
			Requires:                slices.Clone(fact.Requires),
			SoftRequires:            slices.Clone(fact.SoftRequires),
			Mount:                   capability.MountPoint(fact.Mount),
			Drivers:                 slices.Clone(fact.Drivers),
			Owns:                    slices.Clone(fact.Owns),
			MigrationSchemaDeps:     slices.Clone(fact.MigrationSchemaDeps),
			RuntimeServices:         runtimeServices(fact.RuntimeServices),
			ProtectedHTTPMiddleware: capability.ProtectedHTTPMiddlewareMode(fact.ProtectedHTTPMiddleware),
		}
		for _, section := range fact.Configs {
			descriptor.Configs = append(descriptor.Configs, capability.ConfigSpec{Section: section})
		}
		if current, ok := overlayByName[descriptor.Name]; ok {
			descriptor.Conflicts = slices.Clone(current.Conflicts)
			descriptor.Provides = slices.Clone(current.Provides)
			descriptor.Consumes = slices.Clone(current.Consumes)
			descriptor.DriverSpecs = slices.Clone(current.DriverSpecs)
		}
		descriptors = append(descriptors, descriptor)
		migrationFiles[fact.Name] = slices.Clone(fact.Migrations)
		if fact.MigrationOnly {
			carriers = append(carriers, fact.Name)
		}
	}
	return descriptors, migrationFiles, carriers
}

func runtimeServices(values []string) []capability.RuntimeService {
	out := make([]capability.RuntimeService, 0, len(values))
	for _, value := range values {
		out = append(out, capability.RuntimeService(value))
	}
	return out
}

func runtimeServiceDiagnostics(result capability.ComposeResult, doc manifest.Document) []capability.Diagnostic {
	var diagnostics []capability.Diagnostic
	inMigrationSet := map[string]bool{}
	for _, descriptor := range result.Resolved {
		inMigrationSet[descriptor.Name] = true
	}
	for _, name := range result.MigrationCarriers {
		inMigrationSet[name] = true
	}
	declaredSections := map[string]bool{}
	for _, fact := range doc.Capabilities {
		if !inMigrationSet[fact.Name] {
			continue
		}
		for _, section := range fact.Configs {
			declaredSections[section] = true
		}
	}
	for _, service := range result.RuntimeServices {
		section, known := runtimeServiceSection[string(service)]
		if !known {
			diagnostics = append(diagnostics, capability.Diagnostic{
				Phase:   capability.PhaseRuntimeConfig,
				Code:    capability.CodeUnknownRuntimeService,
				Path:    "capabilities.*.runtime_services",
				Message: fmt.Sprintf("unknown runtime service %q", service),
				Fix:     "remove the service or declare it in the framework",
			})
			continue
		}
		if declaredSections[section] {
			continue
		}
		for _, destination := range []string{"configs/app.yaml", "configs/app.prod.yaml"} {
			sections, ok := mergeSections(doc.Merges, destination)
			if ok && sections[section] {
				continue
			}
			diagnostics = append(diagnostics, capability.Diagnostic{
				Phase:   capability.PhaseRuntimeConfig,
				Code:    capability.CodeMissingRuntimeServiceConfig,
				Path:    "merges." + destination,
				Message: fmt.Sprintf("runtime service %q requires config section %q in %s", service, section, destination),
				Fix:     fmt.Sprintf("add %q to the merged config sections", section),
			})
		}
	}
	return diagnostics
}

func middlewareDiagnostics(caps []capability.Descriptor) []capability.Diagnostic {
	var diagnostics []capability.Diagnostic
	hasProtectedRoutes := false
	var primary []string
	var fallback []string
	for _, descriptor := range caps {
		if descriptor.Normalized() == capability.MountProtected {
			hasProtectedRoutes = true
		}
		switch descriptor.ProtectedHTTPMiddleware {
		case capability.ProtectedHTTPMiddlewareNone:
		case capability.ProtectedHTTPMiddlewarePrimary:
			primary = append(primary, descriptor.Name)
		case capability.ProtectedHTTPMiddlewareFallback:
			fallback = append(fallback, descriptor.Name)
		default:
			diagnostics = append(diagnostics, capability.Diagnostic{
				Phase:      capability.PhaseMiddleware,
				Code:       capability.CodeMiddlewareConflict,
				Capability: descriptor.Name,
				Path:       "capabilities." + descriptor.Name + ".protected_http_middleware",
				Message:    fmt.Sprintf("capability %q declares unknown protected middleware mode %q", descriptor.Name, descriptor.ProtectedHTTPMiddleware),
				Fix:        "declare primary, fallback, or leave the mode empty",
			})
		}
	}
	if len(primary) > 1 {
		for _, name := range primary[1:] {
			diagnostics = append(diagnostics, capability.Diagnostic{
				Phase:      capability.PhaseMiddleware,
				Code:       capability.CodeMiddlewareConflict,
				Capability: name,
				Path:       "capabilities." + name + ".protected_http_middleware",
				Message:    "multiple primary protected middleware providers: " + strings.Join(primary, ", "),
				Fix:        "keep exactly one primary provider",
			})
		}
	}
	if len(primary) == 0 && len(fallback) > 1 {
		for _, name := range fallback[1:] {
			diagnostics = append(diagnostics, capability.Diagnostic{
				Phase:      capability.PhaseMiddleware,
				Code:       capability.CodeMiddlewareConflict,
				Capability: name,
				Path:       "capabilities." + name + ".protected_http_middleware",
				Message:    "multiple fallback protected middleware providers: " + strings.Join(fallback, ", "),
				Fix:        "keep exactly one fallback provider",
			})
		}
	}
	if hasProtectedRoutes && len(primary) == 0 && len(fallback) == 0 {
		diagnostics = append(diagnostics, capability.Diagnostic{
			Phase:   capability.PhaseMiddleware,
			Code:    capability.CodeMiddlewareProviderMissing,
			Path:    "capabilities.*.protected_http_middleware",
			Message: "protected routes are selected but no capability provides protected middleware",
			Fix:     "add a capability that provides protected middleware",
		})
	}
	return diagnostics
}

func mergeSections(merges []manifest.MergeAction, destination string) (map[string]bool, bool) {
	for _, merge := range merges {
		if merge.Destination != destination {
			continue
		}
		sections := make(map[string]bool, len(merge.Sections))
		for _, section := range merge.Sections {
			sections[section] = true
		}
		return sections, true
	}
	return nil, false
}
