package catalog

import (
	"jimu/internal/contract"
)

// MigrationSchemaDeps 是「迁移集必须一并带上」的 schema 依赖：某些能力的迁移会修改**其他**
// 能力拥有的表，只按形态声明集过滤会让裁剪形态建出**自己写不进去**的 schema。
//
// 事实源是各能力的 Descriptor.MigrationSchemaDeps；本表只是按能力名索引的派生视图，
// 保留给迁移集计算与生成模板使用。当前 user/access 依赖 tenant 的 005 迁移添加
// tenant_id 列。只影响建表：装配集（形态清单）不受影响，不挂 tenant 路由、不额外 seed 数据。
var MigrationSchemaDeps = collectMigrationSchemaDeps(entries)

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

func collectMigrationSchemaDeps(descriptors []contract.Descriptor) map[string][]string {
	out := map[string][]string{}
	for _, descriptor := range descriptors {
		if len(descriptor.MigrationSchemaDeps) == 0 {
			continue
		}
		out[descriptor.Name] = append([]string(nil), descriptor.MigrationSchemaDeps...)
	}
	return out
}
