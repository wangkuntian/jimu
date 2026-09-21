package role

import (
	"embed"
	"jimu/internal/capabilities/role/application"
	"jimu/internal/capabilities/role/infrastructure"
	"jimu/internal/capabilities/role/interfaces"
	"jimu/internal/contract"

	"gorm.io/gorm"
)

type Module struct {
	service *application.RoleService
}

func New(db *gorm.DB, deps ...interface{}) *Module {
	repo := infrastructure.NewMysqlRepository(db)
	service := application.NewRoleService(repo)
	for _, dep := range deps {
		if quota, ok := dep.(application.TenantQuota); ok {
			service.WithQuota(quota)
		}
	}
	return &Module{service: service}
}

func (m *Module) Name() string {
	return "role"
}

// migrationsFS 能力自带迁移（Task 3：能力迁移经 embed 进二进制）。
//
//go:embed migrations
var migrationsFS embed.FS

// Descriptor 声明角色能力的静态描述。
var Descriptor = contract.Descriptor{
	Name:       "role",
	Migrations: migrationsFS,
	Mount:      contract.MountProtected,
	Permissions: []contract.Permission{
		{Name: "角色列表", Resource: "/api/v1/roles", Action: "GET"},
		{Name: "角色创建", Resource: "/api/v1/roles", Action: "POST"},
		{Name: "角色详情", Resource: "/api/v1/roles/*", Action: "GET"},
		{Name: "角色修改", Resource: "/api/v1/roles/*", Action: "PUT"},
		{Name: "角色删除", Resource: "/api/v1/roles/*", Action: "DELETE"},
		{Name: "角色分配权限", Resource: "/api/v1/roles/*/permissions", Action: "POST"},
	},
}

// Descriptor 实现 contract.Describable。
func (m *Module) Descriptor() contract.Descriptor { return Descriptor }

func (m *Module) RegisterHTTP(r contract.Router) {
	interfaces.RegisterRoleRoutes(r.Group("/api/v1"), m.service)
}

func (m *Module) RegisterJobs(j contract.JobRegistry) {}

func (m *Module) RegisterEvents(e contract.EventBus) {}
