package uploadsec

import (
	"jimu/internal/capabilities/storage"
	"jimu/internal/contract"
	"jimu/internal/kernel/http/middleware"
)

// Module 上传安全能力的模块实例：注册 /api/v1/admin/files 上传与删除端点。
type Module struct {
	storage storage.Storage
	scanner Scanner
}

// New 创建上传安全模块。storage 为空时不注册端点（与拆分前 admin 行为一致）。
func New(st storage.Storage, scanner Scanner) *Module {
	return &Module{storage: st, scanner: scanner}
}

// Name 模块名。
func (m *Module) Name() string { return "uploadsec" }

// Descriptor 声明上传安全能力的静态描述。
var Descriptor = contract.Descriptor{
	Name:  "uploadsec",
	Mount: contract.MountProtected,
	Configs: []contract.ConfigSpec{
		{Section: ConfigKey, New: func() any { return &Config{} }},
	},
}

// Descriptor 实现 contract.Describable。
func (m *Module) Descriptor() contract.Descriptor { return Descriptor }

// RegisterHTTP 注册管理端文件上传端点（接入存储抽象）。
func (m *Module) RegisterHTTP(r contract.Router) {
	if m.storage == nil {
		return
	}
	admin := r.Group("/api/v1/admin")
	admin.Use(middleware.AdminAuth())

	uploadHandler := NewUploadHandler(UploadConfig{
		Storage:    m.storage,
		MaxSize:    10 * 1024 * 1024,
		BasePrefix: "uploads",
		Scanner:    m.scanner,
	})
	admin.POST("/files", uploadHandler.HandleUpload())
	admin.DELETE("/files", uploadHandler.HandleDelete())
}

func (m *Module) RegisterJobs(j contract.JobRegistry) {}

func (m *Module) RegisterEvents(e contract.EventBus) {}

var _ contract.Module = (*Module)(nil)
