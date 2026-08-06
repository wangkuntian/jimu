package admin

import (
	"jimu/internal/contract"
	adminapp "jimu/internal/modules/admin/application"
	admininterfaces "jimu/internal/modules/admin/interfaces"

	"github.com/redis/go-redis/v9"
)

// Module 管理模块
type Module struct {
	service *adminapp.Service
}

// New 创建管理模块
func New(version, env string, rdb *redis.Client) *Module {
	return &Module{
		service: adminapp.NewService(version, env, rdb),
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
