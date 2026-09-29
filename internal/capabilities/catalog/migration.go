package catalog

import (
	"jimu/internal/contract"
)

// MigrationSchemaDeps 是「迁移集必须一并带上」的 schema 依赖：某些能力的迁移会修改**其他**
// 能力拥有的表，只按形态声明集过滤会让裁剪形态建出**自己写不进去**的 schema。
//
// 当前唯一一条：tenant 的迁移 005 给 user/access 拥有的 users/roles 加 tenant_id 列，
// 而这两个能力的 ORM 模型（user/domain.User、access/domain.Role）始终写该列 ——
// 因此迁移集里只要有 user 或 access，就必须带上 tenant 的迁移。
// 只影响建表：装配集（形态清单）不受影响，不挂 tenant 路由、不额外 seed 数据。
var MigrationSchemaDeps = map[string][]string{
	"user":   {"tenant"},
	"access": {"tenant"},
}

// MigrationSet 返回某形态的**迁移集**（按 catalog 拓扑序）：声明的能力名集合 ∪ schema 依赖。
//
// 这是 CLI（`jimu migrate`/`seed`）与 compose-report 的**同一口径**：报告里的「迁移数/表数」
// 必须与 `PROFILE=<name> jimu migrate` 实际执行的迁移一致，否则两者会各说各话。
func MigrationSet(declared map[string]bool) []contract.Descriptor {
	return filterAll(WithMigrationSchemaDeps(declared))
}

// WithMigrationSchemaDeps 在给定能力名集合上补齐 schema 依赖（返回新集合，不改入参）。
func WithMigrationSchemaDeps(declared map[string]bool) map[string]bool {
	out := make(map[string]bool, len(declared))
	for name := range declared {
		out[name] = true
	}
	for name := range declared {
		for _, dep := range MigrationSchemaDeps[name] {
			out[dep] = true
		}
	}
	return out
}

// filterAll 保留 catalog 拓扑序，只留下 declared 里的能力。
func filterAll(declared map[string]bool) []contract.Descriptor {
	out := make([]contract.Descriptor, 0, len(declared))
	for _, d := range All() {
		if declared[d.Name] {
			out = append(out, d)
		}
	}
	return out
}
