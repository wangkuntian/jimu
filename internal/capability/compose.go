package capability

import (
	"fmt"
	"io/fs"
	"slices"
	"sort"
	"strings"

	"jimu/internal/contract"
)

// ComposeRequest 是一次组合解析的输入。
type ComposeRequest struct {
	// Capabilities 是用户请求的装配集，顺序保留用于 requested 输出。
	Capabilities []string
	// Drivers 是每个能力选择的驱动；一个能力可选择多个驱动（内部 profile 会出现）。
	Drivers map[string][]string
	// DefaultDrivers 为 true 时，声明了驱动的能力在未显式选择时取 Drivers[0]。
	// `--with` 使用 true；检查已有 manifest 时使用 false，避免掩盖缺失的选择。
	DefaultDrivers bool
	// RequireResolvedSelection 为 true 时，请求集必须已经包含硬依赖闭包；
	// 否则只报告 missing_dependency，而不是静默补齐。用于检查已有 manifest。
	RequireResolvedSelection bool
	// MigrationCarriers 是项目已经额外携带的迁移能力（manifest 的 MigrationOnly 事实）。
	MigrationCarriers []string
	// CheckMigrationCarriers 为 true 时，schema 依赖必须已被 Capabilities 或
	// MigrationCarriers 覆盖；否则报告 missing_migration_schema_dependency。
	CheckMigrationCarriers bool
	// MigrationFiles 覆盖按能力名索引的迁移文件清单；nil 时从 Descriptor.Migrations 读取。
	// 检查已有 manifest 时使用，因为 manifest 保存的是路径而不是 embed.FS。
	MigrationFiles map[string][]string
	// Support 是本次组合要报告的支持级别。
	Support contract.SupportLevel
}

// ComposeResult 是组合解析的完整结果。即使存在诊断也返回可用的部分结果，
// 便于调用方一次报告多条问题。
type ComposeResult struct {
	Requested         []string
	Resolved          []contract.Descriptor
	AutoAdded         []string
	Drivers           map[string][]string
	MigrationCarriers []string
	Migrations        []string
	RuntimeServices   []contract.RuntimeService
	Validity          contract.Validity
	Support           contract.SupportLevel
	Diagnostics       []Diagnostic
}

// Err 在组合非法时返回结构化 error。
func (r ComposeResult) Err() error {
	if len(r.Diagnostics) == 0 {
		return nil
	}
	return &Error{Diagnostics: slices.Clone(r.Diagnostics)}
}

