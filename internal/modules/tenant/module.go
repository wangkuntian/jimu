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

func (m *Module) RegisterHTTP(r contract.Router) {
	rg := r.Group("/api/v1")
	interfaces.RegisterTenantRoutes(rg, m.service)
	interfaces.RegisterPlanRoutes(rg, m.plans)
}

func (m *Module) RegisterJobs(j contract.JobRegistry) {}

func (m *Module) RegisterEvents(e contract.EventBus) {}
