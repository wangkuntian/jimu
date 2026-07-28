package permission

import (
	"jimu/internal/contract"
	"jimu/internal/modules/permission/application"
	"jimu/internal/modules/permission/infrastructure"
	"jimu/internal/modules/permission/interfaces"

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

func (m *Module) RegisterHTTP(r contract.Router) {
	interfaces.RegisterPermissionRoutes(r.Group("/api/v1"), m.service)
}

func (m *Module) RegisterJobs(j contract.JobRegistry) {}

func (m *Module) RegisterEvents(e contract.EventBus) {}
