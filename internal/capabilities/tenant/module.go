package tenant

import (
	"embed"
	"jimu/internal/capabilities/tenant/application"
	"jimu/internal/capabilities/tenant/infrastructure"
	"jimu/internal/capabilities/tenant/interfaces"
	"jimu/internal/contract"

	"gorm.io/gorm"
)

type Module struct {
	service     *application.TenantService
	plans       *application.PlanService
	quota       *application.QuotaService
	provisioner *application.GormTenantProvisioner
}

// PortName 租户能力对外提供的端口名：contract.TenantQuota（QuotaService）。
const PortName = "tenant"

// ProvisionerPortName 开通式注册端口名：contract.TenantProvisioner（未启用时空）。
const ProvisionerPortName = "tenant.provisioner"

// New 创建 tenant 模块。prov 为开通式注册配置（由组合根从 auth 段构造）。
func New(db *gorm.DB, prov ProvisioningConfig) *Module {
	repo := infrastructure.NewMysqlRepository(db)
	quotaRepo := infrastructure.NewMysqlQuotaRepository(db)
	m := &Module{
		service: application.NewTenantService(repo),
		plans:   application.NewPlanService(infrastructure.NewMysqlPlanRepository(db), quotaRepo),
		quota:   application.NewQuotaService(quotaRepo),
	}
	// 开通式注册（注册 = 开通新租户）：启用时暴露 provisioner 供 auth 经端口消费
	if prov.Enabled {
		m.provisioner = application.NewGormTenantProvisioner(db, prov)
	}
	return m
}

// Quota 暴露配额校验服务，供用户/角色/API Key 创建路径注入
func (m *Module) Quota() *application.QuotaService { return m.quota }

// Provisioner 实现 contract.TenantProvisioner；未启用开通式注册时返回 nil。
func (m *Module) Provisioner() contract.TenantProvisioner {
	if m.provisioner == nil {
		return nil
	}
	return m.provisioner
}

func (m *Module) Name() string {
	return "tenant"
}

// migrationsFS 能力自带迁移（Task 3：能力迁移经 embed 进二进制）。
//
//go:embed migrations
var migrationsFS embed.FS

// Descriptor 声明租户能力的静态描述。
var Descriptor = contract.Descriptor{
	Name: "tenant",
	// tenant 迁移（005_tenants.sql）会 ALTER users/roles，须后于 user/access 执行
	Requires:   []string{"user", "access"},
	Migrations: migrationsFS,
	Owns:       []string{"tenants", "tenant_plans"},
	Mount:      contract.MountProtected,
	Permissions: []contract.Permission{
		{Name: "租户列表", Resource: "/api/v1/tenants", Action: "GET"},
		{Name: "租户创建", Resource: "/api/v1/tenants", Action: "POST"},
		{Name: "租户详情", Resource: "/api/v1/tenants/*", Action: "GET"},
		{Name: "租户修改", Resource: "/api/v1/tenants/*", Action: "PUT"},
		{Name: "租户删除", Resource: "/api/v1/tenants/*", Action: "DELETE"},
		// 租户运营：套餐定义（用量查询与套餐分配分别由「租户详情」「租户修改」通配覆盖）
		{Name: "套餐列表", Resource: "/api/v1/tenant-plans", Action: "GET"},
		{Name: "套餐创建", Resource: "/api/v1/tenant-plans", Action: "POST"},
		{Name: "套餐修改", Resource: "/api/v1/tenant-plans/*", Action: "PUT"},
		{Name: "套餐删除", Resource: "/api/v1/tenant-plans/*", Action: "DELETE"},
	},
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
