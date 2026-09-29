package tenant

import (
	"testing"

	authmodule "jimu/internal/capabilities/auth"

	"github.com/stretchr/testify/assert"
)

// TestProvisioningConfigMapping auth 段的 provisioning 映射为 tenant 自有视图
// （tenant 不得 import auth 的类型混用，两边类型独立，字段必须逐一保留）。
func TestProvisioningConfigMapping(t *testing.T) {
	got := provisioningConfig(&authmodule.Config{Provisioning: authmodule.ProvisioningConfig{
		Enabled:   true,
		OwnerRole: "管理员",
		Roles: []authmodule.ProvisionRoleTemplate{{
			Name:        "管理员",
			Description: "租户管理员",
			Permissions: []authmodule.ProvisionPermission{{Resource: "/api/v1/users", Action: "GET"}},
		}},
	}})

	assert.Equal(t, ProvisioningConfig{
		Enabled:   true,
		OwnerRole: "管理员",
		Roles: []ProvisionRoleTemplate{{
			Name:        "管理员",
			Description: "租户管理员",
			Permissions: []ProvisionPermission{{Resource: "/api/v1/users", Action: "GET"}},
		}},
	}, got)
}

// TestProvisioningConfigEmpty 未配置 provisioning 时映射为空（tenant 不建 provisioner）。
func TestProvisioningConfigEmpty(t *testing.T) {
	got := provisioningConfig(&authmodule.Config{})
	assert.False(t, got.Enabled)
	assert.Empty(t, got.Roles)
}
