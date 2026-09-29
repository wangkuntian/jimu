package passkey

import (
	"embed"
	"time"

	authmodule "jimu/internal/capabilities/auth"
	passkeyapp "jimu/internal/capabilities/passkey/application"
	passkeyinfra "jimu/internal/capabilities/passkey/infrastructure"
	"jimu/internal/capabilities/passkey/interfaces"
	"jimu/internal/contract"
	"jimu/internal/kernel/auth"

	redistore "jimu/internal/kernel/redis"

	"github.com/go-webauthn/webauthn/webauthn"
	"gorm.io/gorm"
)

// Module 通行密钥（WebAuthn）能力。
type Module struct {
	service *passkeyapp.PasskeyService
	jwtUtil *auth.JWT
	limiter *auth.Limiter
	cfg     authmodule.Config
}

// Deps passkey 模块的装配依赖。
type Deps struct {
	DB    *gorm.DB
	Redis redistore.Client
	// AuthCfg auth 能力配置（passkey.Requires 含 auth，方向合法）。
	// 整个 auth 段由 auth 能力拥有（含 auth.webauthn），故 passkey 不单列配置段。
	AuthCfg   authmodule.Config
	Users     contract.UserinfoSource
	Finalizer contract.LoginFinalizer
	// FailClosed 限流器在 Redis 故障时是否拒绝（与 auth 一致）
	FailClosed bool
}

// New 创建 passkey 模块。WebAuthn 未启用时凭证路由仍注册，但调用返回「未配置」。
func New(deps Deps) *Module {
	creds := passkeyinfra.NewMysqlWebAuthnCredentialRepository(deps.DB)
	var handle *webauthn.WebAuthn
	if deps.AuthCfg.WebAuthn.Enabled {
		// 配置合法性已由 config.Validate 保证；构造失败时保持 nil（调用返回未配置）
		if h, err := webauthn.New(&webauthn.Config{
			RPDisplayName: deps.AuthCfg.WebAuthn.RPDisplayName,
			RPID:          deps.AuthCfg.WebAuthn.RPID,
			RPOrigins:     deps.AuthCfg.WebAuthn.RPOrigins,
		}); err == nil {
			handle = h
		}
	}
	svc := passkeyapp.NewPasskeyService(passkeyapp.Deps{
		Users:       deps.Users,
		Credentials: creds,
		WebAuthn:    handle,
		Redis:       deps.Redis,
		SessionTTL:  time.Duration(deps.AuthCfg.WebAuthn.SessionTTLMin) * time.Minute,
		Finalizer:   deps.Finalizer,
	})
	jwtUtil := auth.NewWithRotation(deps.AuthCfg.JWTSecret, deps.AuthCfg.JWTPreviousSecret, deps.AuthCfg.Issuer, deps.AuthCfg.AccessExpireMin, deps.AuthCfg.RefreshExpireDay)
	var limiter *auth.Limiter
	if deps.Redis != nil {
		limiter = auth.NewLimiter(deps.Redis, deps.FailClosed)
	}
	return &Module{service: svc, jwtUtil: jwtUtil, limiter: limiter, cfg: deps.AuthCfg}
}

// Name 模块名。
func (m *Module) Name() string { return "passkey" }

// migrationsFS 能力自带迁移（//go:embed 进二进制）。
//
//go:embed migrations
var migrationsFS embed.FS

// Descriptor 声明 passkey 能力的静态描述。
var Descriptor = contract.Descriptor{
	Name:       "passkey",
	Requires:   []string{"user", "auth"},
	Migrations: migrationsFS,
	Owns:       []string{"webauthn_credentials"},
	Mount:      contract.MountSelfManaged,
	Permissions: []contract.Permission{
		{Name: "通行密钥登录开始", Resource: "/api/v1/auth/webauthn/login/begin", Action: "POST"},
		{Name: "通行密钥登录完成", Resource: "/api/v1/auth/webauthn/login/finish", Action: "POST"},
		{Name: "通行密钥注册开始", Resource: "/api/v1/auth/webauthn/register/begin", Action: "POST"},
		{Name: "通行密钥注册完成", Resource: "/api/v1/auth/webauthn/register/finish", Action: "POST"},
		{Name: "通行密钥列表", Resource: "/api/v1/auth/webauthn/credentials", Action: "GET"},
		{Name: "通行密钥重命名", Resource: "/api/v1/auth/webauthn/credentials/*", Action: "PUT"},
		{Name: "通行密钥删除", Resource: "/api/v1/auth/webauthn/credentials/*", Action: "DELETE"},
	},
}

// Descriptor 实现 contract.Describable。
func (m *Module) Descriptor() contract.Descriptor { return Descriptor }

// RegisterHTTP 注册通行密钥路由：login/begin、login/finish 公开；其余受 JWT 保护。
func (m *Module) RegisterHTTP(r contract.Router) {
	rg := r.Group("/api/v1")
	interfaces.RegisterPasskeyRoutes(rg, m.service, m.cfg, m.limiter, m.jwtUtil)
}

func (m *Module) RegisterJobs(j contract.JobRegistry) {}

func (m *Module) RegisterEvents(e contract.EventBus) {}

var _ contract.Module = (*Module)(nil)
