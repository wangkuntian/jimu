package user

import (
	"jimu/internal/contract"
	"jimu/internal/modules/user/application"
	"jimu/internal/modules/user/infrastructure"
	"jimu/internal/modules/user/interfaces"

	"gorm.io/gorm"
)

type Module struct {
	service *application.UserService
}

func New(db *gorm.DB) *Module {
	repo := infrastructure.NewMysqlRepository(db)
	service := application.NewUserService(repo)
	return &Module{service: service}
}

func (m *Module) Name() string {
	return "user"
}

func (m *Module) RegisterHTTP(r contract.Router) {
	interfaces.RegisterUserRoutes(r.Group("/api/v1"), m.service)
}

func (m *Module) RegisterJobs(j contract.JobRegistry) {
	// No jobs in v1
}

func (m *Module) RegisterEvents(e contract.EventBus) {
	// No events in v1
}
