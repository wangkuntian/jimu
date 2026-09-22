package mfa

import (
	"embed"

	mfaapp "jimu/internal/capabilities/mfa/application"
	mfainfra "jimu/internal/capabilities/mfa/infrastructure"
	"jimu/internal/capabilities/mfa/interfaces"
	"jimu/internal/contract"
	"jimu/internal/kernel/auth"

	"gorm.io/gorm"
)

// Module TOTP 二次验证 + 可信设备能力。
type Module struct {
	service *mfaapp.MFAService
	jwtUtil *auth.JWT
}

// New 创建 mfa 模块。trustedDeviceDays<=0 关闭可信设备「记住此设备」；
// users 用于 otpauth account 兜底（可为 nil）。
func New(db *gorm.DB, cfg Config, users contract.UserinfoSource) *Module {
	repo := mfainfra.NewMysqlMFARepository(db)
	trustedRepo := mfainfra.NewMysqlTrustedDeviceRepository(db)
	svc := mfaapp.NewMFAService(repo, users, trustedRepo, cfg.TrustedDeviceDays, cfg.Issuer)
	jwtUtil := auth.NewWithRotation(cfg.JWTSecret, cfg.JWTPreviousSecret, cfg.Issuer, cfg.AccessExpireMin, cfg.RefreshExpireDay)
	return &Module{service: svc, jwtUtil: jwtUtil}
}

// Name 模块名。
func (m *Module) Name() string { return "mfa" }

// Service 暴露 MFA 服务，供装配期作为 contract.MFAVerifier 注入 auth。
func (m *Module) Service() *mfaapp.MFAService { return m.service }

// migrationsFS 能力自带迁移（//go:embed 进二进制）。
//
//go:embed migrations
var migrationsFS embed.FS

// Descriptor 声明 mfa 能力的静态描述。
var Descriptor = contract.Descriptor{
	Name:         "mfa",
	Requires:     []string{"user"},
	SoftRequires: []string{"auth"},
	Migrations:   migrationsFS,
	Owns:         []string{"user_mfa", "trusted_devices"},
	Mount:        contract.MountSelfManaged,
	Permissions: []contract.Permission{
		{Name: "MFA 绑定密钥", Resource: "/api/v1/auth/mfa/setup", Action: "POST"},
		{Name: "MFA 启用", Resource: "/api/v1/auth/mfa/enable", Action: "POST"},
		{Name: "MFA 关闭", Resource: "/api/v1/auth/mfa/disable", Action: "POST"},
		{Name: "可信设备列表", Resource: "/api/v1/auth/devices", Action: "GET"},
		{Name: "可信设备全部注销", Resource: "/api/v1/auth/devices", Action: "DELETE"},
		{Name: "可信设备注销", Resource: "/api/v1/auth/devices/*", Action: "DELETE"},
	},
}

// Descriptor 实现 contract.Describable。
func (m *Module) Descriptor() contract.Descriptor { return Descriptor }

// RegisterHTTP 注册 MFA/可信设备路由（自管理受保护分组）。
func (m *Module) RegisterHTTP(r contract.Router) {
	rg := r.Group("/api/v1")
	protected := rg.Group("/auth")
	protected.Use(auth.AuthMiddleware(m.jwtUtil))
	interfaces.RegisterMFARoutes(protected, m.service)
}

func (m *Module) RegisterJobs(j contract.JobRegistry) {}

func (m *Module) RegisterEvents(e contract.EventBus) {}

var _ contract.Module = (*Module)(nil)
