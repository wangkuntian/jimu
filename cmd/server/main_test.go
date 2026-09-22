package main

import (
	"slices"
	"testing"

	authmodule "jimu/internal/capabilities/auth"
	"jimu/internal/capabilities/catalog"
	tenantmodule "jimu/internal/capabilities/tenant"
	"jimu/internal/capability"
	"jimu/internal/contract"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestFullAssemblyCoversCatalog 过渡形态的能力清单必须与 catalog 的 18 项逐名一致
// （顺序刻意不同：tenant/access 必须先于 user 才能经端口提供角色分配与配额），
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
	want := catalog.Names()
	slices.Sort(got)
	slices.Sort(want)
	require.Equal(t, want, got, "过渡形态的能力清单必须与 catalog 逐名一致")
}

// TestFullAssemblyResolvesToCatalogDefault 默认配置（capabilities.enabled 为空）下，
// 过渡形态解析出的启用集必须与 catalog.Resolve(nil) 逐名一致 —— full 零退化的第一道护栏。
func TestFullAssemblyResolvesToCatalogDefault(t *testing.T) {
	a := fullAssembly()
	descriptors := make([]contract.Descriptor, 0, len(a.Capabilities))
	for _, c := range a.Capabilities {
		descriptors = append(descriptors, c.Descriptor)
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

// TestFullAssemblyOrder 钉住过渡形态的装配顺序：tenant/access 必须先于 user（提供
// 角色分配/配额端口），captcha/mfa 必须先于 auth（提供验证码/MFA 端口），其余保持
// catalog 的相对顺序。Task 4 的 profile 清单会对齐 catalog 顺序，届时同步更新本断言。
func TestFullAssemblyOrder(t *testing.T) {
	want := []string{
		"tenant", "access", "user", "captcha", "mfa", "auth", "passkey", "audit",
		"console", "oauth", "apikey", "queue", "dataops", "feature", "uploadsec",
		"outbox", "search", "breach",
	}
	got := make([]string, 0, len(want))
	for _, c := range fullAssembly().Capabilities {
		got = append(got, c.Descriptor.Name)
	}
	require.Equal(t, want, got)
}

// TestFullAssemblyHasNoDuplicates 清单内不得重名。
func TestFullAssemblyHasNoDuplicates(t *testing.T) {
	seen := map[string]bool{}
	for _, c := range fullAssembly().Capabilities {
		require.False(t, seen[c.Descriptor.Name], "duplicate capability %q", c.Descriptor.Name)
		seen[c.Descriptor.Name] = true
	}
}

// TestNoModuleWireReturnsNoInstance 空 Wire 表示「无 Module 实例」而不是错误
// （outbox/search/breach 只携带声明、参与迁移）。
func TestNoModuleWireReturnsNoInstance(t *testing.T) {
	mod, err := noModule(nil)
	require.NoError(t, err)
	assert.Nil(t, mod)
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
