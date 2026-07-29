package generator

import (
	"fmt"
	"os"
	"path/filepath"
	"text/template"
	"unicode"
)

// GenerateModule 生成完整的模块骨架（Clean Architecture 分层）
func GenerateModule(name string) error {
	dirs := []string{
		filepath.Join("internal", "modules", name, "domain"),
		filepath.Join("internal", "modules", name, "application"),
		filepath.Join("internal", "modules", name, "infrastructure"),
		filepath.Join("internal", "modules", name, "interfaces"),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	// 生成各层文件
	files := map[string]string{
		filepath.Join("internal", "modules", name, "module.go"):                   moduleTemplate,
		filepath.Join("internal", "modules", name, "domain", "entity.go"):         entityTemplate,
		filepath.Join("internal", "modules", name, "domain", "repository.go"):     repositoryTemplate,
		filepath.Join("internal", "modules", name, "application", "service.go"):   serviceTemplate,
		filepath.Join("internal", "modules", name, "application", "dto.go"):        dtoTemplate,
		filepath.Join("internal", "modules", name, "infrastructure", "mysql_repository.go"): mysqlRepoTemplate,
		filepath.Join("internal", "modules", name, "interfaces", "handler.go"):    handlerTemplate,
		filepath.Join("internal", "modules", name, "interfaces", "router.go"):      routerTemplate,
	}

	for path, tmpl := range files {
		if err := writeTemplate(path, tmpl, name); err != nil {
			return fmt.Errorf("failed to create %s: %w", path, err)
		}
	}

	fmt.Printf("Module '%s' created at internal/modules/%s/\n", name, name)
	fmt.Println("  domain/           - 实体和仓储接口")
	fmt.Println("  application/      - 用例服务和 DTO")
	fmt.Println("  infrastructure/   - 数据库实现")
	fmt.Println("  interfaces/       - HTTP handler 和路由")
	return nil
}

func writeTemplate(path, tmpl, name string) error {
	t, err := template.New(name).Parse(tmpl)
	if err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return t.Execute(f, map[string]string{
		"Name":      name,
		"NameCamel": capitalize(name),
	})
}

func capitalize(s string) string {
	if s == "" {
		return ""
	}
	r := []rune(s)
	r[0] = unicode.ToUpper(r[0])
	return string(r)
}

const moduleTemplate = `package {{.Name}}

import (
	"jimu/internal/contract"
	"jimu/internal/modules/{{.Name}}/infrastructure"
	"jimu/internal/modules/{{.Name}}/application"
	"jimu/internal/modules/{{.Name}}/interfaces"
)

type Module struct {
	service *application.{{.NameCamel}}Service
}

func New(db *gorm.DB) *Module {
	repo := infrastructure.NewMysql{{.NameCamel}}Repository(db)
	service := application.New{{.NameCamel}}Service(repo)
	return &Module{service: service}
}

func (m *Module) Name() string {
	return "{{.Name}}"
}

func (m *Module) RegisterHTTP(r contract.Router) {
	interfaces.Register{{.NameCamel}}Routes(r.Group("/api/v1"), m.service)
}

func (m *Module) RegisterJobs(j contract.JobRegistry) {}

func (m *Module) RegisterEvents(e contract.EventBus) {}
`

const entityTemplate = `package domain

import (
	"time"

	"gorm.io/gorm"
)

type {{.NameCamel}} struct {
	ID        uint64         ` + "`gorm:\"primaryKey\" json:\"id\"`" + `
	CreatedAt time.Time      ` + "`json:\"created_at\"`" + `
	UpdatedAt time.Time      ` + "`json:\"updated_at\"`" + `
	DeletedAt gorm.DeletedAt ` + "`gorm:\"index\" json:\"-\"`" + `
}

func ({{.NameCamel}}) TableName() string {
	return "{{.Name}}s"
}
`

const repositoryTemplate = `package domain

import "context"

type {{.NameCamel}}Repository interface {
	FindByID(ctx context.Context, id uint64) (*{{.NameCamel}}, error)
	List(ctx context.Context, offset, limit int) ([]{{.NameCamel}}, int64, error)
	Create(ctx context.Context, entity *{{.NameCamel}}) error
	Update(ctx context.Context, entity *{{.NameCamel}}) error
	Delete(ctx context.Context, id uint64) error
}
`

const serviceTemplate = `package application

import (
	"context"

	"jimu/internal/modules/{{.Name}}/domain"
	"jimu/internal/shared/errors"
)

type {{.NameCamel}}Service struct {
	repo domain.{{.NameCamel}}Repository
}

func New{{.NameCamel}}Service(repo domain.{{.NameCamel}}Repository) *{{.NameCamel}}Service {
	return &{{.NameCamel}}Service{repo: repo}
}

func (s *{{.NameCamel}}Service) Get(ctx context.Context, id uint64) (*domain.{{.NameCamel}}, error) {
	entity, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, errors.New(errors.CodeNotFound, "{{.Name}} not found")
	}
	return entity, nil
}

func (s *{{.NameCamel}}Service) List(ctx context.Context, page, pageSize int) ([]domain.{{.NameCamel}}, int64, error) {
	offset := (page - 1) * pageSize
	return s.repo.List(ctx, offset, pageSize)
}

func (s *{{.NameCamel}}Service) Create(ctx context.Context, req Create{{.NameCamel}}Request) (*domain.{{.NameCamel}}, error) {
	// TODO: implement
	return nil, nil
}

func (s *{{.NameCamel}}Service) Update(ctx context.Context, id uint64, req Update{{.NameCamel}}Request) error {
	// TODO: implement
	return nil
}

func (s *{{.NameCamel}}Service) Delete(ctx context.Context, id uint64) error {
	return s.repo.Delete(ctx, id)
}
`

const dtoTemplate = `package application

type Create{{.NameCamel}}Request struct {
	// TODO: add fields
}

type Update{{.NameCamel}}Request struct {
	// TODO: add fields
}

type {{.NameCamel}}Response struct {
	ID uint64 ` + "`json:\"id\"`" + `
}
`

const mysqlRepoTemplate = `package infrastructure

import (
	"context"

	"jimu/internal/modules/{{.Name}}/domain"

	"gorm.io/gorm"
)

type mysql{{.NameCamel}}Repository struct {
	db *gorm.DB
}

func NewMysql{{.NameCamel}}Repository(db *gorm.DB) domain.{{.NameCamel}}Repository {
	return &mysql{{.NameCamel}}Repository{db: db}
}

func (r *mysql{{.NameCamel}}Repository) FindByID(ctx context.Context, id uint64) (*domain.{{.NameCamel}}, error) {
	var entity domain.{{.NameCamel}}
	err := r.db.WithContext(ctx).First(&entity, id).Error
	return &entity, err
}

func (r *mysql{{.NameCamel}}Repository) List(ctx context.Context, offset, limit int) ([]domain.{{.NameCamel}}, int64, error) {
	var items []domain.{{.NameCamel}}
	var total int64
	db := r.db.WithContext(ctx).Model(&domain.{{.NameCamel}}{})
	db.Count(&total)
	err := db.Offset(offset).Limit(limit).Find(&items).Error
	return items, total, err
}

func (r *mysql{{.NameCamel}}Repository) Create(ctx context.Context, entity *domain.{{.NameCamel}}) error {
	return r.db.WithContext(ctx).Create(entity).Error
}

func (r *mysql{{.NameCamel}}Repository) Update(ctx context.Context, entity *domain.{{.NameCamel}}) error {
	return r.db.WithContext(ctx).Save(entity).Error
}

func (r *mysql{{.NameCamel}}Repository) Delete(ctx context.Context, id uint64) error {
	return r.db.WithContext(ctx).Delete(&domain.{{.NameCamel}}{}, id).Error
}
`

const handlerTemplate = `package interfaces

import (
	"strconv"

	"jimu/internal/modules/{{.Name}}/application"
	"jimu/internal/shared/pagination"
	"jimu/internal/shared/response"

	"github.com/gin-gonic/gin"
)

type {{.NameCamel}}Handler struct {
	service *application.{{.NameCamel}}Service
}

func New{{.NameCamel}}Handler(service *application.{{.NameCamel}}Service) *{{.NameCamel}}Handler {
	return &{{.NameCamel}}Handler{service: service}
}

func (h *{{.NameCamel}}Handler) Create(c *gin.Context) {
	var req application.Create{{.NameCamel}}Request
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, gin.Error{})
		return
	}
	// TODO: implement
}

func (h *{{.NameCamel}}Handler) Get(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	entity, err := h.service.Get(c.Request.Context(), id)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, entity)
}

func (h *{{.NameCamel}}Handler) List(c *gin.Context) {
	var p pagination.Pagination
	if err := c.ShouldBindQuery(&p); err != nil {
		response.Fail(c, gin.Error{})
		return
	}
	items, total, err := h.service.List(c.Request.Context(), p.Page, p.PageSize)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.Page(c, items, total, p.Page, p.PageSize)
}

func (h *{{.NameCamel}}Handler) Update(c *gin.Context) {
	// TODO: implement
}

func (h *{{.NameCamel}}Handler) Delete(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	if err := h.service.Delete(c.Request.Context(), id); err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, nil)
}
`

const routerTemplate = `package interfaces

import (
	"jimu/internal/modules/{{.Name}}/application"

	"github.com/gin-gonic/gin"
)

func Register{{.NameCamel}}Routes(r *gin.RouterGroup, service *application.{{.NameCamel}}Service) {
	handler := New{{.NameCamel}}Handler(service)
	{{.Name}}s := r.Group("/{{.Name}}s")
	{
		{{.Name}}s.POST("", handler.Create)
		{{.Name}}s.GET("", handler.List)
		{{.Name}}s.GET("/:id", handler.Get)
		{{.Name}}s.PUT("/:id", handler.Update)
		{{.Name}}s.DELETE("/:id", handler.Delete)
	}
}
`
