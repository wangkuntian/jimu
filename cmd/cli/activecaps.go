package main

import (
	"errors"
	"fmt"

	"jimu/internal/app"

	"jimu/internal/capabilities/catalog"
	"jimu/internal/contract"
	"jimu/internal/profiles/active"
)

// errNoCatalogCapabilities 形态声明的能力里没有任何 catalog 条目：fail-closed。
// 静默「迁移零个能力」比报错危险（err113 要求错误静态，故用哨兵 + 包装）。
var errNoCatalogCapabilities = errors.New("profile declares no catalog capabilities")

// errSeedNeedsTenant 结构种子需要 tenant 能力（见 checkSeedCapabilities）。
var errSeedNeedsTenant = errors.New("structural seed needs the tenant capability")

// activeDescriptors 返回当前形态（编译期选点，默认 full）声明的能力描述符。
//
// **必须保持 catalog 的拓扑序**：迁移有真实依赖（如 tenant 的迁移 ALTER users/roles，
// 必须排在 user/access 之后），而 Assembly().Capabilities 是**装配顺序**（端口提供者在前），
// 不能直接拿来跑迁移。做法是从 catalog.All() 过滤出形态声明的能力名 —— 顺序天然保持。
// 非 catalog（Ungated）条目一律没有迁移与权限点，因此这个过滤不会丢东西。
//
// `capabilities.enabled`（层③）**不参与**：关闭能力不删表，迁移跟随的是**编译期形态**。
func activeDescriptors() ([]contract.Descriptor, error) {
	a := active.Assembly()
	declared := make(map[string]bool, len(a.Capabilities))
	for _, c := range a.Capabilities {
		declared[c.Descriptor.Name] = true
	}
	return resolveActive(a.Name, declared)
}

// resolveActive 是 activeDescriptors 的纯函数部分（便于单测）：过滤 + 空集 fail-closed。
func resolveActive(profile string, declared map[string]bool) ([]contract.Descriptor, error) {
	caps := filterCatalog(catalog.All(), declared)
	if len(caps) == 0 {
		return nil, fmt.Errorf("%w: %q", errNoCatalogCapabilities, profile)
	}
	// 更细一层：有 catalog 能力但一个迁移都没有 → 同样是「迁移零个能力」，必须报错。
	hasMigrations := false
	for _, d := range caps {
		if d.Migrations != nil {
			hasMigrations = true
			break
		}
	}
	if !hasMigrations {
		return nil, fmt.Errorf("%w: %q has no capability with migrations", errNoCatalogCapabilities, profile)
	}
	return caps, nil
}

// checkSeedCapabilities 结构种子需要 tenant 能力：tenants/tenant_plans 表与 users/roles 的
// tenant_id 列都由 tenant 能力的迁移产生，而 ORM 模型始终写 tenant_id 列 —— 不含该能力的
// 形态无法播种（P2.6 迁移裁剪后尤其如此）。这里显式拒绝并给出替代路径，
// 而不是让种子在事务中途抛 "no such table: tenants"。
func checkSeedCapabilities(profile string, caps []contract.Descriptor) error {
	if app.HasCapability(caps, "tenant") {
		return nil
	}
	return fmt.Errorf("%w: profile %q has no tenant capability; migrate and seed with a tenant-capable shape (full/saas) — the database must be migrated by that shape first", errSeedNeedsTenant, profile)
}

// filterCatalog 保留 catalog 顺序，只留下 declared 里的能力。
func filterCatalog(all []contract.Descriptor, declared map[string]bool) []contract.Descriptor {
	out := make([]contract.Descriptor, 0, len(all))
	for _, d := range all {
		if declared[d.Name] {
			out = append(out, d)
		}
	}
	return out
}
