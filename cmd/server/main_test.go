package main

import (
	"slices"
	"testing"

	"jimu/internal/assembly"
	authmodule "jimu/internal/capabilities/auth"
	"jimu/internal/capabilities/catalog"
	tenantmodule "jimu/internal/capabilities/tenant"
	"jimu/internal/capability"
	"jimu/internal/contract"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// nonCatalogCapabilities 是 full 形态中的非 catalog 条目（P2.4 裁定 7）：它们是
// assembly.Capability 但不是 catalog 成员（catalog 仍 18 项，不入迁移/权限聚合）。
var nonCatalogCapabilities = []string{"encryption", "storage", "notification", "retention", "ws", "grpc", "apidocs"}

// TestFullAssemblyCoversCatalog 过渡形态的能力清单 = catalog 全量 ∪ 固定非 catalog 条目，
// 且每一项都必须给出 Wire（无 Module 实例的能力给出空 Wire）。
func TestFullAssemblyCoversCatalog(t *testing.T) {
	a := fullAssembly()
	require.Equal(t, "full", a.Name)
	require.Equal(t, version, a.Version, "构建版本必须传给驱动（console 状态页依赖它）")

	got := make([]string, 0, len(a.Capabilities))
	for _, c := range a.Capabilities {
		require.NotNil(t, c.Wire, "capability %q has no Wire", c.Descriptor.Name)
		got = append(got, c.Descriptor.Name)
	}
	want := append(catalog.Names(), nonCatalogCapabilities...)
	slices.Sort(got)
	slices.Sort(want)
	require.Equal(t, want, got, "过渡形态的能力清单必须 = catalog 全量 ∪ 非 catalog 条目")
}

// TestFullAssemblyResolvesToCatalogDefault 默认配置（capabilities.enabled 为空）下，
// 过渡形态的 catalog 能力解析集必须与 catalog.Resolve(nil) 逐名一致 —— full 零退化的第一道护栏。
func TestFullAssemblyResolvesToCatalogDefault(t *testing.T) {
	a := fullAssembly()
	known := map[string]bool{}
	for _, n := range catalog.Names() {
		known[n] = true
	}
	descriptors := make([]contract.Descriptor, 0, len(a.Capabilities))
	for _, c := range a.Capabilities {
		if known[c.Descriptor.Name] {
			descriptors = append(descriptors, c.Descriptor)
		}
	}

	got, err := capability.Resolve(descriptors, nil)
	require.NoError(t, err)

	fromCatalog, err := catalog.Resolve(nil)
	require.NoError(t, err)
	require.ElementsMatch(t, descriptorNames(fromCatalog), descriptorNames(got))
}

// TestFullAssemblyDeclarationsAreWellFormed 过渡形态的声明必须通过全量清单同款校验
// （软依赖不得是错别字），否则 profile 化后会带着声明缺陷上线。
func TestFullAssemblyDeclarationsAreWellFormed(t *testing.T) {
	a := fullAssembly()
	descriptors := make([]contract.Descriptor, 0, len(a.Capabilities))
	for _, c := range a.Capabilities {
		descriptors = append(descriptors, c.Descriptor)
	}
	require.NoError(t, capability.ValidateDeclarations(descriptors, catalog.Names()))
}

// TestFullAssemblyOrder 钉住过渡形态的装配顺序：提供端口的能力必须排在消费它的能力
// 之前（encryption/storage/notification/queue/outbox/breach 先于 tenant/user/auth/
// uploadsec/grpc；tenant/access 先于 user；captcha/mfa 先于 auth）。Task 4 的 profile
// 清单会接手这份顺序。
func TestFullAssemblyOrder(t *testing.T) {
	want := []string{
		"encryption", "storage", "notification", "queue", "outbox", "breach",
		"tenant", "access", "user", "captcha", "mfa", "auth", "passkey", "audit",
		"console", "oauth", "apikey", "dataops", "feature", "uploadsec", "search",
		"retention", "apidocs", "grpc", "ws",
	}
	got := make([]string, 0, len(want))
	for _, c := range fullAssembly().Capabilities {
		got = append(got, c.Descriptor.Name)
	}
	require.Equal(t, want, got)
}

// TestFullAssemblyPortFlow 装配顺序护栏：full 清单里每个被 Wire 读取的端口都必须由内核
// 桥接端口或排在其前的能力提供。过渡期的内联闭包无法静态内省，故该用例真实试运行各
// Wire 并观察 Provide/Port 调用（零值内核件、只读配置段，不连库）。删掉 wireAccess 的
// Provide("access", …) 或把 access 排到 user 之后都会让本用例失败。
func TestFullAssemblyPortFlow(t *testing.T) {
	require.NoError(t, assembly.ValidatePortFlow(fullAssembly()))
}

// TestFullAssemblyHasNoDuplicates 清单内不得重名。
func TestFullAssemblyHasNoDuplicates(t *testing.T) {
	seen := map[string]bool{}
	for _, c := range fullAssembly().Capabilities {
		require.False(t, seen[c.Descriptor.Name], "duplicate capability %q", c.Descriptor.Name)
		seen[c.Descriptor.Name] = true
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

func descriptorNames(ds []contract.Descriptor) []string {
	out := make([]string, 0, len(ds))
	for _, d := range ds {
		out = append(out, d.Name)
	}
	return out
}
