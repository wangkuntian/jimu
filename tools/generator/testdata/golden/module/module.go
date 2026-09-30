package product

import (
	"embed"

	"jimu/internal/capabilities/product/application"
	"jimu/internal/capabilities/product/infrastructure"
	"jimu/internal/capabilities/product/interfaces"
	"jimu/internal/contract"

	"gorm.io/gorm"
)

// migrationsFS 能力自带迁移（mysql/ 与 postgres/ 子目录随二进制嵌入）。
//
//go:embed migrations
var migrationsFS embed.FS

// Descriptor 能力静态描述：迁移与权限点由能力自声明，供迁移运行器与种子消费。
var Descriptor = contract.Descriptor{
	Name:       "product",
	Owns:       []string{"products"},
	Migrations: migrationsFS,
	Mount:      contract.MountProtected,
	Permissions: []contract.Permission{
		{Name: "Product列表", Resource: "/api/v1/products", Action: "GET"},
		{Name: "Product创建", Resource: "/api/v1/products", Action: "POST"},
		{Name: "Product详情", Resource: "/api/v1/products/*", Action: "GET"},
		{Name: "Product修改", Resource: "/api/v1/products/*", Action: "PUT"},
		{Name: "Product删除", Resource: "/api/v1/products/*", Action: "DELETE"},
	},
}

type Module struct {
	service *application.ProductService
}

func New(db *gorm.DB) *Module {
	repo := infrastructure.NewMysqlProductRepository(db)
	service := application.NewProductService(repo)
	return &Module{service: service}
}

func (m *Module) Name() string { return "product" }

// Descriptor 实现 contract.Describable。
func (m *Module) Descriptor() contract.Descriptor { return Descriptor }

func (m *Module) RegisterHTTP(r contract.Router) {
	interfaces.RegisterProductRoutes(r.Group("/api/v1"), m.service)
}

func (m *Module) RegisterJobs(j contract.JobRegistry) {}

func (m *Module) RegisterEvents(e contract.EventBus) {}
