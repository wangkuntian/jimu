// Package preflight validates generator manifest facts before any target file is written.
package preflight

import (
	"fmt"
	"maps"
	"slices"
	"strings"

	"jimu/tools/generator/manifest"
)

const (
	protectedMiddlewareNone     = ""
	protectedMiddlewarePrimary  = "primary"
	protectedMiddlewareFallback = "fallback"
)

var runtimeServiceSection = map[string]string{
	"mysql":    "db",
	"redis":    "redis",
	"kafka":    "queue",
	"rabbitmq": "queue",
}

// Validate rejects selections that cannot produce a self-consistent generated project.
func Validate(doc manifest.Document) error {
	byName := make(map[string]manifest.Capability, len(doc.Capabilities))
	for _, capability := range doc.Capabilities {
		byName[capability.Name] = capability
	}

	declared, err := selectedCapabilities(doc.Selection.Capabilities, byName)
	if err != nil {
		return err
	}
	if err := validateProtectedMiddleware(declared); err != nil {
		return err
	}

	migrationNames := slices.Clone(doc.Selection.Capabilities)
	for _, capability := range doc.Capabilities {
		if capability.MigrationOnly {
			migrationNames = append(migrationNames, capability.Name)
		}
	}
	migrationCaps, err := selectedCapabilities(migrationNames, byName)
	if err != nil {
		return err
	}
	if err := validateMigrations(migrationCaps); err != nil {
		return err
	}
	return validateRuntimeServices(migrationCaps, doc.Merges)
}

func selectedCapabilities(names []string, byName map[string]manifest.Capability) ([]manifest.Capability, error) {
	out := make([]manifest.Capability, 0, len(names))
	seen := make(map[string]bool, len(names))
	for _, name := range names {
		if seen[name] {
			continue
		}
		seen[name] = true
		capability, ok := byName[name]
		if !ok {
			return nil, fmt.Errorf("selected capability %q is missing from manifest facts", name)
		}
		out = append(out, capability)
	}
	return out, nil
}

func validateProtectedMiddleware(caps []manifest.Capability) error {
	hasProtectedRoutes := false
	primary := []string{}
	fallback := []string{}
	for _, capability := range caps {
		if capability.Mount == "protected" {
			hasProtectedRoutes = true
		}
		switch capability.ProtectedHTTPMiddleware {
		case protectedMiddlewareNone:
		case protectedMiddlewarePrimary:
			primary = append(primary, capability.Name)
		case protectedMiddlewareFallback:
			fallback = append(fallback, capability.Name)
		default:
			return fmt.Errorf("capability %q declares unknown protected middleware mode %q", capability.Name, capability.ProtectedHTTPMiddleware)
		}
	}
	if len(primary) > 1 {
		return fmt.Errorf("multiple primary protected middleware providers: %s", strings.Join(primary, ", "))
	}
	if len(primary) == 0 && len(fallback) > 1 {
		return fmt.Errorf("multiple fallback protected middleware providers: %s", strings.Join(fallback, ", "))
	}
	if hasProtectedRoutes && len(primary) == 0 && len(fallback) == 0 {
		return fmt.Errorf("protected routes are selected but no capability provides protected middleware")
	}
	return nil
}

func validateMigrations(caps []manifest.Capability) error {
	present := make(map[string]bool, len(caps))
	for _, capability := range caps {
		present[capability.Name] = true
	}
	tableOwners := map[string]string{}
	migrationPaths := map[string]string{}
	for _, capability := range caps {
		for _, dependency := range capability.MigrationSchemaDeps {
			if !present[dependency] {
				return fmt.Errorf("capability %q has migration schema dependency %q that is not selected", capability.Name, dependency)
			}
		}
		for _, table := range capability.Owns {
			if previous, ok := tableOwners[table]; ok && previous != capability.Name {
				return fmt.Errorf("table %q is owned by both %q and %q", table, previous, capability.Name)
			}
			tableOwners[table] = capability.Name
		}
		if len(capability.Owns) > 0 && len(capability.Migrations) == 0 {
			return fmt.Errorf("capability %q owns table %q but has no migration", capability.Name, capability.Owns[0])
		}
		for _, migration := range capability.Migrations {
			if previous, ok := migrationPaths[migration]; ok {
				return fmt.Errorf("migration %q is included by both %q and %q", migration, previous, capability.Name)
			}
			migrationPaths[migration] = capability.Name
		}
	}
	return nil
}

func validateRuntimeServices(caps []manifest.Capability, merges []manifest.MergeAction) error {
	requiredSections := map[string]string{}
	declaredSections := map[string]bool{}
	for _, capability := range caps {
		for _, section := range capability.Configs {
			declaredSections[section] = true
		}
		for _, service := range capability.RuntimeServices {
			section, ok := runtimeServiceSection[service]
			if !ok {
				return fmt.Errorf("capability %q declares unknown runtime service %q", capability.Name, service)
			}
			requiredSections[section] = service
		}
	}
	if len(requiredSections) == 0 {
		return nil
	}

	for _, destination := range []string{"configs/app.yaml", "configs/app.prod.yaml"} {
		sections, ok := mergeSections(merges, destination)
		for _, section := range slices.Sorted(maps.Keys(requiredSections)) {
			if declaredSections[section] {
				continue
			}
			if !ok {
				return fmt.Errorf("runtime service %q requires config %s but no merge action exists", requiredSections[section], destination)
			}
			if !sections[section] {
				service := requiredSections[section]
				return fmt.Errorf("runtime service %q requires config section %q in %s", service, section, destination)
			}
		}
	}
	return nil
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
