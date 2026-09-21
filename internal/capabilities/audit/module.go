package audit

import (
	"embed"
	"jimu/internal/capabilities/audit/application"
	"jimu/internal/capabilities/audit/infrastructure"
	"jimu/internal/capabilities/audit/interfaces"
	"jimu/internal/config"
	"jimu/internal/contract"
	"jimu/internal/kernel/logger"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type Module struct {
	service *application.AuditService
	worker  *application.Worker
}

func New(db *gorm.DB, cfg config.AuditConfig, log *logger.Logger) *Module {
	repo := infrastructure.NewMysqlAuditRepository(db, cfg.HashSecret)
	return &Module{
		service: application.NewAuditService(repo, cfg.HashSecret),
		worker:  application.NewWorker(repo, cfg, log),
	}
}

func (m *Module) Name() string { return "audit" }

// migrationsFS 能力自带迁移（Task 3：能力迁移经 embed 进二进制）。
//
//go:embed migrations
var migrationsFS embed.FS

// Descriptor 声明审计能力的静态描述。
var Descriptor = contract.Descriptor{
	Name:       "audit",
	Migrations: migrationsFS,
	Mount:      contract.MountProtected,
	Permissions: []contract.Permission{
		{Name: "审计列表", Resource: "/api/v1/audits", Action: "GET"},
		{Name: "审计详情", Resource: "/api/v1/audits/*", Action: "GET"},
		// 审计导出路由 /audits/export 挂在 audit 能力（handler.go:62）
		{Name: "审计导出", Resource: "/api/v1/audits/export", Action: "GET"},
	},
}

// Descriptor 实现 contract.Describable。
func (m *Module) Descriptor() contract.Descriptor { return Descriptor }

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
