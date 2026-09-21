package apikey

import (
	"jimu/internal/capabilities/apikey/application"
	"jimu/internal/capabilities/apikey/infrastructure"
	"jimu/internal/capabilities/apikey/interfaces"
	"jimu/internal/contract"
	"jimu/internal/kernel/http/middleware"

	"gorm.io/gorm"
)

// Module API Key 能力的模块实例：注册 /api/v1/admin/apikeys* 管理端点
// （api_keys 表所有者 = apikey）。
type Module struct {
	service *application.AdminAPIKeyService
}

// New 创建 API Key 模块。quota 可选（tenant 能力提供）。
func New(db *gorm.DB, deps ...interface{}) *Module {
	svc := application.NewAdminAPIKeyService(infrastructure.NewMysqlAPIKeyRepository(db))
	for _, dep := range deps {
		if quota, ok := dep.(application.TenantQuota); ok {
			svc.WithQuota(quota)
		}
	}
	return &Module{service: svc}
}

// Name 模块名。
func (m *Module) Name() string { return "apikey" }

// Descriptor 实现 contract.Describable（复用能力描述）。
func (m *Module) Descriptor() contract.Descriptor { return Descriptor }

// RegisterHTTP 注册 API Key 管理端点。
func (m *Module) RegisterHTTP(r contract.Router) {
	admin := r.Group("/api/v1/admin")
	admin.Use(middleware.AdminAuth())
	handler := interfaces.NewAdminAPIKeyHandler(m.service)
	admin.GET("/apikeys", handler.List)
	admin.POST("/apikeys", handler.Create)
	admin.GET("/apikeys/:id", handler.Get)
	admin.DELETE("/apikeys/:id", handler.Revoke)
}

func (m *Module) RegisterJobs(j contract.JobRegistry) {}

func (m *Module) RegisterEvents(e contract.EventBus) {}

var _ contract.Module = (*Module)(nil)
