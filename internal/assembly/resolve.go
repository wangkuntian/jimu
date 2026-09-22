// Package assembly 承载组合根的装配原语：把能力描述符清单解析成启用集。
//
// 本包只依赖 internal/contract 与标准库 —— 不 import 任何能力包或内核包 —— 因此
// profile 入口包（profiles/<name>）可以只带上自己的能力子集调用这里的算法。
package assembly

import (
	"fmt"
	"strings"

	"jimu/internal/contract"
)

// Degradation 描述一个能力**声明**的可选依赖（SoftRequires）不在已解析启用集里。
// 这是声明层的报告，不是对运行时装配的观测。
type Degradation struct {
	Capability string   `json:"capability"`
	Missing    []string `json:"missing"`
}

// ValidateDeclarations 只校验 SoftRequires 的结构：必须是清单内能力名、不得自引用、
// 不得与 Requires 重叠、不得重复。Owns 与迁移的一致性不在此处，由门禁
// `make check-capabilities`（tools/checkcapabilities）负责。
// 依赖图无环与「清单顺序满足依赖在前」由 catalog 的 TestDescriptorsAreWellFormed 钉住。
func ValidateDeclarations(caps []contract.Descriptor) error {
	known := make(map[string]bool, len(caps))
	for _, d := range caps {
		known[d.Name] = true
	}
	for _, d := range caps {
		hard := make(map[string]bool, len(d.Requires))
		for _, dep := range d.Requires {
			hard[dep] = true
		}
		seen := make(map[string]bool, len(d.SoftRequires))
		for _, dep := range d.SoftRequires {
			if !known[dep] {
				return fmt.Errorf("capability %q soft-requires unknown capability %q", d.Name, dep)
			}
			if dep == d.Name {
				return fmt.Errorf("capability %q soft-requires itself", d.Name)
			}
			if hard[dep] {
				return fmt.Errorf("capability %q declares %q in both Requires and SoftRequires", d.Name, dep)
			}
			if seen[dep] {
				return fmt.Errorf("capability %q duplicates soft requirement %q", d.Name, dep)
			}
			seen[dep] = true
		}
	}
	return nil
}

// Resolve 解析启用集：enabled 为空表示全部启用（向后兼容默认配置）；
// 未知能力报错；硬依赖自动补齐闭包；返回结果按清单顺序排列且为深拷贝
// （含 Requires，调用方修改不影响清单）；依赖缺失时按清单顺序报出第一个
// 违规能力，保证错误文案确定，不随 map 遍历顺序变化。
func Resolve(caps []contract.Descriptor, enabled []string) ([]contract.Descriptor, error) {
	if err := ValidateDeclarations(caps); err != nil {
		return nil, err
	}
	if len(enabled) == 0 {
		return deepCopy(caps), nil
	}
	byName := make(map[string]contract.Descriptor, len(caps))
	for _, d := range caps {
		byName[d.Name] = d
	}
	on := make(map[string]bool, len(enabled))
	for _, name := range enabled {
		if _, ok := byName[name]; !ok {
			return nil, fmt.Errorf("unknown capability %q, available: %s", name, strings.Join(names(caps), ", "))
		}
		on[name] = true
	}
	// 依赖闭包：反复补齐直到不再变化，保证传递依赖也被纳入。
	// 按清单顺序遍历（而非 map）以保证错误文案确定：多个依赖缺失时
	// 始终报出清单顺序里的第一个。
	for changed := true; changed; {
		changed = false
		for _, d := range caps {
			if !on[d.Name] {
				continue
			}
			for _, dep := range d.Requires {
				if _, ok := byName[dep]; !ok {
					return nil, fmt.Errorf("capability %q requires unknown capability %q", d.Name, dep)
				}
				if !on[dep] {
					on[dep] = true
					changed = true
				}
			}
		}
	}
	out := make([]contract.Descriptor, 0, len(on))
	for _, d := range caps {
		if on[d.Name] {
			out = append(out, deepCopyDescriptor(d))
		}
	}
	return out, nil
}

// Degraded 返回已解析启用集里被声明但缺失的软依赖：SoftRequires 中不在集合内的目标。
// 软依赖不会自动补齐（设计 §6.4），缺失只降级、不报错。
//
// 这是**静态的组合根声明层**报告，只比对 Descriptor 声明，不观测运行时装配。组合根
// 仍无条件注入多数依赖时本报告可能多报：声明缺失不等于该能力的可选组件没有被注入。
func Degraded(caps []contract.Descriptor) []Degradation {
	present := make(map[string]bool, len(caps))
	for _, d := range caps {
		present[d.Name] = true
	}
	out := make([]Degradation, 0, len(caps))
	for _, d := range caps {
		var missing []string
		for _, dep := range d.SoftRequires {
			if !present[dep] {
				missing = append(missing, dep)
			}
		}
		if len(missing) > 0 {
			out = append(out, Degradation{Capability: d.Name, Missing: missing})
		}
	}
	return out
}

// names 返回清单中的能力名，按清单顺序。
func names(caps []contract.Descriptor) []string {
	out := make([]string, 0, len(caps))
	for _, d := range caps {
		out = append(out, d.Name)
	}
	return out
}

// deepCopy 返回清单的深拷贝（含 Requires/SoftRequires/Owns/Permissions/Configs），
// 调用方修改不影响原清单。
func deepCopy(caps []contract.Descriptor) []contract.Descriptor {
	out := make([]contract.Descriptor, len(caps))
	for i, d := range caps {
		out[i] = deepCopyDescriptor(d)
	}
	return out
}

func deepCopyDescriptor(d contract.Descriptor) contract.Descriptor {
	d.Requires = append([]string(nil), d.Requires...)
	d.SoftRequires = append([]string(nil), d.SoftRequires...)
	d.Owns = append([]string(nil), d.Owns...)
	d.Permissions = append([]contract.Permission(nil), d.Permissions...)
	d.Configs = append([]contract.ConfigSpec(nil), d.Configs...)
	return d
}
