// Package catalog 维护全仓库唯一的能力清单与启用集解析。
//
// 新增能力：在 entries 中追加一行（位置必须在它的依赖之后）。
// 删除能力：删掉该行与对应目录，其余代码无需改动 —— 这是"可插拔"的中心点。
package catalog

import (
	"fmt"
	"strings"

	adminmodule "jimu/internal/capabilities/admin"
	auditmodule "jimu/internal/capabilities/audit"
	authmodule "jimu/internal/capabilities/auth"
	oauthmodule "jimu/internal/capabilities/oauth"
	"jimu/internal/capabilities/permission"
	"jimu/internal/capabilities/role"
	tenantmodule "jimu/internal/capabilities/tenant"
	"jimu/internal/capabilities/user"
	"jimu/internal/contract"
)

// entries 是唯一的能力清单，顺序即默认启用顺序（同时是依赖拓扑序）。
var entries = []contract.Descriptor{
	user.Descriptor,
	role.Descriptor,
	permission.Descriptor,
	tenantmodule.Descriptor,
	authmodule.Descriptor,
	auditmodule.Descriptor,
	adminmodule.Descriptor,
	oauthmodule.Descriptor,
}

// All 返回清单中全部能力的深拷贝（含 Requires），调用方修改不影响清单。
func All() []contract.Descriptor {
	out := make([]contract.Descriptor, len(entries))
	for i, d := range entries {
		out[i] = d
		out[i].Requires = append([]string(nil), d.Requires...)
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

// Resolve 解析启用集：enabled 为空表示全部启用（向后兼容默认配置）；
// 未知能力报错；硬依赖自动补齐闭包；返回结果按清单顺序排列且为深拷贝
// （含 Requires，调用方修改不影响清单）；依赖缺失时按清单顺序报出第一个
// 违规能力，保证错误文案确定，不随 map 遍历顺序变化。
func Resolve(enabled []string) ([]contract.Descriptor, error) {
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
			out = append(out, d)
		}
	}
	return out, nil
}
