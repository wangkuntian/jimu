package captcha

import (
	"time"

	"jimu/internal/contract"
	redistore "jimu/internal/kernel/redis"

	"github.com/gin-gonic/gin"
)

// Module 图形验证码能力：公开 GET /api/v1/captcha。
type Module struct {
	svc *Service
}

// New 创建验证码模块。
func New(rdb redistore.Client, ttl time.Duration, enabled bool) *Module {
	return &Module{svc: NewServiceWithEnabled(rdb, ttl, enabled)}
}

// Name 模块名。
func (m *Module) Name() string { return "captcha" }

// Service 暴露验证码服务（实现 contract.CaptchaVerifier），供 auth 登录/注册注入。
func (m *Module) Service() *Service { return m.svc }

// Descriptor 声明验证码能力的静态描述（公开挂载，无迁移、无权限点）。
var Descriptor = contract.Descriptor{
	Name:  "captcha",
	Mount: contract.MountPublic,
}

// Descriptor 实现 contract.Describable。
func (m *Module) Descriptor() contract.Descriptor { return Descriptor }

// RegisterHTTP 注册验证码生成路由。
func (m *Module) RegisterHTTP(r contract.Router) {
	r.GET("/api/v1/captcha", NewCaptchaHandler(m.svc).Generate)
}

func (m *Module) RegisterJobs(j contract.JobRegistry) {}

func (m *Module) RegisterEvents(e contract.EventBus) {}

// 显式导入 gin 以保留 handler 返回类型接线（NewCaptchaHandler 使用 gin.Context）。
var _ = gin.HandlerFunc(nil)

var _ contract.Module = (*Module)(nil)
