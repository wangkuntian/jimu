package apidocs

import "jimu/internal/contract"

// Descriptor 声明 API 文档能力的静态描述（非 catalog 条目：profile 显式列出）。
// 公开挂载：在 /swagger 下提供 Swagger UI（release 模式下不注册）。
var Descriptor = contract.Descriptor{
	Name:  "apidocs",
	Mount: contract.MountPublic,
}

// Module API 文档能力的模块实例：按 HTTP 模式决定是否注册 Swagger UI 路由。
type Module struct {
	enabled bool
}

// NewModule 创建 API 文档模块；enabled=false（release 模式）时不注册任何路由。
func NewModule(enabled bool) *Module { return &Module{enabled: enabled} }

// Name 模块名。
func (m *Module) Name() string { return "apidocs" }

// Descriptor 实现 contract.Describable。
func (m *Module) Descriptor() contract.Descriptor { return Descriptor }

// RegisterHTTP 注册 /swagger/* 路由。
func (m *Module) RegisterHTTP(r contract.Router) {
	if !m.enabled {
		return
	}
	RegisterSwagger(r.Group("/swagger"))
}

// RegisterJobs 本能力无定时任务。
func (m *Module) RegisterJobs(contract.JobRegistry) {}

// RegisterEvents 本能力无事件订阅。
func (m *Module) RegisterEvents(contract.EventBus) {}

var _ contract.Module = (*Module)(nil)
