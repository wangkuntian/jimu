// Package catalog 维护全仓库唯一的能力清单与启用集解析。
//
// 新增能力：在 entries 中追加一行（位置必须在它的依赖之后）。
// 删除能力：删掉该行与对应目录，其余代码无需改动 —— 这是"可插拔"的中心点。
package catalog

import (
	"fmt"
	"strings"

	"jimu/internal/contract"
	adminmodule "jimu/internal/modules/admin"
	auditmodule "jimu/internal/modules/audit"
	authmodule "jimu/internal/modules/auth"
	oauthmodule "jimu/internal/modules/oauth"
	"jimu/internal/modules/permission"
	"jimu/internal/modules/role"
	tenantmodule "jimu/internal/modules/tenant"
	"jimu/internal/modules/user"
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

// All 返回清单中的全部能力描述（浅拷贝，调用方增删元素不影响清单；
// 但 Descriptor.Requires 切片仍与包级描述符共享底层数组，勿原地修改其元素）。
func All() []contract.Descriptor {
	return append([]contract.Descriptor(nil), entries...)
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
// 未知能力报错；硬依赖自动补齐闭包；返回结果按清单顺序排列。
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
	for changed := true; changed; {
		changed = false
		for name := range on {
			for _, dep := range byName[name].Requires {
				if _, ok := byName[dep]; !ok {
					return nil, fmt.Errorf("capability %q requires unknown capability %q", name, dep)
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
			out = append(out, d)
		}
	}
	return out, nil
}
