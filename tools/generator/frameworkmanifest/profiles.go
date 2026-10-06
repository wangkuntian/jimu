package frameworkmanifest

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"jimu/internal/assembly"
	"jimu/internal/capabilities/catalog"
	"jimu/internal/capability"
	"jimu/internal/contract"
	"jimu/internal/profiles/full"
	"jimu/internal/profiles/registry"
)

func resolveSelection(root string, req Request) (selection, error) {
	if req.Profile != "" && req.With != "" {
		return selection{}, fmt.Errorf("--profile and --with are mutually exclusive")
	}
	if req.Profile == "" && req.With == "" {
		return selection{}, fmt.Errorf("one of --profile or --with is required")
	}
	all, inCatalog := allDescriptors()
	byName := make(map[string]contract.Descriptor, len(all))
	for _, descriptor := range all {
		byName[descriptor.Name] = descriptor
	}
	selected := selection{drivers: map[string][]string{}, ungated: map[string]bool{}}
	if req.Profile != "" {
		assemblyValue, err := lookupProfile(req.Profile)
		if err != nil {
			return selection{}, err
		}
		selected.profile = req.Profile
		selected.shape = req.Profile
		declared := map[string]bool{}
		for _, item := range assemblyValue.Capabilities {
			declared[item.Descriptor.Name] = true
			if len(item.Drivers) > 0 {
				selected.drivers[item.Descriptor.Name] = slices.Clone(item.Drivers)
			}
			if item.Ungated {
				selected.ungated[item.Descriptor.Name] = true
			}
		}
		for _, descriptor := range catalog.All() {
			if declared[descriptor.Name] {
				selected.declared = append(selected.declared, descriptor.Name)
			}
		}
		for _, item := range assemblyValue.Capabilities {
			name := item.Descriptor.Name
			if !inCatalog[name] {
				selected.declared = append(selected.declared, name)
			}
		}
		selected.assemblies = slices.Clone(assemblyValue.Capabilities)
	} else {
		selected.shape = req.Shape
		if selected.shape == "" {
			selected.shape = "app"
		}
		if err := validateShape(selected.shape); err != nil {
			return selection{}, err
		}
		names, overrides, err := parseWith(req.With)
		if err != nil {
			return selection{}, err
		}
		resolved, err := capability.Resolve(all, names)
		if err != nil {
			return selection{}, err
		}
		for _, descriptor := range resolved {
			selected.declared = append(selected.declared, descriptor.Name)
			if !inCatalog[descriptor.Name] {
				selected.ungated[descriptor.Name] = true
			}
			if len(descriptor.Drivers) > 0 {
				selected.drivers[descriptor.Name] = []string{descriptor.Drivers[0]}
			}
		}
		for name, driver := range overrides {
			descriptor, ok := byName[name]
			if !ok || !slices.Contains(descriptor.Drivers, driver) {
				available := ""
				if ok {
					available = strings.Join(descriptor.Drivers, ", ")
				}
				return selection{}, fmt.Errorf("unknown driver %q for capability %q (available: %s)", driver, name, available)
			}
			selected.drivers[name] = []string{driver}
		}
		selected.assemblies = assemblyCapabilitiesFor(selected.declared, selected.drivers)
	}
	return completeSelection(root, selected, all)
}

func allDescriptors() ([]contract.Descriptor, map[string]bool) {
	all := catalog.All()
	inCatalog := make(map[string]bool, len(all))
	for _, descriptor := range all {
		inCatalog[descriptor.Name] = true
	}
	for _, item := range full.Assembly().Capabilities {
		if inCatalog[item.Descriptor.Name] {
			continue
		}
		all = append(all, item.Descriptor)
	}
	return all, inCatalog
}

func lookupProfile(name string) (assembly.Assembly, error) {
	return registry.Lookup(name)
}

func assemblyCapabilitiesFor(names []string, drivers map[string][]string) []assembly.Capability {
	byName := map[string]assembly.Capability{}
	for _, item := range full.Assembly().Capabilities {
		byName[item.Descriptor.Name] = item
	}
	out := make([]assembly.Capability, 0, len(names))
	for _, name := range names {
		item := byName[name]
		item.Drivers = slices.Clone(drivers[name])
		out = append(out, item)
	}
	return out
}

