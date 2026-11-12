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
	selected := selection{drivers: map[string][]string{}, ungated: map[string]bool{}}
	if req.Profile != "" {
		assemblyValue, err := lookupProfile(req.Profile)
		if err != nil {
			return selection{}, err
		}
		selected.profile = req.Profile
		selected.support = "internal"
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
		for name, values := range req.Drivers {
			var descriptor contract.Descriptor
			found := false
			for _, candidate := range all {
				if candidate.Name == name {
					descriptor = candidate
					found = true
					break
				}
			}
			if !found {
				return selection{}, fmt.Errorf("unknown capability %q", name)
			}
			for _, driver := range values {
				if !slices.Contains(descriptor.Drivers, driver) {
					return selection{}, fmt.Errorf("unknown driver %q for capability %q (available: %s)",
						driver, name, strings.Join(descriptor.Drivers, ", "))
				}
			}
			selected.drivers[name] = slices.Clone(values)
		}
		selected.assemblies = slices.Clone(assemblyValue.Capabilities)
	} else {
		selected.support = "experimental"
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
		requestedDrivers := cloneDrivers(req.Drivers)
		for name, driver := range overrides {
			requestedDrivers[name] = []string{driver}
		}
		composed := capability.Compose(all, capability.ComposeRequest{
			Capabilities:   names,
			Drivers:        requestedDrivers,
			DefaultDrivers: true,
			Support:        contract.SupportExperimental,
		})
		if err := composed.Err(); err != nil {
			return selection{}, err
		}
		selected.requested = composed.Requested
		selected.autoAdded = composed.AutoAdded
		for _, descriptor := range composed.Resolved {
			selected.declared = append(selected.declared, descriptor.Name)
			if !inCatalog[descriptor.Name] {
				selected.ungated[descriptor.Name] = true
			}
		}
		selected.drivers = composed.Drivers
		selected.migration = composed.MigrationCarriers
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

// CompositionDescriptors 返回框架当前的全量组合事实，供写前预检与 `jimu check`
// 叠加到 manifest 事实之上（schema v2 尚未持久化 conflicts/provides/consumes）。
func CompositionDescriptors() []contract.Descriptor {
	all, _ := allDescriptors()
	return all
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
	for _, name := range selected.declared {
		if info, err := os.Stat(filepath.Join(root, "internal", "capabilities", name)); err != nil || !info.IsDir() {
			return selection{}, fmt.Errorf("capability %q has no source directory", name)
		}
	}
	carriers := capability.MigrationCarriers(all, selected.declared)
	only := make(map[string]bool, len(carriers))
	for _, name := range carriers {
		only[name] = true
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
	selected.migration = slices.Clone(carriers)
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

// resolvePresetSelection 把预设解析成一次生成选择。
//
// 与 --with 的差别只有两处：能力集/驱动集来自预设（不是用户手敲），
// 并把 template 记为预设名，使 manifest 能区分"预设起点"与"专家组合"。
// 其余（编译闭包、迁移携带、拓扑序）全部复用同一套 completeSelection。
func resolvePresetSelection(root string, preset Preset) (selection, error) {
	all, inCatalog := allDescriptors()
	composed := capability.Compose(all, capability.ComposeRequest{
		Capabilities:   preset.Capabilities,
		Drivers:        preset.Drivers,
		DefaultDrivers: false,
		Support:        contract.SupportLevel(preset.Support),
	})
	if err := composed.Err(); err != nil {
		return selection{}, err
	}
	selected := selection{
		shape:    preset.Name,
		template: preset.Name,
		support:  preset.Support,
		drivers:  composed.Drivers,
		ungated:  map[string]bool{},
	}
	if selected.drivers == nil {
		selected.drivers = map[string][]string{}
	}
	selected.requested = composed.Requested
	selected.autoAdded = composed.AutoAdded
	for _, descriptor := range composed.Resolved {
		selected.declared = append(selected.declared, descriptor.Name)
		if !inCatalog[descriptor.Name] {
			selected.ungated[descriptor.Name] = true
		}
	}
	selected.migration = composed.MigrationCarriers
	selected.assemblies = assemblyCapabilitiesFor(selected.declared, selected.drivers)
	return completeSelection(root, selected, all)
}
