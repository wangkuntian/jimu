// Package catalog 维护全仓库唯一的能力清单与启用集解析。
//
// 新增能力：在 entries 中追加一行（位置必须在它的依赖之后）。
// 删除能力：删掉该行与对应目录，其余代码无需改动 —— 这是"可插拔"的中心点。
package catalog

import (
	"fmt"
	"strings"

	accessmodule "jimu/internal/capabilities/access"
	"jimu/internal/capabilities/apikey"
	auditmodule "jimu/internal/capabilities/audit"
	authmodule "jimu/internal/capabilities/auth"
	"jimu/internal/capabilities/breach"
	"jimu/internal/capabilities/captcha"
	consolemodule "jimu/internal/capabilities/console"
	"jimu/internal/capabilities/dataops"
	"jimu/internal/capabilities/feature"
	mfamodule "jimu/internal/capabilities/mfa"
	oauthmodule "jimu/internal/capabilities/oauth"
	"jimu/internal/capabilities/outbox"
	passkeymodule "jimu/internal/capabilities/passkey"
	"jimu/internal/capabilities/queue"
	"jimu/internal/capabilities/search"
	tenantmodule "jimu/internal/capabilities/tenant"
	"jimu/internal/capabilities/uploadsec"
	"jimu/internal/capabilities/user"
	"jimu/internal/contract"
)

// entries 是唯一的能力清单，顺序即默认启用顺序（同时是依赖拓扑序）。
// 尾部基础设施能力（breach/captcha 等）无 Requires；breach 无 Module 实例，
// 仅携带声明与端口实现；captcha 本轮起有实例并自挂公开路由。
var entries = []contract.Descriptor{
	user.Descriptor,
	accessmodule.Descriptor,
	tenantmodule.Descriptor,
	mfamodule.Descriptor,
	authmodule.Descriptor,
	passkeymodule.Descriptor,
	auditmodule.Descriptor,
	consolemodule.Descriptor,
	oauthmodule.Descriptor,
	apikey.Descriptor,
	queue.Descriptor,
	outbox.Descriptor,
	dataops.Descriptor,
	search.Descriptor,
	captcha.Descriptor,
	feature.Descriptor,
	uploadsec.Descriptor,
	breach.Descriptor,
}

// ValidateDeclarations 只校验 SoftRequires 的结构：必须是清单内能力名、不得自引用、
// 不得与 Requires 重叠、不得重复。Owns 与迁移的一致性不在此处，由门禁
// `make check-capabilities`（tools/checkcapabilities）负责。
// 依赖图无环与「清单顺序满足依赖在前」由 TestDescriptorsAreWellFormed 钉住。
func ValidateDeclarations() error {
	known := make(map[string]bool, len(entries))
	for _, d := range entries {
		known[d.Name] = true
	}
	for _, d := range entries {
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

// All 返回清单中全部能力的深拷贝（含 Requires/SoftRequires/Owns/Permissions/Configs），
// 调用方修改不影响清单。
func All() []contract.Descriptor {
	out := make([]contract.Descriptor, len(entries))
	for i, d := range entries {
		out[i] = d
		out[i].Requires = append([]string(nil), d.Requires...)
		out[i].SoftRequires = append([]string(nil), d.SoftRequires...)
		out[i].Owns = append([]string(nil), d.Owns...)
		out[i].Permissions = append([]contract.Permission(nil), d.Permissions...)
		out[i].Configs = append([]contract.ConfigSpec(nil), d.Configs...)
	}
	return out
}

// Names 返回清单中的能力名，按清单顺序。
func Names() []string {
	out := make([]string, 0, len(entries))
	for _, d := range entries {
		out = append(out, d.Name)
	}
	return out
}

// Degradation 描述一个能力**声明**的可选依赖（SoftRequires）不在已解析启用集里。
// 这是声明层的报告，不是对运行时装配的观测。
type Degradation struct {
	Capability string   `json:"capability"`
	Missing    []string `json:"missing"`
}

// Degraded 返回已解析启用集里被声明但缺失的软依赖：SoftRequires 中不在集合内的目标。
// 软依赖不会自动补齐（设计 §6.4），缺失只降级、不报错。
//
// 这是**静态的组合根声明层**报告，只比对 Descriptor 声明，不观测运行时装配。当前
// 组合根（cmd/server/main.go、internal/app/container.go）仍无条件注入多数依赖，故在
// 组合根改为按启用集驱动（P1 显式 Deps）之前本报告可能多报：声明缺失不等于该能力的
// 可选组件没有被注入。
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

// Resolve 解析启用集：enabled 为空表示全部启用（向后兼容默认配置）；
// 未知能力报错；硬依赖自动补齐闭包；返回结果按清单顺序排列且为深拷贝
// （含 Requires，调用方修改不影响清单）；依赖缺失时按清单顺序报出第一个
// 违规能力，保证错误文案确定，不随 map 遍历顺序变化。
func Resolve(enabled []string) ([]contract.Descriptor, error) {
	if err := ValidateDeclarations(); err != nil {
		return nil, err
	}
	if len(enabled) == 0 {
		return All(), nil
	}
	byName := make(map[string]contract.Descriptor, len(entries))
	for _, d := range entries {
		byName[d.Name] = d
	}
	on := make(map[string]bool, len(enabled))
	for _, name := range enabled {
		if _, ok := byName[name]; !ok {
			return nil, fmt.Errorf("unknown capability %q, available: %s", name, strings.Join(Names(), ", "))
		}
		on[name] = true
	}
	// 依赖闭包：反复补齐直到不再变化，保证传递依赖也被纳入。
	// 按清单顺序遍历（而非 map）以保证错误文案确定：多个依赖缺失时
	// 始终报出清单顺序里的第一个。
	for changed := true; changed; {
		changed = false
		for _, d := range entries {
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
	for _, d := range entries {
		if on[d.Name] {
			d.Requires = append([]string(nil), d.Requires...)
			d.SoftRequires = append([]string(nil), d.SoftRequires...)
			d.Owns = append([]string(nil), d.Owns...)
			d.Permissions = append([]contract.Permission(nil), d.Permissions...)
			d.Configs = append([]contract.ConfigSpec(nil), d.Configs...)
			out = append(out, d)
		}
	}
	return out, nil
}
