package authmodule

import (
	"jimu/internal/config"
	"jimu/internal/contract"
	"jimu/internal/modules/auth/application"
	"jimu/internal/modules/auth/interfaces"
	"jimu/internal/modules/user/infrastructure"
	"jimu/internal/platform/auth"

	"gorm.io/gorm"
)

type Module struct {
	service *application.AuthService
	jwtUtil *auth.JWT
}

func New(db *gorm.DB, cfg config.AuthConfig) *Module {
	userRepo := infrastructure.NewMysqlRepository(db)
	jwtUtil := auth.New(cfg.JWTSecret, cfg.AccessExpireMin, cfg.RefreshExpireDay)
	service := application.NewAuthService(userRepo, jwtUtil, cfg.AccessExpireMin)
	return &Module{service: service, jwtUtil: jwtUtil}
}

func (m *Module) Name() string {
	return "auth"
}

func (m *Module) RegisterHTTP(r contract.Router) {
	interfaces.RegisterAuthRoutes(r.Group("/api/v1"), m.service, m.jwtUtil)
}

func (m *Module) RegisterJobs(j contract.JobRegistry) {}

func (m *Module) RegisterEvents(e contract.EventBus) {}
