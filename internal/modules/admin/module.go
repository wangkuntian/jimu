package admin

import (
	"jimu/internal/config"
	"jimu/internal/contract"
	adminapp "jimu/internal/modules/admin/application"
	admininterfaces "jimu/internal/modules/admin/interfaces"
)

// Module 管理模块
type Module struct {
	service *adminapp.Service
}

// New 创建管理模块
func New(cfg config.Config) *Module {
	return &Module{
		service: adminapp.NewService(cfg.Version, cfg.Environment),
	}
}

func (m *Module) Name() string {
	return "admin"
}

func (m *Module) RegisterHTTP(r contract.Router) {
	admininterfaces.RegisterRoutes(r.Group("/api/v1"), m.service)
}

func (m *Module) RegisterJobs(j contract.JobRegistry) {}

func (m *Module) RegisterEvents(e contract.EventBus) {}
