package audit

import (
	"jimu/internal/config"
	"jimu/internal/contract"
	"jimu/internal/modules/audit/application"
	"jimu/internal/modules/audit/infrastructure"
	"jimu/internal/modules/audit/interfaces"
	"jimu/internal/platform/logger"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type Module struct {
	service *application.AuditService
	worker  *application.Worker
}

func New(db *gorm.DB, cfg config.AuditConfig, log *logger.Logger) *Module {
	repo := infrastructure.NewMysqlAuditRepository(db)
	return &Module{
		service: application.NewAuditService(repo),
		worker:  application.NewWorker(repo, cfg, log),
	}
}

func (m *Module) Name() string { return "audit" }

func (m *Module) RegisterHTTP(r contract.Router) {
	interfaces.RegisterAuditRoutes(r.Group("/api/v1"), m.service)
}

func (m *Module) HTTPMiddleware() []gin.HandlerFunc {
	return []gin.HandlerFunc{interfaces.AuditMiddleware(m.worker)}
}

func (m *Module) Components() []contract.Component {
	return []contract.Component{m.worker}
}

func (m *Module) RegisterJobs(j contract.JobRegistry) {}
func (m *Module) RegisterEvents(e contract.EventBus)  {}
