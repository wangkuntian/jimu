package main

import (
	"slices"
	"testing"

	"jimu/internal/capabilities/catalog"
)

// TestWiredCapabilitiesSubsetOfCatalog main 装配的 8 个能力必须是清单的子集：
// 清单尾部的基础设施能力（apikey/queue/outbox/dataops/search）只带迁移、
// 尚无 Module 实例，故不在装配名册中（run() 按 catalog.Resolve 结果过滤装配）。
func TestWiredCapabilitiesSubsetOfCatalog(t *testing.T) {
	known := catalog.Names()
	for _, name := range wiredCapabilities {
		if !slices.Contains(known, name) {
			t.Fatalf("wired capability %q missing from catalog (%v)", name, known)
		}
	}
}

func TestWiredCapabilitiesHaveNoDuplicates(t *testing.T) {
	seen := map[string]bool{}
	for _, name := range wiredCapabilities {
		if seen[name] {
			t.Fatalf("duplicate wired capability %q", name)
		}
		seen[name] = true
	}
}
