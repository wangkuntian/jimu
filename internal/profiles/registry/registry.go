// Package registry 汇总全部形态清单，作为「形态有哪些」的唯一来源，供 tools/* 使用。
//
// 它 import 全部 5 个形态包，因此**绝不能被 cmd/server 或 internal/profiles/active 引用**
// （否则所有形态都会被拉回二进制，层②裁剪失效）；这一条由 make check-capabilities 断言。
package registry

import (
	"fmt"
	"strings"

	"jimu/internal/assembly"
	"jimu/internal/profiles/enterprise"
	"jimu/internal/profiles/full"
	"jimu/internal/profiles/machine"
	"jimu/internal/profiles/minimal"
	"jimu/internal/profiles/saas"
)

// names 固定顺序：报告行序、门禁与脚本的遍历顺序都依赖它。
var names = []string{"full", "minimal", "saas", "enterprise", "machine"}

// Names 返回全部形态名（按固定顺序）。
func Names() []string { return append([]string(nil), names...) }

// All 返回形态名 → 装配清单。
func All() map[string]assembly.Assembly {
	return map[string]assembly.Assembly{
		"full":       full.Assembly(),
		"minimal":    minimal.Assembly(),
		"saas":       saas.Assembly(),
		"enterprise": enterprise.Assembly(),
		"machine":    machine.Assembly(),
	}
}

// Lookup 按名取装配清单；未知形态返回错误并列出可用形态。
func Lookup(name string) (assembly.Assembly, error) {
	if a, ok := All()[name]; ok {
		return a, nil
	}
	return assembly.Assembly{}, fmt.Errorf("unknown profile %q (available: %s)", name, strings.Join(names, ", "))
}
