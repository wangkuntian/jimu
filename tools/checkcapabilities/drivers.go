// Command 内新增的驱动门禁：设计 §3.7「漏 import 会在运行时报未知驱动而非编译失败」
// 的静态保险。五条断言：
//
//	① 可用集自洽：Descriptor.Drivers 项非空不重复，且对应目录真实存在；
//	② 核心零驱动：能力核心的生产闭包不含任何驱动包，也不含重型第三方依赖；
//	③ 选中 == 实际：每个形态的驱动包闭包与 Assembly().Capabilities[*].Drivers 逐值相等；
//	④ 归属：驱动包的生产 import 方只能是 internal/profiles/*；
//	⑤ 形态只 import 已声明驱动：形态生产代码的直接 import 中，凡 capabilities 子包必为
//	   能力根包或已声明驱动包 —— 抓「新增驱动目录 + profile blank import 却忘声明」。
package main

import (
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"jimu/internal/assembly"
	"jimu/internal/capabilities/catalog"
	"jimu/internal/contract"
	"jimu/internal/profiles/enterprise"
	"jimu/internal/profiles/full"
	"jimu/internal/profiles/machine"
	"jimu/internal/profiles/minimal"
	"jimu/internal/profiles/saas"
	"jimu/tools/internal/heavydeps"

	"golang.org/x/tools/go/packages"
)

// modulePath 本模块的 import 前缀；驱动包路径与归属判定都基于它。
const modulePath = "jimu"

// driverPath 是驱动包全路径（驱动名 = 目录名，故无需对照表）。
func driverPath(cap, driver string) string {
	return modulePath + "/internal/capabilities/" + cap + "/" + driver
}

// profileAssemblies 驱动的形态清单表：新增形态需同步这里（门禁的错误定位依赖形态名）。
func profileAssemblies() map[string]assembly.Assembly {
	return map[string]assembly.Assembly{
		"full":       full.Assembly(),
		"minimal":    minimal.Assembly(),
		"saas":       saas.Assembly(),
		"enterprise": enterprise.Assembly(),
		"machine":    machine.Assembly(),
	}
}

// driverDescriptors 汇总所有「声明过驱动」的 Descriptor：catalog 条目 + 各形态清单里的
// 非 catalog 条目（storage 等 Ungated 能力不进 catalog，只在形态清单里声明 Drivers）。
// 同一能力在不同形态可选中不同子集（enterprise 只选 storage/local），因此这里是并集来源。
func driverDescriptors(asms map[string]assembly.Assembly) []contract.Descriptor {
	out := make([]contract.Descriptor, 0, len(catalog.All()))
	for _, d := range catalog.All() {
		if len(d.Drivers) > 0 {
			out = append(out, d)
		}
	}
	for _, name := range slices.Sorted(maps.Keys(asms)) {
		for _, c := range asms[name].Capabilities {
			if len(c.Descriptor.Drivers) > 0 {
				out = append(out, c.Descriptor)
			}
		}
	}
	return out
}

// availableDrivers 汇总 catalog 与形态清单声明的可用集：驱动包全路径 → true。
// 粒度是驱动包路径而非配置取值：queue.RegisteredTypes()/storage.RegisteredTypes() 返回的是
// 配置取值，Capability.Drivers/Descriptor.Drivers 是驱动包名，两者不可直接比集合。
func availableDrivers(asms map[string]assembly.Assembly) map[string]bool {
	out := map[string]bool{}
	for _, d := range driverDescriptors(asms) {
		for _, drv := range d.Drivers {
			out[driverPath(d.Name, drv)] = true
		}
	}
	return out
}

// selectedDrivers 汇总一个形态声明的选中集：驱动包全路径 → true。
func selectedDrivers(a assembly.Assembly) map[string]bool {
	out := map[string]bool{}
	for _, c := range a.Capabilities {
		for _, drv := range c.Drivers {
			out[driverPath(c.Descriptor.Name, drv)] = true
		}
	}
	return out
}

// driverCapabilities 返回声明了可用集的能力名（升序）：核心零驱动断言的作用域。
func driverCapabilities(descs []contract.Descriptor) []string {
	set := map[string]bool{}
	for _, d := range descs {
		set[d.Name] = true
	}
	return slices.Sorted(maps.Keys(set))
}

