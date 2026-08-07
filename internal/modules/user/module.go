package user

import (
	"fmt"

	"jimu/internal/config"
	"jimu/internal/contract"
	"jimu/internal/modules/user/application"
	"jimu/internal/modules/user/infrastructure"
	"jimu/internal/modules/user/interfaces"
	"jimu/internal/platform/cache"
	"jimu/internal/platform/event"
	"jimu/internal/platform/notification"
	"jimu/internal/platform/outbox"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type Module struct {
	service  *application.UserService
	rdb      *redis.Client
	eventBus contract.EventBus
	outbox   *outbox.Outbox
}

func New(db *gorm.DB, cfg config.Config, deps ...interface{}) *Module {
	repo := infrastructure.NewMysqlRepository(db)
	var c cache.Cache
	var ob *outbox.Outbox
	for _, dep := range deps {
		switch d := dep.(type) {
		case *redis.Client:
			c = cache.NewRedisCache(d, cfg.Cache.Prefix)
		case *outbox.Outbox:
			ob = d
		}
	}
	eb := event.New()
	service := application.NewUserService(repo, c, eb, ob)
	m := &Module{service: service, eventBus: eb, outbox: ob}
	for _, dep := range deps {
		if rdb, ok := dep.(*redis.Client); ok {
			m.rdb = rdb
			break
		}
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

// RegisterEvents 注册用户事件处理器
func (m *Module) RegisterEvents(e contract.EventBus) {
	// 将本地事件总线桥接到全局事件总线
	m.eventBus.Subscribe(contract.EventUserCreated, func(payload interface{}) {
		if evt, ok := payload.(contract.UserCreatedEvent); ok {
			e.Publish(contract.UserCreatedEmailNotification, notification.Message{
				Channel: notification.ChannelEmail,
				To:      "", // 实际应从用户服务获取邮箱
				Subject: "Welcome to Jimu",
				Body:    fmt.Sprintf("Hi %s, your account has been created successfully.", evt.Username),
				Data: map[string]string{
					"username": evt.Username,
				},
			})
		}
	})

	m.eventBus.Subscribe(contract.EventUserDeleted, func(payload interface{}) {
		if evt, ok := payload.(contract.UserDeletedEvent); ok {
			e.Publish(contract.UserDeletedEventLog, evt)
		}
	})
}
