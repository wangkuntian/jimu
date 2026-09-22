package main

import (
	"slices"
	"testing"

	authmodule "jimu/internal/capabilities/auth"
	"jimu/internal/capabilities/catalog"
	tenantmodule "jimu/internal/capabilities/tenant"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestWiredCapabilitiesSubsetOfCatalog main 装配名册（15 个能力）必须是清单（18 项）的子集：
// 名册含 queue/apikey/dataops 等有 Module 实例的基础设施能力；不在名册中的是
// outbox/search/breach 等无实例能力（run() 按 catalog.Resolve 结果过滤装配）。
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

// TestAssemblyFilterYieldsWiredOnly run() 按 wiredCapabilities 过滤
// catalog.Resolve 结果后进入 Bootstrap 的模块应恰好是名册本身。
func TestAssemblyFilterYieldsWiredOnly(t *testing.T) {
	caps, err := catalog.Resolve(nil)
	if err != nil {
		t.Fatalf("resolve all capabilities: %v", err)
	}

	wired := make(map[string]bool, len(wiredCapabilities))
	for _, name := range wiredCapabilities {
		wired[name] = true
	}

	var modules []string
	for _, d := range caps {
		if wired[d.Name] {
			modules = append(modules, d.Name)
		}
	}

	if len(modules) != len(wiredCapabilities) {
		t.Fatalf("filtered modules %v (%d) != wiredCapabilities %v (%d)",
			modules, len(modules), wiredCapabilities, len(wiredCapabilities))
	}
	slices.Sort(modules)
	expected := slices.Clone(wiredCapabilities)
	slices.Sort(expected)
	if !slices.Equal(modules, expected) {
		t.Fatalf("filtered modules %v != wiredCapabilities %v", modules, expected)
	}
}

// TestValidateAuthConfigProvisioningRequiresPublicRegistration 组合根承担 auth 段的
// 跨字段校验：开通式注册必须同时开启公开注册（原 config.validateCommon 语义）。
func TestValidateAuthConfigProvisioningRequiresPublicRegistration(t *testing.T) {
	err := validateAuthConfig(&authmodule.Config{
		Provisioning: authmodule.ProvisioningConfig{Enabled: true},
	})
	require.ErrorIs(t, err, errProvisioningRequiresPublicRegistration)

	require.NoError(t, validateAuthConfig(&authmodule.Config{
		PublicRegistration: true,
		Provisioning:       authmodule.ProvisioningConfig{Enabled: true},
	}))
	require.NoError(t, validateAuthConfig(&authmodule.Config{}))
}

// TestTenantProvisioningConfigMapping 组合根把 auth 段 provisioning 映射为 tenant 自有视图
// （tenant 不得 import auth，两边类型独立，字段必须逐一保留）。
func TestTenantProvisioningConfigMapping(t *testing.T) {
	got := tenantProvisioningConfig(authmodule.ProvisioningConfig{
		Enabled:   true,
		OwnerRole: "管理员",
		Roles: []authmodule.ProvisionRoleTemplate{{
			Name:        "管理员",
			Description: "租户管理员",
			Permissions: []authmodule.ProvisionPermission{{Resource: "/api/v1/users", Action: "GET"}},
		}},
	})

	assert.Equal(t, tenantmodule.ProvisioningConfig{
		Enabled:   true,
		OwnerRole: "管理员",
		Roles: []tenantmodule.ProvisionRoleTemplate{{
			Name:        "管理员",
			Description: "租户管理员",
			Permissions: []tenantmodule.ProvisionPermission{{Resource: "/api/v1/users", Action: "GET"}},
		}},
	}, got)
}

// TestTenantProvisioningConfigEmpty 未配置 provisioning 时映射为空（tenant 不建 provisioner）。
func TestTenantProvisioningConfigEmpty(t *testing.T) {
	got := tenantProvisioningConfig(authmodule.ProvisioningConfig{})
	assert.False(t, got.Enabled)
	assert.Empty(t, got.Roles)
}