// Compose 按固定顺序解析固定 catalog 子集：
// 输入校验 → 硬依赖闭包 → 冲突 → Provides/Consumes → 驱动 → 迁移 schema 依赖 → 表/迁移。
// 运行时服务/配置与中间件需要 manifest 的 merge 事实，由生成器的 preflight 追加。
func Compose(caps []contract.Descriptor, request ComposeRequest) ComposeResult {
	result := ComposeResult{
		Requested: slices.Clone(request.Capabilities),
		Drivers:   map[string][]string{},
		Support:   request.Support,
	}
	byName := make(map[string]contract.Descriptor, len(caps))
	order := make([]string, 0, len(caps))
	for _, descriptor := range caps {
		if _, exists := byName[descriptor.Name]; !exists {
			order = append(order, descriptor.Name)
		}
		byName[descriptor.Name] = descriptor
	}
	add := func(phase Phase, code DiagnosticCode, capability, path, message, fix string) {
		result.Diagnostics = append(result.Diagnostics, Diagnostic{
			Phase: phase, Code: code, Capability: capability, Path: path, Message: message, Fix: fix,
		})
	}

	requested := make(map[string]bool, len(request.Capabilities))
	selected := make(map[string]bool, len(request.Capabilities))
	reportedInput := map[string]bool{}
	for index, name := range request.Capabilities {
		path := fmt.Sprintf("selection.capabilities[%d]", index)
		if name == "" {
			add(PhaseInput, CodeUnknownCapability, "", path, "capability name must not be empty", "remove the empty selection")
			continue
		}
		if _, known := byName[name]; !known {
			key := "unknown:" + name
			if !reportedInput[key] {
				reportedInput[key] = true
				add(PhaseInput, CodeUnknownCapability, name, path,
					fmt.Sprintf("unknown capability %q, available: %s", name, strings.Join(order, ", ")),
					"choose a capability name from the catalog")
			}
			continue
		}
		if requested[name] {
			add(PhaseInput, CodeDuplicateCapability, name, path,
				fmt.Sprintf("capability %q is requested more than once", name),
				"remove the duplicate selection")
			continue
		}
		requested[name] = true
		selected[name] = true
	}
	for _, name := range sortedMapKeys(request.Drivers) {
		if _, known := byName[name]; known {
			continue
		}
		key := "unknown:" + name
		if reportedInput[key] {
			continue
		}
		reportedInput[key] = true
		add(PhaseInput, CodeUnknownCapability, name, "selection.drivers."+name,
			fmt.Sprintf("unknown capability %q, available: %s", name, strings.Join(order, ", ")),
			"choose a capability name from the catalog")
	}

	// 硬依赖闭包。按清单顺序反复补齐，保证传递依赖也进入结果。
	autoAdded := map[string]bool{}
	reportedDependency := map[string]bool{}
	for changed := true; changed; {
		changed = false
		for _, name := range order {
			if !selected[name] {
				continue
			}
			for _, dependency := range byName[name].Requires {
				path := "capabilities." + name + ".requires"
				if _, known := byName[dependency]; !known {
					key := name + "\x00" + dependency
					if !reportedDependency[key] {
						reportedDependency[key] = true
						add(PhaseClosure, CodeMissingDependency, name, path,
							fmt.Sprintf("capability %q requires unknown capability %q", name, dependency),
							"fix the declared dependency or add the capability")
					}
					continue
				}
				if selected[dependency] {
					continue
				}
				if request.RequireResolvedSelection {
					add(PhaseClosure, CodeMissingDependency, name, path,
						fmt.Sprintf("capability %q requires %q but the selection does not include it", name, dependency),
						fmt.Sprintf("add %q to the composition", dependency))
					// 严格模式只报告，不把缺失依赖并入 selected：否则 Resolved 会包含
					// manifest 未声明的能力，掩盖后续端口/中间件诊断。
					continue
				}
				if !requested[dependency] {
					autoAdded[dependency] = true
				}
				selected[dependency] = true
				changed = true
			}
		}
	}
	for _, name := range order {
		if !selected[name] {
			continue
		}
		result.Resolved = append(result.Resolved, deepCopyDescriptor(byName[name]))
		if autoAdded[name] {
			result.AutoAdded = append(result.AutoAdded, name)
		}
	}

	// 冲突：只检查已选中的成对能力。
	for _, name := range order {
		if !selected[name] {
			continue
		}
		for _, other := range byName[name].Conflicts {
			if !selected[other] {
				continue
			}
			add(PhaseConflict, CodeConflict, name, "capabilities."+name+".conflicts",
				fmt.Sprintf("capability %q conflicts with %q", name, other),
				fmt.Sprintf("remove %q or %q from the composition", name, other))
		}
	}

	// Provides/Consumes：重复提供方与必需消费者缺失。
	providers := map[string][]string{}
	for _, name := range order {
		if !selected[name] {
			continue
		}
		for _, port := range byName[name].Provides {
			providers[port.Name] = append(providers[port.Name], name)
		}
	}
	for _, port := range sortedMapKeys(providers) {
		names := providers[port]
		if len(names) < 2 {
			continue
		}
		for _, name := range names[1:] {
			add(PhasePorts, CodeDuplicateProvider, name, "capabilities."+name+".provides",
				fmt.Sprintf("port %q is provided by both %q and %q", port, names[0], name),
				"keep exactly one provider for this port")
		}
	}
	for _, name := range order {
		if !selected[name] {
			continue
		}
		for _, port := range byName[name].Consumes {
			if !port.Required || len(providers[port.Name]) > 0 {
				continue
			}
			add(PhasePorts, CodeMissingProvider, name, "capabilities."+name+".consumes",
				fmt.Sprintf("capability %q requires port %q but no selected capability provides it", name, port.Name),
				"add a capability that provides this port or make the dependency optional")
		}
	}

	// 驱动：校验选择并推导驱动级运行时服务。
	for _, name := range order {
		if !selected[name] {
			continue
		}
		descriptor := byName[name]
		chosen := slices.Clone(request.Drivers[name])
		path := "selection.drivers." + name
		if len(descriptor.Drivers) == 0 {
			if len(chosen) > 0 {
				add(PhaseDrivers, CodeUnknownDriver, name, path,
					fmt.Sprintf("capability %q has no drivers; cannot select %q", name, chosen[0]),
					"remove the driver selection")
			}
			continue
		}
		if len(chosen) == 0 {
			if request.DefaultDrivers {
				chosen = []string{descriptor.Drivers[0]}
			} else {
				add(PhaseDrivers, CodeMissingDriver, name, path,
					fmt.Sprintf("capability %q requires a driver selection (available: %s)", name, strings.Join(descriptor.Drivers, ", ")),
					"select one of: "+strings.Join(descriptor.Drivers, ", "))
				continue
			}
		}
		var valid []string
		for _, driver := range chosen {
			if !slices.Contains(descriptor.Drivers, driver) {
				add(PhaseDrivers, CodeUnknownDriver, name, path,
					fmt.Sprintf("unknown driver %q for capability %q (available: %s)", driver, name, strings.Join(descriptor.Drivers, ", ")),
					"choose one of: "+strings.Join(descriptor.Drivers, ", "))
				continue
			}
			if !slices.Contains(valid, driver) {
				valid = append(valid, driver)
			}
		}
		if len(valid) > 0 {
			result.Drivers[name] = valid
		}
	}
	for _, name := range sortedMapKeys(request.Drivers) {
		if selected[name] {
			continue
		}
		if _, known := byName[name]; !known {
			continue
		}
		add(PhaseDrivers, CodeUnknownDriver, name, "selection.drivers."+name,
			fmt.Sprintf("driver selected for capability %q which is not in the composition", name),
			"remove the driver selection or add the capability")
	}

	// 迁移 schema 依赖：检查覆盖，并在非严格模式下自动携带。
	carriers := map[string]bool{}
	for _, name := range request.MigrationCarriers {
		carriers[name] = true
		if _, known := byName[name]; !known {
			add(PhaseMigrationSchema, CodeUnknownCapability, name, "capabilities."+name,
				fmt.Sprintf("migration carrier capability %q is unknown", name),
				"remove the migration carrier or add the capability")
		}
	}
	reportedSchema := map[string]bool{}
	for changed := true; changed; {
		changed = false
		for _, name := range order {
			if !selected[name] && !carriers[name] {
				continue
			}
			for _, dependency := range byName[name].MigrationSchemaDeps {
				path := "capabilities." + name + ".migration_schema_deps"
				if _, known := byName[dependency]; !known {
					key := name + "\x00" + dependency
					if reportedSchema[key] {
						continue
					}
					reportedSchema[key] = true
					add(PhaseMigrationSchema, CodeMissingMigrationSchemaDependency, name, path,
						fmt.Sprintf("capability %q has unknown migration schema dependency %q", name, dependency),
						"fix the declared migration schema dependency")
					continue
				}
				if selected[dependency] || carriers[dependency] {
					continue
				}
				if request.CheckMigrationCarriers {
					add(PhaseMigrationSchema, CodeMissingMigrationSchemaDependency, name, path,
						fmt.Sprintf("capability %q has migration schema dependency %q that is not selected or carried", name, dependency),
						fmt.Sprintf("include %q migrations in the composition", dependency))
				}
				carriers[dependency] = true
				changed = true
			}
		}
	}
	for _, name := range order {
		if carriers[name] && !selected[name] {
			result.MigrationCarriers = append(result.MigrationCarriers, name)
		}
	}

	// 表与迁移：所有权唯一，迁移文件唯一，拥有表必须有迁移。
	tableOwners := map[string]string{}
	migrationOwners := map[string]string{}
	for _, name := range order {
		if !selected[name] && !carriers[name] {
			continue
		}
		descriptor := byName[name]
		files := migrationFiles(name, descriptor, request.MigrationFiles)
		for _, table := range descriptor.Owns {
			if previous, exists := tableOwners[table]; exists && previous != name {
				add(PhaseTables, CodeDuplicateTableOwner, name, "capabilities."+name+".owns",
					fmt.Sprintf("table %q is owned by both %q and %q", table, previous, name),
					"assign each table to exactly one capability")
				continue
			}
			tableOwners[table] = name
		}
		if len(descriptor.Owns) > 0 && len(files) == 0 {
			add(PhaseTables, CodeMissingMigration, name, "capabilities."+name+".migrations",
				fmt.Sprintf("capability %q owns table %q but has no migration", name, descriptor.Owns[0]),
				"add a migration for the owned table")
		}
		for _, file := range files {
			if previous, exists := migrationOwners[file]; exists && previous != name {
				add(PhaseTables, CodeDuplicateMigration, name, "capabilities."+name+".migrations",
					fmt.Sprintf("migration %q is included by both %q and %q", file, previous, name),
					"remove the duplicate migration inclusion")
				continue
			}
			migrationOwners[file] = name
		}
		result.Migrations = append(result.Migrations, files...)
	}
	result.Migrations = uniqueSorted(result.Migrations)

	// 运行时服务：声明服务 ∪ 选中驱动规格声明的额外服务。
	serviceSet := map[contract.RuntimeService]bool{}
	for _, name := range order {
		if !selected[name] && !carriers[name] {
			continue
		}
		descriptor := byName[name]
		for _, service := range descriptor.RuntimeServices {
			serviceSet[service] = true
		}
		for _, driver := range result.Drivers[name] {
			for _, spec := range descriptor.DriverSpecs {
				if spec.Name != driver {
					continue
				}
				for _, service := range spec.RuntimeServices {
					serviceSet[service] = true
				}
			}
		}
	}
	result.RuntimeServices = append(result.RuntimeServices, sortedServices(serviceSet)...)

	result.Validity = Validity(result.Diagnostics)
	SortDiagnostics(result.Diagnostics)
	return result
}

