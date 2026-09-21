package admin

import (
	"context"
	"net/http"

	adminapp "jimu/internal/capabilities/admin/application"
	admininfra "jimu/internal/capabilities/admin/infrastructure"
	admininterfaces "jimu/internal/capabilities/admin/interfaces"
	"jimu/internal/capabilities/feature"
	"jimu/internal/capabilities/storage"
	"jimu/internal/capabilities/uploadsec"
	userinfra "jimu/internal/capabilities/user/infrastructure"
	"jimu/internal/capabilities/ws"
	"jimu/internal/contract"
	"jimu/internal/kernel/auth"
	"jimu/internal/kernel/http/middleware"
	"jimu/internal/kernel/scheduler"

	redistore "jimu/internal/kernel/redis"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// Module 管理模块
type Module struct {
	ipAllowlist gin.HandlerFunc
	service     *adminapp.Service
	rdb         redistore.Client
	db          *gorm.DB
	sched       *scheduler.CronScheduler
	storage     storage.Storage
	scanner     uploadsec.Scanner
	feature     *feature.Manager
	eventBus    contract.EventBus
	wsHub       *ws.ClientHub
	wsPres      *ws.PresenceManager
	wsChannels  *ws.ChannelManager
	jwt         *auth.JWT
	quota       adminapp.TenantQuota
}

// New 创建管理模块
func New(version, env string, rdb redistore.Client, db *gorm.DB, deps ...interface{}) *Module {
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
		case uploadsec.Scanner:
			m.scanner = d
		case *feature.Manager:
			m.feature = d
		case contract.EventBus:
			m.eventBus = d
		case *auth.JWT:
			m.jwt = d
		case gin.HandlerFunc:
			m.ipAllowlist = d
		case adminapp.TenantQuota:
			m.quota = d
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

// Descriptor 声明管理端能力的静态描述。
var Descriptor = contract.Descriptor{
	Name:     "admin",
	Requires: []string{"user", "audit"},
	Mount:    contract.MountProtected,
	Permissions: []contract.Permission{
		// 管理后台端点（/api/v1/admin/* 由 keyMatch 通配覆盖全部管理 API）
		{Name: "管理后台读取", Resource: "/api/v1/admin/*", Action: "GET"},
		{Name: "管理后台写入", Resource: "/api/v1/admin/*", Action: "POST"},
		{Name: "管理后台修改", Resource: "/api/v1/admin/*", Action: "PUT"},
		{Name: "管理后台删除", Resource: "/api/v1/admin/*", Action: "DELETE"},
	},
}

// Descriptor 实现 contract.Describable。
func (m *Module) Descriptor() contract.Descriptor { return Descriptor }

// RegisterHTTP 注册管理端路由
func (m *Module) RegisterHTTP(r contract.Router) {
	// 管理员权限中间件，统一挂载在 /api/v1/admin 前缀下
	admin := r.Group("/api/v1/admin")
	// 管理端 IP 白名单需先于鉴权生效（未配置时不挂载）
	if m.ipAllowlist != nil {
		admin.Use(m.ipAllowlist)
	}
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

	// API Key 管理端点
	apiKeyHandler := admininterfaces.NewAdminAPIKeyHandler(
		adminapp.NewAdminAPIKeyService(admininfra.NewMysqlAPIKeyRepository(m.db)).WithQuota(m.quota),
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

	// WebSocket 实时通信端点
	m.initWS()
	admin.GET("/ws", gin.WrapF(m.wsHandler()))
	wsAdmin := admininterfaces.NewAdminWSHandler(m.wsHub, m.wsPres)
	admin.POST("/ws/push", wsAdmin.Push)
	admin.GET("/ws/presence/:userId", wsAdmin.Presence)
	admin.GET("/ws/online", wsAdmin.OnlineUsers)

	// 文件上传端点（接入存储抽象）
	if m.storage != nil {
		uploadHandler := uploadsec.NewUploadHandler(uploadsec.UploadConfig{
			Storage:    m.storage,
			MaxSize:    10 * 1024 * 1024,
			BasePrefix: "uploads",
			Scanner:    m.scanner,
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