func parseWith(value string) ([]string, map[string]string, error) {
	var names []string
	overrides := map[string]string{}
	for _, item := range strings.Split(value, ",") {
		item = strings.TrimSpace(item)
		if item == "" {
			return nil, nil, fmt.Errorf("--with contains an empty capability")
		}
		name, driver, hasDriver := strings.Cut(item, ":")
		name = strings.TrimSpace(name)
		if name == "" {
			return nil, nil, fmt.Errorf("--with contains an empty capability")
		}
		names = append(names, name)
		if !hasDriver {
			continue
		}
		driver = strings.TrimSpace(driver)
		if driver == "" {
			return nil, nil, fmt.Errorf("capability %q has an empty driver", name)
		}
		if previous, ok := overrides[name]; ok && previous != driver {
			return nil, nil, fmt.Errorf("capability %q selects two drivers (%q and %q)", name, previous, driver)
		}
		overrides[name] = driver
	}
	return names, overrides, nil
}

func validateShape(shape string) error {
	if shape == "" || shape[0] < 'a' || shape[0] > 'z' {
		return fmt.Errorf("invalid --shape %q", shape)
	}
	for _, char := range shape {
		if (char < 'a' || char > 'z') && (char < '0' || char > '9') && char != '_' {
			return fmt.Errorf("invalid --shape %q", shape)
		}
	}
	for _, reserved := range []string{"registry", "active", "catalog", "configs", "profiles", "internal", "cmd"} {
		if shape == reserved {
			return fmt.Errorf("invalid --shape %q: reserved name", shape)
		}
	}
	return nil
}

func completeSelection(root string, selected selection, all []contract.Descriptor) (selection, error) {
	requires := make(map[string][]string, len(all))
	known := make(map[string]bool, len(all))
	for _, descriptor := range all {
		requires[descriptor.Name] = descriptor.Requires
		known[descriptor.Name] = true
	}
	selected.known = make([]string, 0, len(all))
	for _, descriptor := range all {
		selected.known = append(selected.known, descriptor.Name)
	}
	closure, roots, err := capabilityClosure(root, selected.declared, requires, known)
	if err != nil {
		return selection{}, err
	}
	selected.roots = roots
	declaredSet := make(map[string]bool, len(selected.declared))
	for _, name := range selected.declared {
		declaredSet[name] = true
	}
	for _, name := range selected.declared {
		if info, err := os.Stat(filepath.Join(root, "internal", "capabilities", name)); err != nil || !info.IsDir() {
			return selection{}, fmt.Errorf("capability %q has no source directory", name)
		}
	}
	only := map[string]bool{}
	for name := range declaredSet {
		for _, dependency := range catalog.MigrationSchemaDeps[name] {
			if !declaredSet[dependency] {
				only[dependency] = true
			}
		}
	}
	covered := map[string]bool{}
	copySet := map[string]bool{}
	for _, name := range closure {
		copySet[name] = true
		covered[name] = true
	}
	for name := range only {
		copySet[name] = true
		covered[name] = true
	}
	for _, entry := range []string{"access/domain", "tenant/domain", "user/domain"} {
		name := strings.SplitN(entry, "/", 2)[0]
		if !covered[name] {
			selected.domain = append(selected.domain, name)
			copySet[name] = true
		}
	}
	selected.migration = make([]string, 0, len(only))
	for name := range only {
		selected.migration = append(selected.migration, name)
	}
	slices.Sort(selected.migration)
	selected.domain = uniqueSorted(selected.domain)
	selected.copy = make([]string, 0, len(copySet))
	for name := range copySet {
		selected.copy = append(selected.copy, name)
	}
	slices.Sort(selected.copy)
	return selected, nil
}

func uniqueSorted(values []string) []string {
	values = slices.Clone(values)
	slices.Sort(values)
	return slices.Compact(values)
}
