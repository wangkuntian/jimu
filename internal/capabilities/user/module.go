package user

import (
	"embed"
	"fmt"

	"jimu/internal/capabilities/encryption"
	"jimu/internal/capabilities/notification"
	"jimu/internal/capabilities/outbox"
	"jimu/internal/capabilities/user/application"
	"jimu/internal/capabilities/user/infrastructure"
	"jimu/internal/capabilities/user/interfaces"
	"jimu/internal/config"
	"jimu/internal/contract"
	"jimu/internal/kernel/cache"

	redistore "jimu/internal/kernel/redis"

	"gorm.io/gorm"
)

type Module struct {
	service *application.UserService
	rdb     redistore.Client
	outbox  *outbox.Outbox
}

func New(db *gorm.DB, cfg config.Config, deps ...interface{}) *Module {
	repo := infrastructure.NewMysqlRepository(db)
	var c cache.Cache
	var ob *outbox.Outbox
	var cipher *encryption.Cipher
	for _, dep := range deps {
		switch d := dep.(type) {
		case redistore.Client:
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
		if rdb, ok := dep.(redistore.Client); ok {
			m.rdb = rdb
			break
		}
	}
	return m
}

func (m *Module) Name() string {
	return "user"
}

// migrationsFS 能力自带迁移（Task 3：能力迁移经 embed 进二进制）。
//
//go:embed migrations
var migrationsFS embed.FS

// Descriptor 声明用户能力的静态描述。
var Descriptor = contract.Descriptor{
	Name:       "user",
	Migrations: migrationsFS,
	Mount:      contract.MountProtected,
}

// Descriptor 实现 contract.Describable。
func (m *Module) Descriptor() contract.Descriptor { return Descriptor }

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
