package captcha

import (
	"time"

	"jimu/internal/contract"

	"github.com/gin-gonic/gin"
)

// Module 图形验证码能力：公开 GET /api/v1/captcha。
type Module struct {
	svc *Service
}

// New 创建验证码模块。
func New(cache contract.Cache, ttl time.Duration, enabled bool) *Module {
	return &Module{svc: NewServiceWithEnabled(cache, ttl, enabled)}
}

// Name 模块名。
func (m *Module) Name() string { return "captcha" }

// Service 暴露验证码服务（实现 contract.CaptchaVerifier），供 auth 登录/注册注入。
func (m *Module) Service() *Service { return m.svc }

// Descriptor 声明验证码能力的静态描述（公开挂载，无迁移、无权限点）。
// cache 是软依赖：有 cache 端口时用共享缓存（跨实例有效），没有时降级到进程内，
// 与 auth 的存储降级同口径 —— 缺可选组件不该让能力无法组合。
var Descriptor = contract.Descriptor{
	Name:         "captcha",
	SoftRequires: []string{"cache"},
	Mount:        contract.MountPublic,
	Provides:     []contract.ProvidedPort{{Name: PortName}},
	Consumes:     []contract.ConsumedPort{{Name: contract.CachePortName}},
	Configs: []contract.ConfigSpec{
		{Section: ConfigKey, New: func() any { return &Config{} }},
	},
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
