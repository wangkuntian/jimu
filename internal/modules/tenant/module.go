package tenant

import (
	"jimu/internal/config"
	"jimu/internal/contract"
	"jimu/internal/modules/tenant/application"
	"jimu/internal/modules/tenant/infrastructure"
	"jimu/internal/modules/tenant/interfaces"

	"gorm.io/gorm"
)

type Module struct {
	service *application.TenantService
}

func New(db *gorm.DB, _ config.Config) *Module {
	repo := infrastructure.NewMysqlRepository(db)
	service := application.NewTenantService(repo)
	return &Module{service: service}
}

func (m *Module) Name() string {
	return "tenant"
}

func (m *Module) RegisterHTTP(r contract.Router) {
	interfaces.RegisterTenantRoutes(r.Group("/api/v1"), m.service)
}

func (m *Module) RegisterJobs(j contract.JobRegistry) {}

func (m *Module) RegisterEvents(e contract.EventBus) {}
