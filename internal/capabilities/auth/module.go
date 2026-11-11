package authmodule

import (
	"embed"

	"jimu/internal/capabilities/auth/application"
	"jimu/internal/capabilities/auth/authstore"
	authinfra "jimu/internal/capabilities/auth/infrastructure"
	"jimu/internal/capabilities/auth/interfaces"
	"jimu/internal/contract"
	"jimu/internal/platform/authz"
	"jimu/internal/platform/identity"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// PortName auth 能力对外提供的端口名：contract.LoginFinalizer（供 passkey 复用登录收尾）。
const PortName = "auth"

type Module struct {
	cfg                 Config
	service             *application.AuthService
	jwtUtil             *identity.JWT
	limiter             *identity.Limiter
	db                  *gorm.DB
	captcha             contract.CaptchaVerifier
	outbox              contract.EventWriter
	provisioningEnabled bool
	tenants             contract.TenantStateSource
	users               contract.UserStateSource
}

// New 创建 auth 模块。
// deps 接受：*outbox.Outbox、notification.Dispatcher、*encryption.Cipher、
// application.TenantQuota、contract.MFAVerifier、contract.TenantProvisioner、
// contract.BreachChecker、ResetCodeStore。
func New(db *gorm.DB, stores authstore.Stores, cfg Config, failClosed bool, captchaVerifier contract.CaptchaVerifier, deps ...interface{}) *Module {
	var userRepo contract.AccountRepository
	var provisioningEnabled bool
	for _, dep := range deps {
		if repo, ok := dep.(contract.AccountRepository); ok {
			userRepo = repo
		}
		if _, ok := dep.(contract.TenantProvisioner); ok {
			provisioningEnabled = true
		}
	}
	jwtUtil := identity.NewWithRotation(cfg.JWTSecret, cfg.JWTPreviousSecret, cfg.Issuer, cfg.AccessExpireMin, cfg.RefreshExpireDay)
	limiter := identity.NewLimiter(stores.RateCounter, failClosed)
	lockoutTracker := identity.NewLoginFailureTracker(stores.Failures, identity.DefaultLockoutConfig())
	loginHistoryRepo := authinfra.NewMysqlLoginHistoryRepository(db)
	passwordHistoryRepo := authinfra.NewMysqlPasswordHistoryRepository(db)
	allDeps := append([]interface{}{}, deps...)
	allDeps = append(allDeps, stores.ResetCodes, application.WithPasswordHistory(cfg.PasswordHistoryCount),
		loginHistoryRepo, passwordHistoryRepo, authz.NewDBAuthorizationStore(db))
	service := application.NewAuthService(userRepo, jwtUtil, stores.Sessions, lockoutTracker, cfg.AccessExpireMin, allDeps...)
	m := &Module{cfg: cfg, service: service, jwtUtil: jwtUtil, limiter: limiter, db: db, captcha: captchaVerifier,
		provisioningEnabled: provisioningEnabled}
	for _, dep := range deps {
		if ob, ok := dep.(contract.EventWriter); ok {
			m.outbox = ob
		}
		if tenants, ok := dep.(contract.TenantStateSource); ok {
			m.tenants = tenants
		}
		if users, ok := dep.(contract.UserStateSource); ok {
			m.users = users
		}
	}
	return m
}

func (m *Module) Name() string {
	return "auth"
}

// migrationsFS 能力自带迁移（能力迁移经 embed 进二进制）。
//
//go:embed migrations
var migrationsFS embed.FS

// Descriptor 声明认证能力的静态描述。
// tenant/mfa 是软依赖：缺失时开通式注册关闭、二次验证跳过（minimal profile 无租户/MFA）。
var Descriptor = contract.Descriptor{
	Name:         "auth",
	Migrations:   migrationsFS,
	Requires:     []string{"user", "access"},
	SoftRequires: []string{"tenant", "mfa", "captcha", "breach"},
	Owns:         []string{"login_histories", "password_histories"},
	Mount:        contract.MountSelfManaged,
	Drivers:      []string{"redis", "memory"},
	DriverSpecs: []contract.DriverSpec{
		{Name: "redis", RuntimeServices: []contract.RuntimeService{contract.RuntimeServiceRedis}},
		{Name: "memory"},
	},
	RuntimeServices: []contract.RuntimeService{contract.RuntimeServiceMySQL},
	Provides:        []contract.ProvidedPort{{Name: PortName}},
	Consumes: []contract.ConsumedPort{
		{Name: "user.account", Required: true},
		{Name: "access.provisioning"},
		{Name: "tenant.provisioner"},
		{Name: "tenant"},
		{Name: contract.TenantInfoPortName},
		{Name: contract.UserinfoPortName},
		{Name: "mfa"},
		{Name: "captcha"},
		{Name: "outbox"},
		{Name: "notification"},
		{Name: "encryption"},
		{Name: "breach"},
	},
	ProtectedHTTPMiddleware: contract.ProtectedHTTPMiddlewarePrimary,
	// auth 拥有整个 auth 段（含嵌套 webauthn/provisioning），不拆段（设计 §8 ¶2）
	Configs: []contract.ConfigSpec{
		{Section: ConfigKey, New: func() any { return &Config{} }},
	},
}

// Descriptor 实现 contract.Describable。
func (m *Module) Descriptor() contract.Descriptor { return Descriptor }

// Finalizer 暴露 auth 服务作为 contract.LoginFinalizer，供 passkey 无密码登录复用登录收尾。
func (m *Module) Finalizer() contract.LoginFinalizer { return m.service }

func (m *Module) RegisterHTTP(r contract.Router) {
	interfaces.RegisterAuthRoutes(r.Group("/api/v1"), m.service, m.jwtUtil, interfaces.Config{
		PublicRegistration:    m.cfg.PublicRegistration,
		ProvisioningEnabled:   m.provisioningEnabled,
		LoginRateLimit:        m.cfg.LoginRateLimit,
		LoginRateWindowSec:    m.cfg.LoginRateWindowSec,
		RegisterRateLimit:     m.cfg.RegisterRateLimit,
		RegisterRateWindowSec: m.cfg.RegisterRateWindowSec,
	}, m.limiter, m.captcha, identity.StateCheckers{Tenant: m.tenants, User: m.users})
}

func (m *Module) ProtectedHTTPMiddleware() ([]gin.HandlerFunc, error) {
	enforcer, err := authz.NewPathEnforcer()
	if err != nil {
		return nil, err
	}
	return interfaces.ProtectedMiddleware(m.jwtUtil, authz.NewDBAuthorizationStore(m.db), enforcer,
		identity.StateCheckers{Tenant: m.tenants, User: m.users}), nil
}

func (m *Module) RegisterJobs(j contract.JobRegistry) {}

func (m *Module) RegisterEvents(e contract.EventBus) {}

var _ contract.Module = (*Module)(nil)
