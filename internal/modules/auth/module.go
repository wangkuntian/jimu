package authmodule

import (
	"jimu/internal/config"
	"jimu/internal/contract"
	"jimu/internal/modules/auth/application"
	"jimu/internal/modules/auth/interfaces"
	"jimu/internal/modules/user/infrastructure"
	"jimu/internal/platform/auth"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type Module struct {
	cfg     config.AuthConfig
	service *application.AuthService
	jwtUtil *auth.JWT
	limiter *auth.Limiter
	db      *gorm.DB
}

func New(db *gorm.DB, rdb *redis.Client, cfg config.AuthConfig, failClosed bool) *Module {
	userRepo := infrastructure.NewMysqlRepository(db)
	jwtUtil := auth.New(cfg.JWTSecret, cfg.Issuer, cfg.AccessExpireMin, cfg.RefreshExpireDay)
	sessionStore := auth.NewRedisSessionStore(rdb)
	limiter := auth.NewLimiter(rdb, failClosed)
	lockoutTracker := auth.NewLoginFailureTracker(rdb, auth.DefaultLockoutConfig())
	service := application.NewAuthService(userRepo, jwtUtil, sessionStore, lockoutTracker, cfg.AccessExpireMin)
	return &Module{cfg: cfg, service: service, jwtUtil: jwtUtil, limiter: limiter, db: db}
}

func (m *Module) Name() string {
	return "auth"
}

func (m *Module) RegisterHTTP(r contract.Router) {
	interfaces.RegisterAuthRoutes(r.Group("/api/v1"), m.service, m.jwtUtil, m.cfg, m.limiter)
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
