package user

import (
	"jimu/internal/config"
	"jimu/internal/contract"
	"jimu/internal/modules/user/application"
	"jimu/internal/modules/user/infrastructure"
	"jimu/internal/modules/user/interfaces"
	"jimu/internal/platform/cache"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type Module struct {
	service *application.UserService
	rdb     *redis.Client
}

func New(db *gorm.DB, cfg config.Config, rdb ...*redis.Client) *Module {
	repo := infrastructure.NewMysqlRepository(db)
	var c cache.Cache
	if len(rdb) > 0 {
		c = cache.NewRedisCache(rdb[0], cfg.Cache.Prefix)
	}
	service := application.NewUserService(repo, c)
	m := &Module{service: service}
	if len(rdb) > 0 {
		m.rdb = rdb[0]
	}
	return m
}

func (m *Module) Name() string {
	return "user"
}

func (m *Module) RegisterHTTP(r contract.Router) {
	interfaces.RegisterUserRoutes(r.Group("/api/v1"), m.service, m.rdb)
}

func (m *Module) RegisterJobs(j contract.JobRegistry) {}

func (m *Module) RegisterEvents(e contract.EventBus) {}
