// Package console 平台级控制台能力：管理端准入 + 无业务归属的平台视图
// （错误码文档、运维监控、限流状态、配置热更新、WebSocket 管理端）。
//
// 它不拥有任何表，只承接「真正无归属的平台级视图」；各业务管理端点
// （users/jobs/apikeys/import/audit/features）已归还各自能力。
package console

import (
	"context"
	"net/http"

	"jimu/internal/capabilities/console/application"
	"jimu/internal/capabilities/console/interfaces"
	"jimu/internal/capabilities/ws"
	"jimu/internal/contract"
	"jimu/internal/kernel/auth"
	"jimu/internal/kernel/http/middleware"

	redistore "jimu/internal/kernel/redis"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// Module 平台控制台能力。
type Module struct {
	service     *application.Service
	rdb         redistore.Client
	eventBus    contract.EventBus
	jwt         *auth.JWT
	ipAllowlist gin.HandlerFunc
	db          *gorm.DB

	wsHub      *ws.ClientHub
	wsPres     *ws.PresenceManager
	wsChannels *ws.ChannelManager
}

// New 创建 console 模块。jwt 用于 WebSocket 认证；ipAllowlist 为空时不挂载白名单。
func New(version, env string, rdb redistore.Client, db *gorm.DB, jwt *auth.JWT, eventBus contract.EventBus, deps ...interface{}) *Module {
	m := &Module{
		service:  application.NewService(version, env, rdb),
		rdb:      rdb,
		db:       db,
		jwt:      jwt,
		eventBus: eventBus,
	}
	for _, dep := range deps {
		if fn, ok := dep.(gin.HandlerFunc); ok {
			m.ipAllowlist = fn
		}
	}
	return m
}

// Name 模块名。
func (m *Module) Name() string { return "console" }

// Descriptor 声明平台控制台能力的静态描述。
// 管理端准入（AdminAuth + IP 白名单）随本能力挂载；/api/v1/admin/* 通配权限点
// 也由本能力声明（覆盖拆分后各能力注册的管理端端点）。
var Descriptor = contract.Descriptor{
	Name:     "console",
	Requires: []string{"auth", "access"},
	Mount:    contract.MountSelfManaged,
	Permissions: []contract.Permission{
		{Name: "管理后台读取", Resource: "/api/v1/admin/*", Action: "GET"},
		{Name: "管理后台写入", Resource: "/api/v1/admin/*", Action: "POST"},
		{Name: "管理后台修改", Resource: "/api/v1/admin/*", Action: "PUT"},
		{Name: "管理后台删除", Resource: "/api/v1/admin/*", Action: "DELETE"},
	},
}

// Descriptor 实现 contract.Describable。
func (m *Module) Descriptor() contract.Descriptor { return Descriptor }

// initWS 初始化 WebSocket hub 与 presence（幂等）。
func (m *Module) initWS() {
	if m.wsHub != nil {
		return
	}
	m.wsPres = ws.NewPresenceManager()
	m.wsChannels = ws.NewChannelManager()
	m.wsHub = ws.NewClientHub(m.wsPres, m.wsChannels)
	go m.wsHub.Run(context.Background())
}

// RegisterHTTP 注册平台控制台端点（/api/v1/admin 前缀 + 管理端准入）。
func (m *Module) RegisterHTTP(r contract.Router) {
	admin := r.Group("/api/v1/admin")
	if m.ipAllowlist != nil {
		admin.Use(m.ipAllowlist)
	}
	admin.Use(middleware.AdminAuth())

	// 公开端点（错误码文档）
	admin.GET("/error-codes", interfaces.NewHandler(m.service).GetErrorCodes)

	// 运维监控端点
	monitoring := interfaces.NewAdminMonitoringHandler(
		application.NewAdminMonitoringService(m.service.Version(), m.service.Env(), m.rdb),
	)
	admin.GET("/monitoring/status", monitoring.Status)
	admin.GET("/monitoring/health", monitoring.Health)
	admin.GET("/monitoring/metrics", monitoring.Metrics)

	// 限流状态可视化端点（仅读，不消费令牌）
	admin.GET("/ratelimit/auth", interfaces.NewAdminRateLimitHandler(m.rdb).AuthPeek)

	// 配置热更新端点
	configHandler := interfaces.NewAdminConfigHandler(
		application.NewAdminConfigService(m.rdb, m.eventBus, "jimu"),
	)
	admin.GET("/config", configHandler.Get)
	admin.PUT("/config/:key", configHandler.Update)
	admin.POST("/config/reload", configHandler.Reload)

	// WebSocket 实时通信端点
	m.initWS()
	admin.GET("/ws", gin.WrapF(m.wsHandler()))
	wsAdmin := interfaces.NewAdminWSHandler(m.wsHub, m.wsPres)
	admin.POST("/ws/push", wsAdmin.Push)
	admin.GET("/ws/presence/:userId", wsAdmin.Presence)
	admin.GET("/ws/online", wsAdmin.OnlineUsers)
}

// wsHandler 创建 WebSocket 处理器；JWT 未注入时返回 nil（路由仍注册，调用会 500）。
func (m *Module) wsHandler() http.HandlerFunc {
	m.initWS()
	if m.jwt == nil {
		return func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "websocket not configured", http.StatusInternalServerError)
		}
	}
	return ws.WSHandler(m.wsHub, m.jwt, m.wsPres, m.wsChannels)
}

func (m *Module) RegisterJobs(j contract.JobRegistry) {}

func (m *Module) RegisterEvents(e contract.EventBus) {}

var _ contract.Module = (*Module)(nil)