// checkDriverAvailability 断言①：可用集自洽 —— 驱动名非空、同一 Descriptor 内不重复，
// 且 internal/capabilities/<cap>/<driver> 目录真实存在（拼写错误在这里暴露）。
func checkDriverAvailability(root string, descs []contract.Descriptor) error {
	seen := map[string]bool{} // 驱动包路径 → 已查过目录
	for _, d := range descs {
		names := map[string]bool{}
		for _, drv := range d.Drivers {
			if drv == "" {
				return fmt.Errorf("capability %q declares an empty driver name", d.Name)
			}
			if names[drv] {
				return fmt.Errorf("capability %q declares driver %q twice", d.Name, drv)
			}
			names[drv] = true
			pkg := driverPath(d.Name, drv)
			if seen[pkg] {
				continue
			}
			seen[pkg] = true
			dir := filepath.Join(root, "internal", "capabilities", d.Name, drv)
			info, err := os.Stat(dir)
			if err != nil {
				return fmt.Errorf("capability %q declares driver %q but %s does not exist: %w",
					d.Name, drv, filepath.ToSlash(dir), err)
			}
			if !info.IsDir() {
				return fmt.Errorf("capability %q driver %q is not a directory: %s",
					d.Name, drv, filepath.ToSlash(dir))
			}
		}
	}
	return nil
}

// checkCoreClosures 断言②：有可用集的能力，其核心包（./internal/capabilities/<cap>）的
// 生产 import 闭包里不得出现任何驱动包，也不得出现重型第三方依赖 —— 否则「驱动可插拔」
// 名存实亡（核心仍把重型依赖编进闭包）。
func checkCoreClosures(root string, descs []contract.Descriptor, available map[string]bool) error {
	for _, capName := range driverCapabilities(descs) {
		closure, err := closureImports(root, "./internal/capabilities/"+capName)
		if err != nil {
			return fmt.Errorf("load closure of capability %q: %w", capName, err)
		}
		for _, pkg := range slices.Sorted(maps.Keys(closure)) {
			if available[pkg] {
				return fmt.Errorf("capability %q core imports driver package %s", capName, pkg)
			}
			if name := heavydeps.Of(pkg); name != "" {
				return fmt.Errorf("capability %q core imports heavy dependency %s (%s)", capName, name, pkg)
			}
		}
	}
	return nil
}

// checkProfileClosures 断言③④：每个形态的驱动包闭包必须与 Assembly 的选中集逐值相等，
// 且驱动包的生产 import 方只能是 internal/profiles/*。
func checkProfileClosures(root string, asms map[string]assembly.Assembly, available map[string]bool) error {
	for _, name := range slices.Sorted(maps.Keys(asms)) {
		closure, err := closureImports(root, "./profiles/"+name)
		if err != nil {
			return fmt.Errorf("load closure of profile %q: %w", name, err)
		}
		imported := map[string]bool{}
		for pkg := range closure {
			if available[pkg] {
				imported[pkg] = true
			}
		}
		if err := compareDriverSets(selectedDrivers(asms[name]), imported); err != nil {
			return fmt.Errorf("profile %q: %w", name, err)
		}
		for _, importer := range slices.Sorted(maps.Keys(closure)) {
			for _, dep := range closure[importer] {
				if !available[dep] {
					continue
				}
				if !strings.HasPrefix(importer, modulePath+"/internal/profiles/") {
					return fmt.Errorf("driver package %s is imported by %s; only %s/internal/profiles/* may import drivers",
						dep, importer, modulePath)
				}
			}
		}
	}
	return nil
}

// isCapabilityRootPackage 判断 pkg 是否为能力根包（jimu/internal/capabilities/<cap>，
// 其后不再有 "/"）：形态生产代码合法 import 的 capabilities 包有两类 —— 能力根包
// （取 Descriptor/Wire）与已声明的驱动包，其余一律视为未声明的驱动。
func isCapabilityRootPackage(pkg string) bool {
	rest, ok := strings.CutPrefix(pkg, modulePath+"/internal/capabilities/")
	return ok && !strings.Contains(rest, "/")
}

// driverImportViolation 判断形态包 importer 直接 import dep 是否违规。只看形态生产图
// （importer 前缀必须为 jimu/internal/profiles/），且只盯 capabilities 子包。
func driverImportViolation(importer, dep string, available map[string]bool) bool {
	if !strings.HasPrefix(importer, modulePath+"/internal/profiles/") {
		return false
	}
	if !strings.HasPrefix(dep, modulePath+"/internal/capabilities/") {
		return false
	}
	if isCapabilityRootPackage(dep) {
		return false
	}
	return !available[dep]
}

