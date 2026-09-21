package dataops

import (
	"jimu/internal/capabilities/dataops/application"
	"jimu/internal/capabilities/dataops/infrastructure"
	"jimu/internal/capabilities/dataops/interfaces"
	"jimu/internal/contract"
	"jimu/internal/kernel/http/middleware"

	"gorm.io/gorm"
)

// Module 数据导入导出的模块实例：注册 /api/v1/admin/users/import* 端点
// （import_jobs 表所有者 = dataops）。
type Module struct {
	db *gorm.DB
}

// New 创建 dataops 模块。
func New(db *gorm.DB) *Module {
	return &Module{db: db}
}

// Name 模块名。
func (m *Module) Name() string { return "dataops" }

// Descriptor 实现 contract.Describable（复用能力描述）。
func (m *Module) Descriptor() contract.Descriptor { return Descriptor }

// RegisterHTTP 注册用户导入端点。
func (m *Module) RegisterHTTP(r contract.Router) {
	admin := r.Group("/api/v1/admin")
	admin.Use(middleware.AdminAuth())

	svc := application.NewImportService(infrastructure.NewMysqlImportJobRepository(m.db), m.db)
	handler := interfaces.NewAdminImportHandler(svc)
	admin.POST("/users/import/preview", handler.Preview)
	admin.POST("/users/import", handler.Import)
	admin.GET("/users/import/template", handler.Template)
	admin.GET("/users/import/:id", handler.Get)
}

func (m *Module) RegisterJobs(j contract.JobRegistry) {}

func (m *Module) RegisterEvents(e contract.EventBus) {}

var _ contract.Module = (*Module)(nil)
