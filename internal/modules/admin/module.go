package admin

import (
	"context"
	"net/http"

	"jimu/internal/contract"
	adminapp "jimu/internal/modules/admin/application"
	admininfra "jimu/internal/modules/admin/infrastructure"
	admininterfaces "jimu/internal/modules/admin/interfaces"
	userinfra "jimu/internal/modules/user/infrastructure"
	"jimu/internal/platform/auth"
	"jimu/internal/platform/feature"
	platformhttp "jimu/internal/platform/http"
	"jimu/internal/platform/http/middleware"
	"jimu/internal/platform/scheduler"
	"jimu/internal/platform/storage"
	"jimu/internal/platform/ws"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// Module 管理模块
type Module struct {
	service    *adminapp.Service
	rdb        *redis.Client
	db         *gorm.DB
	sched      *scheduler.CronScheduler
	storage    storage.Storage
	feature    *feature.Manager
	eventBus   contract.EventBus
	wsHub      *ws.ClientHub
	wsPres     *ws.PresenceManager
	wsChannels *ws.ChannelManager
	jwt        *auth.JWT
}

// New 创建管理模块
func New(version, env string, rdb *redis.Client, db *gorm.DB, deps ...interface{}) *Module {
	m := &Module{
		service: adminapp.NewService(version, env, rdb),
		rdb:     rdb,
		db:      db,
	}
	for _, dep := range deps {
		switch d := dep.(type) {
		case *scheduler.CronScheduler:
			m.sched = d
		case storage.Storage:
			m.storage = d
		case *feature.Manager:
			m.feature = d
		case contract.EventBus:
			m.eventBus = d
		case *auth.JWT:
			m.jwt = d
		}
	}
	return m
}

// initWS 初始化 WebSocket hub 与 presence（幂等）
func (m *Module) initWS() {
	if m.wsHub != nil {
		return
	}
	m.wsPres = ws.NewPresenceManager()
	m.wsChannels = ws.NewChannelManager()
	m.wsHub = ws.NewClientHub(m.wsPres, m.wsChannels)
	go m.wsHub.Run(context.Background())
}

// wsHandler 创建 WebSocket 处理器
// JWT 实例由 main 注入（真实配置），避免 WS 与 HTTP 认证签名不一致
func (m *Module) wsHandler() http.HandlerFunc {
	m.initWS()
	if m.jwt == nil {
		return nil
	}
	return ws.WSHandler(m.wsHub, m.jwt, m.wsPres, m.wsChannels)
}

// Name 返回模块名称
func (m *Module) Name() string { return "admin" }

// RegisterHTTP 注册管理端路由
func (m *Module) RegisterHTTP(r contract.Router) {
	// 管理员权限中间件，统一挂载在 /api/v1/admin 前缀下
	admin := r.Group("/api/v1/admin")
	admin.Use(middleware.AdminAuth())

	// 公开端点（错误码文档）
	admin.GET("/error-codes", admininterfaces.NewHandler(m.service).GetErrorCodes)

	// 监控端点
	adminSvc := admininterfaces.NewAdminMonitoringHandler(
		adminapp.NewAdminMonitoringService(m.service.Version(), m.service.Env(), m.rdb),
	)
	admin.GET("/monitoring/status", adminSvc.Status)
	admin.GET("/monitoring/health", adminSvc.Health)
	admin.GET("/monitoring/metrics", adminSvc.Metrics)

	// 限流状态可视化端点（仅读，不消费令牌；查看登录爆破防护等计数）
	admin.GET("/ratelimit/auth", admininterfaces.NewAdminRateLimitHandler(m.rdb).AuthPeek)

	// 用户管理端点
	userHandler := admininterfaces.NewAdminUserHandler(
		adminapp.NewAdminUserService(userinfra.NewMysqlRepository(m.db), m.db),
	)
	admin.GET("/users", userHandler.List)
	admin.POST("/users", userHandler.Create)
	admin.GET("/users/:id", userHandler.Get)
	admin.PUT("/users/:id", userHandler.Update)
	admin.DELETE("/users/:id", userHandler.Disable)
	admin.POST("/users/:id/roles", userHandler.AssignRole)

	// API Key 管理端点
	apiKeyHandler := admininterfaces.NewAdminAPIKeyHandler(
		adminapp.NewAdminAPIKeyService(admininfra.NewMysqlAPIKeyRepository(m.db)),
	)
	admin.GET("/apikeys", apiKeyHandler.List)
	admin.POST("/apikeys", apiKeyHandler.Create)
	admin.GET("/apikeys/:id", apiKeyHandler.Get)
	admin.DELETE("/apikeys/:id", apiKeyHandler.Revoke)

	// 配置热更新端点
	configHandler := admininterfaces.NewAdminConfigHandler(
		adminapp.NewAdminConfigService(m.rdb, m.eventBus, "jimu"),
	)
	admin.GET("/config", configHandler.Get)
	admin.PUT("/config/:key", configHandler.Update)
	admin.POST("/config/reload", configHandler.Reload)

	// 任务调度端点
	taskHandler := admininterfaces.NewAdminTaskHandler(adminapp.NewAdminTaskService(m.sched))
	admin.GET("/tasks", taskHandler.List)
	admin.POST("/tasks/:id/run", taskHandler.Trigger)
	admin.POST("/tasks/:id/toggle", taskHandler.Toggle)
	admin.GET("/tasks/:id/history", taskHandler.History)

	// 数据导入端点
	importHandler := admininterfaces.NewAdminImportHandler(
		adminapp.NewImportService(
			admininfra.NewMysqlImportJobRepository(m.db),
			userinfra.NewMysqlRepository(m.db),
			m.db,
		),
	)
	admin.POST("/users/import/preview", importHandler.Preview)
	admin.POST("/users/import", importHandler.Import)
	admin.GET("/users/import/template", importHandler.Template)
	admin.GET("/users/import/:id", importHandler.Get)

	// 审计日志端点（复用 audit 模块仓储，读 006 迁移的 audit_logs 表）
	admin.GET("/audit", admininterfaces.NewAdminAuditHandler(m.db).List)

	// 任务队列端点
	jobHandler := admininterfaces.NewAdminJobHandler(
		admininfra.NewMysqlJobRepository(m.db),
		admininfra.NewMysqlDeadLetterRepository(m.db),
	)
	admin.GET("/jobs", jobHandler.List)
	admin.POST("/jobs", jobHandler.Submit)
	admin.GET("/jobs/:id", jobHandler.Get)
	admin.POST("/jobs/:id/retry", jobHandler.Retry)
	admin.GET("/jobs/dead-letters", jobHandler.ListDeadLetters)
	admin.POST("/jobs/dead-letters/:id/resolve", jobHandler.ResolveDeadLetter)

	// WebSocket 实时通信端点
	m.initWS()
	admin.GET("/ws", gin.WrapF(m.wsHandler()))
	wsAdmin := admininterfaces.NewAdminWSHandler(m.wsHub, m.wsPres)
	admin.POST("/ws/push", wsAdmin.Push)
	admin.GET("/ws/presence/:userId", wsAdmin.Presence)
	admin.GET("/ws/online", wsAdmin.OnlineUsers)

	// 文件上传端点（接入存储抽象）
	if m.storage != nil {
		uploadHandler := platformhttp.NewUploadHandler(platformhttp.UploadConfig{
			Storage:    m.storage,
			MaxSize:    10 * 1024 * 1024,
			BasePrefix: "uploads",
		})
		admin.POST("/files", uploadHandler.HandleUpload())
		admin.DELETE("/files", uploadHandler.HandleDelete())
	}

	// Feature Flag 端点（运行时开关功能）
	if m.feature != nil {
		featureHandler := admininterfaces.NewAdminFeatureHandler(m.feature)
		admin.GET("/features", featureHandler.List)
		admin.PUT("/features/:name", featureHandler.Update)
	}
}

// RegisterJobs 注册定时任务
func (m *Module) RegisterJobs(j contract.JobRegistry) {}

// RegisterEvents 注册事件处理器
func (m *Module) RegisterEvents(e contract.EventBus) {}
