package authmodule

import (
	"embed"
	"time"

	"jimu/internal/capabilities/auth/application"
	authinfra "jimu/internal/capabilities/auth/infrastructure"
	"jimu/internal/capabilities/auth/interfaces"
	"jimu/internal/capabilities/outbox"
	"jimu/internal/capabilities/user/infrastructure"
	"jimu/internal/config"
	"jimu/internal/contract"
	"jimu/internal/kernel/access"
	"jimu/internal/kernel/auth"

	redistore "jimu/internal/kernel/redis"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type Module struct {
	cfg     config.AuthConfig
	service *application.AuthService
	jwtUtil *auth.JWT
	limiter *auth.Limiter
	db      *gorm.DB
	captcha contract.CaptchaVerifier
	outbox  *outbox.Outbox
}

// New 创建 auth 模块。
// deps 接受：*outbox.Outbox、notification.Dispatcher、*encryption.Cipher、
// application.TenantQuota、contract.MFAVerifier、contract.TenantProvisioner、
// contract.BreachChecker、*application.ResetStore。
func New(db *gorm.DB, rdb redistore.Client, cfg config.AuthConfig, failClosed bool, captchaVerifier contract.CaptchaVerifier, deps ...interface{}) *Module {
	userRepo := infrastructure.NewMysqlRepository(db)
	jwtUtil := auth.NewWithRotation(cfg.JWTSecret, cfg.JWTPreviousSecret, cfg.Issuer, cfg.AccessExpireMin, cfg.RefreshExpireDay)
	sessionStore := auth.NewRedisSessionStore(rdb)
	limiter := auth.NewLimiter(rdb, failClosed)
	lockoutTracker := auth.NewLoginFailureTracker(rdb, auth.DefaultLockoutConfig())
	// 密码重置验证码存储：redis 一次性码，TTL 取配置
	resetStore := application.NewResetStore(rdb, time.Duration(cfg.ResetCodeTTLMin)*time.Minute)
	loginHistoryRepo := authinfra.NewMysqlLoginHistoryRepository(db)
	passwordHistoryRepo := authinfra.NewMysqlPasswordHistoryRepository(db)
	allDeps := append([]interface{}{}, deps...)
	allDeps = append(allDeps, resetStore, application.WithPasswordHistory(cfg.PasswordHistoryCount),
		loginHistoryRepo, passwordHistoryRepo)
	service := application.NewAuthService(userRepo, jwtUtil, sessionStore, lockoutTracker, cfg.AccessExpireMin, allDeps...)
	m := &Module{cfg: cfg, service: service, jwtUtil: jwtUtil, limiter: limiter, db: db, captcha: captchaVerifier}
	for _, dep := range deps {
		if ob, ok := dep.(*outbox.Outbox); ok {
			m.outbox = ob
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
var Descriptor = contract.Descriptor{
	Name:       "auth",
	Migrations: migrationsFS,
	Requires:   []string{"user", "role", "tenant", "mfa"},
	Mount:      contract.MountSelfManaged,
}

// Descriptor 实现 contract.Describable。
func (m *Module) Descriptor() contract.Descriptor { return Descriptor }

// Finalizer 暴露 auth 服务作为 contract.LoginFinalizer，供 passkey 无密码登录复用登录收尾。
func (m *Module) Finalizer() contract.LoginFinalizer { return m.service }

func (m *Module) RegisterHTTP(r contract.Router) {
	interfaces.RegisterAuthRoutes(r.Group("/api/v1"), m.service, m.jwtUtil, m.cfg, m.limiter, m.captcha)
}

func (m *Module) ProtectedHTTPMiddleware() ([]gin.HandlerFunc, error) {
	enforcer, err := access.NewPathEnforcer()
	if err != nil {
		return nil, err
	}
	return interfaces.ProtectedMiddleware(m.jwtUtil, access.NewDBAuthorizationStore(m.db), enforcer), nil
}

func (m *Module) RegisterJobs(j contract.JobRegistry) {}

func (m *Module) RegisterEvents(e contract.EventBus) {}

var _ contract.Module = (*Module)(nil)
