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
	"jimu/internal/platform/event"
	"jimu/internal/platform/http/middleware"
	"jimu/internal/platform/scheduler"
	"jimu/internal/platform/ws"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// Module 管理模块
type Module struct {
	service *adminapp.Service
	rdb     *redis.Client
	db      *gorm.DB
	sched   *scheduler.CronScheduler
}

// New 创建管理模块
func New(version, env string, rdb *redis.Client, db *gorm.DB, sched ...*scheduler.CronScheduler) *Module {
	m := &Module{
		service: adminapp.NewService(version, env, rdb),
		rdb:     rdb,
		db:      db,
	}
	if len(sched) > 0 {
		m.sched = sched[0]
	}
	return m
}

// wsHandler 创建 WebSocket 处理器
func (m *Module) wsHandler() http.HandlerFunc {
	presence := ws.NewPresenceManager()
	channels := ws.NewChannelManager()
	hub := ws.NewClientHub(presence, channels)
	go hub.Run(context.Background())
	return ws.WSHandler(hub, auth.New("dev-secret", "jimu", 30, 7), presence, channels)
}

// Name 返回模块名称
func (m *Module) Name() string { return "admin" }

// RegisterHTTP 注册管理端路由
func (m *Module) RegisterHTTP(r contract.Router) {
	// 管理员权限中间件
	admin := r.Group("")
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

	// 用户管理端点
	userHandler := admininterfaces.NewAdminUserHandler(
		adminapp.NewAdminUserService(userinfra.NewMysqlRepository(m.db)),
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
		adminapp.NewAdminConfigService(m.rdb, event.New(), "jimu"),
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

	// 审计日志端点
	auditHandler := admininterfaces.NewAdminAuditHandler(
		adminapp.NewAdminAuditService(admininfra.NewMysqlAuditRepository(m.db)),
	)
	admin.GET("/audit", auditHandler.List)

	// 任务队列端点
	jobHandler := admininterfaces.NewAdminJobHandler(admininfra.NewMysqlJobRepository(m.db))
	admin.GET("/jobs", jobHandler.List)
	admin.POST("/jobs", jobHandler.Submit)
	admin.GET("/jobs/:id", jobHandler.Get)
	admin.POST("/jobs/:id/retry", jobHandler.Retry)
	admin.GET("/jobs/dead-letters", jobHandler.ListDeadLetters)
	admin.POST("/jobs/dead-letters/:id/resolve", jobHandler.ResolveDeadLetter)

	// WebSocket 实时通信端点
	wsHandler := m.wsHandler()
	admin.GET("/ws", gin.WrapF(wsHandler))
}

// RegisterJobs 注册定时任务
func (m *Module) RegisterJobs(j contract.JobRegistry) {}

// RegisterEvents 注册事件处理器
func (m *Module) RegisterEvents(e contract.EventBus) {}
