package audit

import (
	"jimu/internal/contract"
	"jimu/internal/modules/audit/application"
	"jimu/internal/modules/audit/infrastructure"
	"jimu/internal/modules/audit/interfaces"

	"gorm.io/gorm"
)

type Module struct {
	service *application.AuditService
}

func New(db *gorm.DB) *Module {
	repo := infrastructure.NewMysqlAuditRepository(db)
	service := application.NewAuditService(repo)
	return &Module{service: service}
}

func (m *Module) Name() string {
	return "audit"
}

func (m *Module) RegisterHTTP(r contract.Router) {
	interfaces.RegisterAuditRoutes(r.Group("/api/v1"), m.service)
}

func (m *Module) RegisterJobs(j contract.JobRegistry) {}

func (m *Module) RegisterEvents(e contract.EventBus) {}
