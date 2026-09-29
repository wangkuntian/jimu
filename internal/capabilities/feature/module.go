package feature

import (
	"jimu/internal/contract"
	"jimu/internal/kernel/http/middleware"

	"gorm.io/gorm"
)

// Module Feature Flag 能力的模块实例：注册 /api/v1/admin/features* 端点。
type Module struct {
	manager *Manager
}

// New 创建 Feature Flag 模块。
func New(_ *gorm.DB) *Module {
	return &Module{manager: NewManager()}
}

// Name 模块名。
func (m *Module) Name() string { return "feature" }

// Manager 暴露 Feature Flag 管理器（供读侧判定开关）。
func (m *Module) Manager() *Manager { return m.manager }

// Descriptor 声明 Feature Flag 能力的静态描述。
var Descriptor = contract.Descriptor{
	Name:  "feature",
	Mount: contract.MountProtected,
}

// Descriptor 实现 contract.Describable。
func (m *Module) Descriptor() contract.Descriptor { return Descriptor }

// RegisterHTTP 注册 Feature Flag 管理端点。
func (m *Module) RegisterHTTP(r contract.Router) {
	admin := r.Group("/api/v1/admin")
	admin.Use(middleware.AdminAuth())
	handler := NewAdminFeatureHandler(m.manager)
	admin.GET("/features", handler.List)
	admin.PUT("/features/:name", handler.Update)
}

func (m *Module) RegisterJobs(j contract.JobRegistry) {}

func (m *Module) RegisterEvents(e contract.EventBus) {}

var _ contract.Module = (*Module)(nil)
