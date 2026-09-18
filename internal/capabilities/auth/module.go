package authmodule

import (
	"time"

	"jimu/internal/capabilities/auth/application"
	authinfra "jimu/internal/capabilities/auth/infrastructure"
	"jimu/internal/capabilities/auth/interfaces"
	"jimu/internal/capabilities/captcha"
	"jimu/internal/capabilities/outbox"
	"jimu/internal/capabilities/user/infrastructure"
	"jimu/internal/config"
	"jimu/internal/contract"
	"jimu/internal/kernel/auth"

	redistore "jimu/internal/kernel/redis"

	"github.com/go-webauthn/webauthn/webauthn"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type Module struct {
	cfg        config.AuthConfig
	service    *application.AuthService
	jwtUtil    *auth.JWT
	limiter    *auth.Limiter
	db         *gorm.DB
	captcha    *captcha.Service
	captchaCfg config.CaptchaConfig
	outbox     *outbox.Outbox
}

func New(db *gorm.DB, rdb redistore.Client, cfg config.AuthConfig, failClosed bool, captchaSvc *captcha.Service, captchaCfg config.CaptchaConfig, deps ...interface{}) *Module {
	userRepo := infrastructure.NewMysqlRepository(db)
	jwtUtil := auth.NewWithRotation(cfg.JWTSecret, cfg.JWTPreviousSecret, cfg.Issuer, cfg.AccessExpireMin, cfg.RefreshExpireDay)
	sessionStore := auth.NewRedisSessionStore(rdb)
	limiter := auth.NewLimiter(rdb, failClosed)
	lockoutTracker := auth.NewLoginFailureTracker(rdb, auth.DefaultLockoutConfig())
	// 密码重置验证码存储：redis 一次性码，TTL 取配置
	resetStore := application.NewResetStore(rdb, time.Duration(cfg.ResetCodeTTLMin)*time.Minute)
	loginHistoryRepo := authinfra.NewMysqlLoginHistoryRepository(db)
	passwordHistoryRepo := authinfra.NewMysqlPasswordHistoryRepository(db)
	trustedDeviceRepo := authinfra.NewMysqlTrustedDeviceRepository(db)
	webauthnRepo := authinfra.NewMysqlWebAuthnCredentialRepository(db)
	allDeps := append(deps, resetStore, rdb, application.WithIssuer(cfg.Issuer), loginHistoryRepo,
		passwordHistoryRepo, application.WithPasswordHistory(cfg.PasswordHistoryCount),
		trustedDeviceRepo, application.WithTrustedDeviceTTL(cfg.TrustedDeviceDays),
		webauthnRepo, application.WithWebAuthnSessionTTL(time.Duration(cfg.WebAuthn.SessionTTLMin)*time.Minute))
	// WebAuthn/通行密钥：仅在启用时构造库句柄（配置合法性已由 config.Validate 保证）
	if cfg.WebAuthn.Enabled {
		handle, err := webauthn.New(&webauthn.Config{
			RPDisplayName: cfg.WebAuthn.RPDisplayName,
			RPID:          cfg.WebAuthn.RPID,
			RPOrigins:     cfg.WebAuthn.RPOrigins,
		})
		if err != nil {
			return nil
		}
		allDeps = append(allDeps, handle)
	}
	// 开通式注册：注册 = 开通新租户（单事务，模板模式初始化角色权限）
	if cfg.Provisioning.Enabled {
		allDeps = append(allDeps, application.NewGormTenantProvisioner(db, cfg.Provisioning))
	}
	service := application.NewAuthService(userRepo, jwtUtil, sessionStore, lockoutTracker, cfg.AccessExpireMin, allDeps...)
	m := &Module{cfg: cfg, service: service, jwtUtil: jwtUtil, limiter: limiter, db: db, captcha: captchaSvc, captchaCfg: captchaCfg}
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

// Descriptor 声明认证能力的静态描述。
var Descriptor = contract.Descriptor{
	Name:     "auth",
	Requires: []string{"user", "role", "tenant"},
	Mount:    contract.MountSelfManaged,
}

// Descriptor 实现 contract.Describable。
func (m *Module) Descriptor() contract.Descriptor { return Descriptor }

func (m *Module) RegisterHTTP(r contract.Router) {
	interfaces.RegisterAuthRoutes(r.Group("/api/v1"), m.service, m.jwtUtil, m.cfg, m.limiter, m.captcha, m.captchaCfg)
	interfaces.RegisterCaptchaRoute(r.Group("/api/v1"), m.captcha)
}

func (m *Module) ProtectedHTTPMiddleware() ([]gin.HandlerFunc, error) {
	enforcer, err := auth.NewPathEnforcer()
	if err != nil {
		return nil, err
	}
	return interfaces.ProtectedMiddleware(m.jwtUtil, auth.NewDBAuthorizationStore(m.db), enforcer), nil
}

func (m *Module) RegisterJobs(j contract.JobRegistry) {}

func (m *Module) RegisterEvents(e contract.EventBus) {}
