package user

import (
	"embed"
	"fmt"

	"jimu/internal/capabilities/user/application"
	"jimu/internal/capabilities/user/infrastructure"
	"jimu/internal/capabilities/user/interfaces"
	"jimu/internal/contract"
	"jimu/internal/kernel/auth"

	redistore "jimu/internal/kernel/redis"

	"gorm.io/gorm"
)

type Module struct {
	service *application.UserService
	rdb     redistore.Client
}

func New(db *gorm.DB, deps ...interface{}) *Module {
	repo := infrastructure.NewMysqlRepository(db)
	var ob contract.EventWriter
	var cipher contract.BlindIndexer
	var roles contract.UserRoleAssigner
	var tenants contract.TenantInfoSource
	var quota application.TenantQuota
	for _, dep := range deps {
		switch d := dep.(type) {
		case contract.EventWriter:
			ob = d
		case contract.BlindIndexer:
			cipher = d
		case contract.UserRoleAssigner:
			roles = d
		case contract.TenantInfoSource:
			tenants = d
		case application.TenantQuota:
			quota = d
		}
	}
	service := application.NewUserService(repo, ob, cipher, roles, tenants, quota).WithDB(db)
	m := &Module{service: service}
	for _, dep := range deps {
		if rdb, ok := dep.(redistore.Client); ok {
			m.rdb = rdb
			break
		}
	}
	if m.rdb != nil {
		m.service.WithSessionRevoker(auth.NewRedisSessionStore(m.rdb))
	}
	return m
}

func (m *Module) Name() string {
	return "user"
}

// WithRoles 注入用户角色分配端口（access 能力提供，装配期调用）。
func (m *Module) WithRoles(roles contract.UserRoleAssigner) *Module {
	m.service.WithRoles(roles)
	return m
}

// WithQuota 注入租户配额校验（tenant 能力提供，装配期调用）。
func (m *Module) WithQuota(quota application.TenantQuota) *Module {
	m.service.WithQuota(quota)
	return m
}

// WithTenantState 注入租户 active 校验（tenant 能力提供，装配期调用）。
func (m *Module) WithTenantState(states contract.TenantStateSource) *Module {
	m.service.WithTenantState(states)
	return m
}

// migrationsFS 能力自带迁移（Task 3：能力迁移经 embed 进二进制）。
//
//go:embed migrations
var migrationsFS embed.FS

// Descriptor 声明用户能力的静态描述。
var Descriptor = contract.Descriptor{
	Name:                "user",
	SoftRequires:        []string{"access", "tenant"},
	Migrations:          migrationsFS,
	Owns:                []string{"users"},
	Mount:               contract.MountProtected,
	MigrationSchemaDeps: []string{"tenant"},
	RuntimeServices:     []contract.RuntimeService{contract.RuntimeServiceMySQL},
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
}

func (m *Module) RegisterJobs(j contract.JobRegistry) {}

// RegisterEvents 注册用户事件处理器（订阅全局事件总线的裸业务主题）
func (m *Module) RegisterEvents(e contract.EventBus) {
	// 订阅全局总线的用户创建事件，桥接到通知系统
	e.Subscribe(contract.EventUserCreated, func(payload interface{}) {
		if evt, ok := payload.(contract.UserCreatedEvent); ok {
			e.Publish(contract.UserCreatedEmailNotification, contract.NotificationMessage{
				Channel: "email",
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
