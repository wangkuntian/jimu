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

func New(db *gorm.DB) *Module {
	repo := infrastructure.NewMysqlRepository(db)
	service := application.NewRoleService(repo)
	return &Module{service: service}
}

func (m *Module) Name() string {
	return "role"
}

func (m *Module) RegisterHTTP(r contract.Router) {
	interfaces.RegisterRoleRoutes(r.Group("/api/v1"), m.service)
}

func (m *Module) RegisterJobs(j contract.JobRegistry) {}

func (m *Module) RegisterEvents(e contract.EventBus) {}
