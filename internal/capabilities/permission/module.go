package permission

import (
	"embed"
	"jimu/internal/capabilities/permission/application"
	"jimu/internal/capabilities/permission/infrastructure"
	"jimu/internal/capabilities/permission/interfaces"
	"jimu/internal/contract"

	"gorm.io/gorm"
)

type Module struct {
	service *application.PermissionService
}

func New(db *gorm.DB) *Module {
	repo := infrastructure.NewMysqlPermissionRepository(db)
	service := application.NewPermissionService(repo)
	return &Module{service: service}
}

func (m *Module) Name() string {
	return "permission"
}

// migrationsFS 能力自带迁移（Task 3：能力迁移经 embed 进二进制）。
//
//go:embed migrations
var migrationsFS embed.FS

// Descriptor 声明权限能力的静态描述。
var Descriptor = contract.Descriptor{
	Name:       "permission",
	Migrations: migrationsFS,
	Requires:   []string{"role"},
	Mount:      contract.MountProtected,
}

// Descriptor 实现 contract.Describable。
func (m *Module) Descriptor() contract.Descriptor { return Descriptor }

func (m *Module) RegisterHTTP(r contract.Router) {
	interfaces.RegisterPermissionRoutes(r.Group("/api/v1"), m.service)
}

func (m *Module) RegisterJobs(j contract.JobRegistry) {}

func (m *Module) RegisterEvents(e contract.EventBus) {}
