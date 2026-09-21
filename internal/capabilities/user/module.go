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
	"jimu/internal/kernel/http/middleware"

	redistore "jimu/internal/kernel/redis"

	"gorm.io/gorm"
)

type Module struct {
	service *application.UserService
	admin   *application.AdminUserService
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
	adminSvc := application.NewAdminUserService(repo, service).WithDB(db)
	m := &Module{service: service, admin: adminSvc, outbox: ob}
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

// WithRoles 注入用户角色分配端口（access 能力提供，装配期调用）。
func (m *Module) WithRoles(roles application.UserRoleAssigner) *Module {
	m.admin.WithRoles(roles)
	return m
}

// WithQuota 注入租户配额校验（tenant 能力提供，装配期调用）。
func (m *Module) WithQuota(quota application.TenantQuota) *Module {
	m.admin.WithQuota(quota)
	return m
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
	Permissions: []contract.Permission{
		{Name: "用户列表", Resource: "/api/v1/users", Action: "GET"},
		{Name: "用户创建", Resource: "/api/v1/users", Action: "POST"},
		{Name: "用户详情", Resource: "/api/v1/users/*", Action: "GET"},
		{Name: "用户修改", Resource: "/api/v1/users/*", Action: "PUT"},
		{Name: "用户删除", Resource: "/api/v1/users/*", Action: "DELETE"},
	},
}

// Descriptor 实现 contract.Describable。
func (m *Module) Descriptor() contract.Descriptor { return Descriptor }

func (m *Module) RegisterHTTP(r contract.Router) {
	rg := r.Group("/api/v1")
	interfaces.RegisterUserRoutes(rg, m.service, m.rdb)

	// 管理面用户用例：与管理面同一 repository/配额/租户可见性
	// （/users/:id/roles 归 access 注册，此处不重复）
	admin := rg.Group("/admin")
	admin.Use(middleware.AdminAuth())
	interfaces.RegisterAdminUserRoutes(admin, m.admin)
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
