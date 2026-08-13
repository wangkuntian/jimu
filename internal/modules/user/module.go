package user

import (
	"fmt"

	"jimu/internal/config"
	"jimu/internal/contract"
	"jimu/internal/modules/user/application"
	"jimu/internal/modules/user/infrastructure"
	"jimu/internal/modules/user/interfaces"
	"jimu/internal/platform/cache"
	"jimu/internal/platform/encryption"
	"jimu/internal/platform/notification"
	"jimu/internal/platform/outbox"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type Module struct {
	service *application.UserService
	rdb     *redis.Client
	outbox  *outbox.Outbox
}

func New(db *gorm.DB, cfg config.Config, deps ...interface{}) *Module {
	repo := infrastructure.NewMysqlRepository(db)
	var c cache.Cache
	var ob *outbox.Outbox
	var cipher *encryption.Cipher
	for _, dep := range deps {
		switch d := dep.(type) {
		case *redis.Client:
			c = cache.NewRedisCache(d, cfg.Cache.Prefix)
		case *outbox.Outbox:
			ob = d
		case *encryption.Cipher:
			cipher = d
		}
	}
	service := application.NewUserService(repo, c, ob, cipher)
	m := &Module{service: service, outbox: ob}
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

// RegisterEvents 注册用户事件处理器（订阅全局事件总线的裸业务主题）
func (m *Module) RegisterEvents(e contract.EventBus) {
	// 订阅全局总线的用户创建事件，桥接到通知系统
	e.Subscribe(contract.EventUserCreated, func(payload interface{}) {
		if evt, ok := payload.(contract.UserCreatedEvent); ok {
			e.Publish(contract.UserCreatedEmailNotification, notification.Message{
				Channel: notification.ChannelEmail,
				To:      evt.Email,
				Subject: "Welcome to Jimu",
				Body:    fmt.Sprintf("Hi %s, your account has been created successfully.", evt.Username),
				Data: map[string]string{
					"username": evt.Username,
				},
			})
		}
	})

	e.Subscribe(contract.EventUserDeleted, func(payload interface{}) {
		if evt, ok := payload.(contract.UserDeletedEvent); ok {
			e.Publish(contract.UserDeletedEventLog, evt)
		}
	})
}
