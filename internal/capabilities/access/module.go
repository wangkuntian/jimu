// Package access 访问控制能力：角色 + 权限 + 用户角色分配（user_roles 表所有者）。
// Casbin RBAC 机制在 kernel/access，本能力负责实体、用例与管理端点。
package access

import (
	"embed"

	"jimu/internal/capabilities/access/application"
	"jimu/internal/capabilities/access/infrastructure"
	"jimu/internal/capabilities/access/interfaces"
	"jimu/internal/contract"
	"jimu/internal/kernel/http/middleware"

	"gorm.io/gorm"
)

// Module 访问控制能力。
type Module struct {
	roles       *application.RoleService
	permissions *application.PermissionService
	userRoles   *application.UserRoleService
}

// PortName access 能力对外提供的端口名：contract.UserRoleAssigner（UserRoleService）。
const PortName = "access"

// New 创建 access 模块。quota 可选（tenant 能力提供）。
func New(db *gorm.DB, deps ...interface{}) *Module {
	roleRepo := infrastructure.NewMysqlRepository(db)
	permRepo := infrastructure.NewMysqlPermissionRepository(db)
	roleSvc := application.NewRoleService(roleRepo)
	for _, dep := range deps {
		if quota, ok := dep.(application.TenantQuota); ok {
			roleSvc.WithQuota(quota)
		}
	}
	return &Module{
		roles:       roleSvc,
		permissions: application.NewPermissionService(permRepo),
		userRoles:   application.NewUserRoleService(db),
	}
}

// Name 模块名。
func (m *Module) Name() string { return "access" }

// UserRoleAssigner 实现 contract.UserRoleAssigner，供 user 管理面复用。
func (m *Module) UserRoleAssigner() contract.UserRoleAssigner { return m.userRoles }

// migrationsFS 能力自带迁移（roles/permissions/user_roles 表）。
//
//go:embed migrations
var migrationsFS embed.FS

// Descriptor 声明访问控制能力的静态描述。
var Descriptor = contract.Descriptor{
	Name:         "access",
	Requires:     []string{"user"},
	SoftRequires: []string{"tenant"},
	Migrations:   migrationsFS,
	Owns:         []string{"roles", "permissions", "role_permissions", "user_roles"},
	Mount:        contract.MountProtected,
	Permissions: []contract.Permission{
		// 角色
		{Name: "角色列表", Resource: "/api/v1/roles", Action: "GET"},
		{Name: "角色创建", Resource: "/api/v1/roles", Action: "POST"},
		{Name: "角色详情", Resource: "/api/v1/roles/*", Action: "GET"},
		{Name: "角色修改", Resource: "/api/v1/roles/*", Action: "PUT"},
		{Name: "角色删除", Resource: "/api/v1/roles/*", Action: "DELETE"},
		{Name: "角色分配权限", Resource: "/api/v1/roles/*/permissions", Action: "POST"},
		// 权限
		{Name: "权限列表", Resource: "/api/v1/permissions", Action: "GET"},
		{Name: "权限创建", Resource: "/api/v1/permissions", Action: "POST"},
		{Name: "权限详情", Resource: "/api/v1/permissions/*", Action: "GET"},
		{Name: "权限修改", Resource: "/api/v1/permissions/*", Action: "PUT"},
		{Name: "权限删除", Resource: "/api/v1/permissions/*", Action: "DELETE"},
		// 用户分配角色 /api/v1/admin/users/:id/roles 由 console 的 /api/v1/admin/* 通配权限点覆盖，
		// 此处不重复声明（保持拆分前的权限点集合不变）。
	},
}

// Descriptor 实现 contract.Describable。
func (m *Module) Descriptor() contract.Descriptor { return Descriptor }

// RegisterHTTP 注册角色/权限路由与管理端「用户分配角色」端点。
func (m *Module) RegisterHTTP(r contract.Router) {
	rg := r.Group("/api/v1")
	interfaces.RegisterRoleRoutes(rg, m.roles)
	interfaces.RegisterPermissionRoutes(rg, m.permissions)

	// 管理端「用户分配角色」：写 user_roles，由本能力（表所有者）注册
	admin := rg.Group("/admin")
	admin.Use(middleware.AdminAuth())
	admin.POST("/users/:id/roles", interfaces.NewUserRoleHandler(m.userRoles).AssignRole)
}

func (m *Module) RegisterJobs(j contract.JobRegistry) {}

func (m *Module) RegisterEvents(e contract.EventBus) {}

var _ contract.Module = (*Module)(nil)
