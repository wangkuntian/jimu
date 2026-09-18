package permission

import (
	"jimu/internal/capabilities/permission/application"
	"jimu/internal/capabilities/permission/infrastructure"
	"jimu/internal/capabilities/permission/interfaces"
	"jimu/internal/contract"

	"gorm.io/gorm"
)

type Module struct {
	service *application.PermissionService
}

func New(db *gorm.DB) *Module {
	repo := infrastructure.NewMysqlPermissionRepository(db)
	service := application.NewPermissionService(repo)
	return &Module{service: service}
}

func (m *Module) Name() string {
	return "permission"
}

// Descriptor 声明权限能力的静态描述。
var Descriptor = contract.Descriptor{
	Name:     "permission",
	Requires: []string{"role"},
	Mount:    contract.MountProtected,
}

// Descriptor 实现 contract.Describable。
func (m *Module) Descriptor() contract.Descriptor { return Descriptor }

func (m *Module) RegisterHTTP(r contract.Router) {
	interfaces.RegisterPermissionRoutes(r.Group("/api/v1"), m.service)
}

func (m *Module) RegisterJobs(j contract.JobRegistry) {}

func (m *Module) RegisterEvents(e contract.EventBus) {}
