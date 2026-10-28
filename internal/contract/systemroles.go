package contract

import "strings"

type RoleScope string

const (
	RoleScopePlatform RoleScope = "platform"
	RoleScopeTenant   RoleScope = "tenant"
)

const (
	RoleCodePlatformSuperAdmin = "platform_super_admin"
	RoleCodePlatformAuditor    = "platform_auditor"
	RoleCodeTenantAdmin        = "tenant_admin"
	RoleCodeTenantMember       = "tenant_member"
)

// SystemRoleSpec 是 seed 与租户开通共享的系统角色定义。
type SystemRoleSpec struct {
	Code           string
	Name           string
	Description    string
	Scope          RoleScope
	AllPermissions bool
	Permissions    []ProvisioningPermission
}

func PlatformSystemRoles() []SystemRoleSpec {
	return []SystemRoleSpec{
		{
			Code:           RoleCodePlatformSuperAdmin,
			Name:           "平台超级管理员",
			Description:    "拥有平台全部权限",
			Scope:          RoleScopePlatform,
			AllPermissions: true,
		},
		{
			Code:        RoleCodePlatformAuditor,
			Name:        "平台审计员",
			Description: "只读查看平台租户、用户、角色、权限与审计日志",
			Scope:       RoleScopePlatform,
			Permissions: []ProvisioningPermission{
				{Resource: "/api/v1/tenants", Action: "GET"},
				{Resource: "/api/v1/users", Action: "GET"},
				{Resource: "/api/v1/users/*", Action: "GET"},
				{Resource: "/api/v1/roles", Action: "GET"},
				{Resource: "/api/v1/roles/*", Action: "GET"},
				{Resource: "/api/v1/roles/*/permissions", Action: "GET"},
				{Resource: "/api/v1/permissions", Action: "GET"},
				{Resource: "/api/v1/audits", Action: "GET"},
				{Resource: "/api/v1/audits/*", Action: "GET"},
				{Resource: "/api/v1/audits/export", Action: "GET"},
			},
		},
	}
}

func TenantSystemRoles() []SystemRoleSpec {
	return []SystemRoleSpec{
		{
			Code:        RoleCodeTenantAdmin,
			Name:        "租户管理员",
			Description: "管理当前租户用户并分配租户内角色",
			Scope:       RoleScopeTenant,
			Permissions: []ProvisioningPermission{
				{Resource: "/api/v1/users", Action: "GET"},
				{Resource: "/api/v1/users", Action: "POST"},
				{Resource: "/api/v1/users/*", Action: "GET"},
				{Resource: "/api/v1/users/*", Action: "PUT"},
				{Resource: "/api/v1/users/*", Action: "DELETE"},
				{Resource: "/api/v1/users/*/roles", Action: "POST"},
				{Resource: "/api/v1/roles/options", Action: "GET"},
				{Resource: "/api/v1/audits", Action: "GET"},
				{Resource: "/api/v1/audits/*", Action: "GET"},
			},
		},
		{
			Code:        RoleCodeTenantMember,
			Name:        "租户成员",
			Description: "当前租户的普通成员",
			Scope:       RoleScopeTenant,
		},
	}
}

func IsPlatformOnlyPermission(resource, _ string) bool {
	switch {
	case resource == "/api/v1/tenants", strings.HasPrefix(resource, "/api/v1/tenants/"):
		return true
	case resource == "/api/v1/tenant-plans", strings.HasPrefix(resource, "/api/v1/tenant-plans/"):
		return true
	case resource == "/api/v1/permissions", strings.HasPrefix(resource, "/api/v1/permissions/"):
		return true
	case resource == "/api/v1/roles":
		return true
	case strings.HasPrefix(resource, "/api/v1/roles/") && resource != "/api/v1/roles/options":
		return true
	case resource == "/api/v1/admin", strings.HasPrefix(resource, "/api/v1/admin/"):
		return true
	default:
		return false
	}
}