// MigrationCarriers 返回必须为 schema 依赖携带的迁移能力名，按清单顺序。
// 只计算已知能力；未知依赖由 Compose 报告。
func MigrationCarriers(caps []contract.Descriptor, declared []string) []string {
	byName := make(map[string]contract.Descriptor, len(caps))
	order := make([]string, 0, len(caps))
	for _, descriptor := range caps {
		if _, exists := byName[descriptor.Name]; !exists {
			order = append(order, descriptor.Name)
		}
		byName[descriptor.Name] = descriptor
	}
	present := make(map[string]bool, len(declared))
	for _, name := range declared {
		present[name] = true
	}
	carriers := map[string]bool{}
	for changed := true; changed; {
		changed = false
		for _, name := range order {
			if !present[name] && !carriers[name] {
				continue
			}
			descriptor, known := byName[name]
			if !known {
				continue
			}
			for _, dependency := range descriptor.MigrationSchemaDeps {
				if _, known := byName[dependency]; !known {
					continue
				}
				if present[dependency] || carriers[dependency] {
					continue
				}
				carriers[dependency] = true
				changed = true
			}
		}
	}
	out := make([]string, 0, len(carriers))
	for _, name := range order {
		if carriers[name] {
			out = append(out, name)
		}
	}
	return out
}

func migrationFiles(name string, descriptor contract.Descriptor, overrides map[string][]string) []string {
	if overrides != nil {
		if files, ok := overrides[name]; ok {
			return uniqueSorted(files)
		}
	}
	if descriptor.Migrations == nil {
		return []string{}
	}
	var files []string
	_ = fs.WalkDir(descriptor.Migrations, "migrations/mysql", func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() || !strings.HasSuffix(path, ".sql") {
			return err
		}
		files = append(files, "internal/capabilities/"+name+"/"+path)
		return nil
	})
	return uniqueSorted(files)
}

func uniqueSorted(values []string) []string {
	out := slices.Clone(values)
	slices.Sort(out)
	return slices.Compact(out)
}

func sortedMapKeys[V any](values map[string]V) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func sortedServices(values map[contract.RuntimeService]bool) []contract.RuntimeService {
	out := make([]contract.RuntimeService, 0, len(values))
	for value := range values {
		out = append(out, value)
	}
	slices.Sort(out)
	return out
}
