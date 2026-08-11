// internal/modules/oauth/module.go
package oauth

import (
	"jimu/internal/contract"
	oauthapp "jimu/internal/modules/oauth/application"
	"jimu/internal/modules/oauth/interfaces"
)

// Module OAuth 模块
type Module struct {
	service *oauthapp.OAuthService
}

// New 创建 OAuth 模块
func New(service *oauthapp.OAuthService) *Module {
	return &Module{service: service}
}

// Name 模块名
func (m *Module) Name() string { return "oauth" }

// RegisterHTTP 注册 HTTP 路由
func (m *Module) RegisterHTTP(r contract.Router) {
	rg := r.Group("/api/v1")
	interfaces.RegisterOAuthRoutes(rg, m.service)
}

// RegisterJobs 无定时任务
func (m *Module) RegisterJobs(j contract.JobRegistry) {}

// RegisterEvents 无事件
func (m *Module) RegisterEvents(e contract.EventBus) {}

var _ contract.Module = (*Module)(nil)
