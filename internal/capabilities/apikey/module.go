package apikey

import (
	"jimu/internal/capabilities/apikey/application"
	"jimu/internal/capabilities/apikey/infrastructure"
	"jimu/internal/capabilities/apikey/interfaces"
	"jimu/internal/contract"
	"jimu/internal/kernel/http/middleware"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// Module API Key 能力的模块实例：注册 /api/v1/admin/apikeys* 管理端点
// （api_keys 表所有者 = apikey）。
type Module struct {
	service  *application.AdminAPIKeyService
	verifier *APIKeyVerifier
	// protected 声明本能力是否充当受保护中间件提供者：machine 形态没有 auth，
	// 受保护路由由 X-API-Key 承担；full 形态 auth 已在，apikey 让位（空链），
	// 以免触发 bootstrap 的单提供者 fail-closed 规则。
	protected bool
}

// protectedMiddleware 装配期 dep：本能力是否接管受保护路由。
type protectedMiddleware bool

// WithProtectedMiddleware 返回装配期 dep，供 Wire 按启用集决定 apikey 是否充当
// 受保护中间件提供者（启用集里有 auth 时让位）。
func WithProtectedMiddleware(enabled bool) interface{} { return protectedMiddleware(enabled) }

// New 创建 API Key 模块。quota 可选（tenant 能力提供）。
func New(db *gorm.DB, deps ...interface{}) *Module {
	svc := application.NewAdminAPIKeyService(infrastructure.NewMysqlAPIKeyRepository(db))
	m := &Module{service: svc, verifier: NewAPIKeyVerifier(NewDBAPIKeyStore(db))}
	for _, dep := range deps {
		switch d := dep.(type) {
		case application.TenantQuota:
			svc.WithQuota(d)
		case protectedMiddleware:
			m.protected = bool(d)
		}
	}
	return m
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

// ProtectedHTTPMiddleware 实现 contract.ProtectedHTTPMiddlewareProvider：machine 形态
// 没有 auth（无登录/会话端点），受保护路由改由 X-API-Key 认证 + scope 校验 + Key 归属
// 租户注入承担，复用能力内既有的两条中间件而不是新写认证路径。
// 启用集里已有 auth 时本能力不是提供者，返回空链让位（见 bootstrap 的单提供者规则）。
func (m *Module) ProtectedHTTPMiddleware() ([]gin.HandlerFunc, error) {
	if !m.protected {
		return nil, nil
	}
	return []gin.HandlerFunc{APIKeyAuthMiddleware(m.verifier), RequireScope(ScopeProtected)}, nil
}

var _ contract.Module = (*Module)(nil)

// 受保护中间件提供者（machine 形态；启用集含 auth 时返回空链让位）。
var _ contract.ProtectedHTTPMiddlewareProvider = (*Module)(nil)
