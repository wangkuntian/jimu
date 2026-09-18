package role

import (
	"jimu/internal/contract"
	"jimu/internal/modules/role/application"
	"jimu/internal/modules/role/infrastructure"
	"jimu/internal/modules/role/interfaces"

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

// Descriptor 声明角色能力的静态描述。
var Descriptor = contract.Descriptor{
	Name:  "role",
	Mount: contract.MountProtected,
}

// Descriptor 实现 contract.Describable。
func (m *Module) Descriptor() contract.Descriptor { return Descriptor }

func (m *Module) RegisterHTTP(r contract.Router) {
	interfaces.RegisterRoleRoutes(r.Group("/api/v1"), m.service)
}

func (m *Module) RegisterJobs(j contract.JobRegistry) {}

func (m *Module) RegisterEvents(e contract.EventBus) {}