// checkProfileDriverImports 断言⑤：校验形态包（internal/profiles/*）生产代码的直接 import
// 只允许「能力根包」与「已声明的驱动包」。任何其它 internal/capabilities/<cap>/<sub> import
// 都视为未声明的驱动 —— 这是「驱动宇宙由已声明项推导」口径下唯一能抓住「新增驱动目录 +
// profile blank import，却忘了在 Descriptor.Drivers/Assembly Drivers 里声明」的断言：
// 其余断言对只查已声明项的集合比较、按 available 过滤的闭包交集全部不可见。
func checkProfileDriverImports(root string, names []string, available map[string]bool) error {
	for _, name := range slices.Sorted(slices.Values(names)) {
		closure, err := closureImports(root, "./internal/profiles/"+name)
		if err != nil {
			return fmt.Errorf("load closure of profile %q: %w", name, err)
		}
		for _, importer := range slices.Sorted(maps.Keys(closure)) {
			for _, dep := range closure[importer] {
				if !driverImportViolation(importer, dep, available) {
					continue
				}
				return fmt.Errorf("profile package %s imports %s, which no capability declares as a driver; "+
					"add it to Descriptor.Drivers and the profile Assembly Drivers", importer, dep)
			}
		}
	}
	return nil
}

// compareDriverSets 逐值比较声明集与 import 实际集。集合比较、不比较顺序：full 的 blank
// import 是字母序而 Assembly 声明是装配序（redis/kafka/rabbitmq），有序比较会误报。
func compareDriverSets(declared, imported map[string]bool) error {
	for _, pkg := range slices.Sorted(maps.Keys(imported)) {
		if !declared[pkg] {
			return fmt.Errorf("driver package %s is imported but not declared in the profile assembly", pkg)
		}
	}
	for _, pkg := range slices.Sorted(maps.Keys(declared)) {
		if !imported[pkg] {
			return fmt.Errorf("driver package %s is declared in the profile assembly but not imported", pkg)
		}
	}
	return nil
}

// closureImports 载入 pattern 的 import 闭包（生产包，不含测试），返回 包路径 → 直接 import 列表。
//
// Tests 显式保持 false，只统计生产图：Tests=true 时模式匹配到的**初始包**会额外载入其测试
// 变体（依赖包的测试变体本就不进闭包），于是将来 internal/capabilities/queue/*_test.go 之类
// 为测试 import 驱动包时，该 import 会混进断言② 的核心闭包，把「核心零驱动」误报成违规。
func closureImports(root, pattern string) (map[string][]string, error) {
	cfg := &packages.Config{
		Mode:  packages.NeedName | packages.NeedFiles | packages.NeedImports | packages.NeedDeps,
		Dir:   root,
		Tests: false,
	}
	pkgs, err := packages.Load(cfg, pattern)
	if err != nil {
		return nil, err
	}
	if len(pkgs) == 0 {
		return nil, fmt.Errorf("no packages matched %s", pattern)
	}

	var loadErrs []string
	out := map[string][]string{}
	packages.Visit(pkgs, func(p *packages.Package) bool {
		for _, e := range p.Errors {
			loadErrs = append(loadErrs, e.Error())
		}
		deps := make([]string, 0, len(p.Imports))
		for dep := range p.Imports {
			deps = append(deps, dep)
		}
		slices.Sort(deps)
		out[p.PkgPath] = deps
		return true
	}, nil)
	if len(loadErrs) > 0 {
		return nil, fmt.Errorf("load package graph of %s: %s", pattern, strings.Join(loadErrs, "; "))
	}
	return out, nil
}

// checkDrivers 是五条断言的入口；root 为仓库根。
func checkDrivers(root string, assemblies map[string]assembly.Assembly) error {
	descs := driverDescriptors(assemblies)
	if err := checkDriverAvailability(root, descs); err != nil {
		return err
	}
	available := availableDrivers(assemblies)
	if err := checkCoreClosures(root, descs, available); err != nil {
		return err
	}
	if err := checkProfileDriverImports(root, slices.Sorted(maps.Keys(assemblies)), available); err != nil {
		return err
	}
	return checkProfileClosures(root, assemblies, available)
}
