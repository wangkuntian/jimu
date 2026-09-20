package tenant

import (
	"embed"
	"jimu/internal/capabilities/tenant/application"
	"jimu/internal/capabilities/tenant/infrastructure"
	"jimu/internal/capabilities/tenant/interfaces"
	"jimu/internal/config"
	"jimu/internal/contract"

	"gorm.io/gorm"
)

type Module struct {
	service *application.TenantService
	plans   *application.PlanService
	quota   *application.QuotaService
}

func New(db *gorm.DB, _ config.Config) *Module {
	repo := infrastructure.NewMysqlRepository(db)
	quotaRepo := infrastructure.NewMysqlQuotaRepository(db)
	return &Module{
		service: application.NewTenantService(repo),
		plans:   application.NewPlanService(infrastructure.NewMysqlPlanRepository(db), quotaRepo),
		quota:   application.NewQuotaService(quotaRepo),
	}
}

// Quota 暴露配额校验服务，供用户/角色/API Key 创建路径注入
func (m *Module) Quota() *application.QuotaService { return m.quota }

func (m *Module) Name() string {
	return "tenant"
}

// migrationsFS 能力自带迁移（Task 3：能力迁移经 embed 进二进制）。
//
//go:embed migrations
var migrationsFS embed.FS

// Descriptor 声明租户能力的静态描述。
var Descriptor = contract.Descriptor{
	Name:       "tenant",
	Migrations: migrationsFS,
	Mount:      contract.MountProtected,
}

// Descriptor 实现 contract.Describable。
func (m *Module) Descriptor() contract.Descriptor { return Descriptor }

func (m *Module) RegisterHTTP(r contract.Router) {
	rg := r.Group("/api/v1")
	interfaces.RegisterTenantRoutes(rg, m.service)
	interfaces.RegisterPlanRoutes(rg, m.plans)
}

func (m *Module) RegisterJobs(j contract.JobRegistry) {}

func (m *Module) RegisterEvents(e contract.EventBus) {}
