package generator

const moduleTemplate = `package {{.Name}}

import (
	"jimu/internal/contract"
	"jimu/internal/modules/{{.Name}}/application"
	"jimu/internal/modules/{{.Name}}/infrastructure"
	"jimu/internal/modules/{{.Name}}/interfaces"

	"gorm.io/gorm"
)

type Module struct {
	service *application.{{.NameCamel}}Service
}

func New(db *gorm.DB) *Module {
	repo := infrastructure.NewMysql{{.NameCamel}}Repository(db)
	service := application.New{{.NameCamel}}Service(repo)
	return &Module{service: service}
}

func (m *Module) Name() string { return "{{.Name}}" }

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
	ID          uint64         ` + "`gorm:\"primaryKey\" json:\"id\"`" + `
	Name        string         ` + "`gorm:\"size:128;uniqueIndex;not null\" json:\"name\"`" + `
	Description string         ` + "`gorm:\"size:255;default:''\" json:\"description\"`" + `
	CreatedAt   time.Time      ` + "`json:\"created_at\"`" + `
	UpdatedAt   time.Time      ` + "`json:\"updated_at\"`" + `
	DeletedAt   gorm.DeletedAt ` + "`gorm:\"index\" json:\"-\"`" + `
}

func ({{.NameCamel}}) TableName() string {
	return "{{.TableName}}"
}
`

const repositoryTemplate = `package domain

import "context"

type {{.NameCamel}}Repository interface {
	FindByID(ctx context.Context, id uint64) (*{{.NameCamel}}, error)
	List(ctx context.Context, offset, limit int, sort, order string) ([]{{.NameCamel}}, int64, error)
	Create(ctx context.Context, entity *{{.NameCamel}}) error
	Update(ctx context.Context, entity *{{.NameCamel}}) error
	Delete(ctx context.Context, id uint64) error
}
`

const dtoTemplate = `package application

import (
	"time"

	"jimu/internal/modules/{{.Name}}/domain"
)

type Create{{.NameCamel}}Request struct {
	Name        string ` + "`json:\"name\" binding:\"required,max=128\"`" + `
	Description string ` + "`json:\"description\" binding:\"max=255\"`" + `
}

type Update{{.NameCamel}}Request struct {
	Name        string ` + "`json:\"name\" binding:\"required,max=128\"`" + `
	Description string ` + "`json:\"description\" binding:\"max=255\"`" + `
}

type {{.NameCamel}}Response struct {
	ID          uint64    ` + "`json:\"id\"`" + `
	Name        string    ` + "`json:\"name\"`" + `
	Description string    ` + "`json:\"description\"`" + `
	CreatedAt   time.Time ` + "`json:\"created_at\"`" + `
	UpdatedAt   time.Time ` + "`json:\"updated_at\"`" + `
}

func To{{.NameCamel}}Response(entity domain.{{.NameCamel}}) {{.NameCamel}}Response {
	return {{.NameCamel}}Response{
		ID:          entity.ID,
		Name:        entity.Name,
		Description: entity.Description,
		CreatedAt:   entity.CreatedAt,
		UpdatedAt:   entity.UpdatedAt,
	}
}

func To{{.NameCamel}}Responses(entities []domain.{{.NameCamel}}) []{{.NameCamel}}Response {
	out := make([]{{.NameCamel}}Response, 0, len(entities))
	for _, entity := range entities {
		out = append(out, To{{.NameCamel}}Response(entity))
	}
	return out
}
`

const errorsTemplate = `package application

import (
	stderrors "errors"

	mysqlerr "github.com/go-sql-driver/mysql"
	"gorm.io/gorm"
)

func isDuplicateKey(err error) bool {
	if stderrors.Is(err, gorm.ErrDuplicatedKey) {
		return true
	}
	var mysqlErr *mysqlerr.MySQLError
	return stderrors.As(err, &mysqlErr) && mysqlErr.Number == 1062
}
`

const serviceTemplate = `package application

import (
	"context"
	stderrors "errors"

	"jimu/internal/modules/{{.Name}}/domain"
	"jimu/internal/shared/errors"
	"jimu/internal/shared/pagination"

	"gorm.io/gorm"
)

type {{.NameCamel}}Service struct {
	repo domain.{{.NameCamel}}Repository
}

func New{{.NameCamel}}Service(repo domain.{{.NameCamel}}Repository) *{{.NameCamel}}Service {
	return &{{.NameCamel}}Service{repo: repo}
}

func (s *{{.NameCamel}}Service) Create(ctx context.Context, req Create{{.NameCamel}}Request) (*{{.NameCamel}}Response, error) {
	entity := &domain.{{.NameCamel}}{Name: req.Name, Description: req.Description}
	if err := s.repo.Create(ctx, entity); err != nil {
		if isDuplicateKey(err) {
			return nil, errors.Wrap(errors.CodeConflict, "{{.Name}} already exists", err)
		}
		return nil, errors.Wrap(errors.CodeInternalError, "failed to create {{.Name}}", err)
	}
	resp := To{{.NameCamel}}Response(*entity)
	return &resp, nil
}

func (s *{{.NameCamel}}Service) Get(ctx context.Context, id uint64) (*{{.NameCamel}}Response, error) {
	entity, err := s.repo.FindByID(ctx, id)
	if err != nil {
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.Wrap(errors.CodeNotFound, "{{.Name}} not found", err)
		}
		return nil, errors.Wrap(errors.CodeInternalError, "failed to get {{.Name}}", err)
	}
	resp := To{{.NameCamel}}Response(*entity)
	return &resp, nil
}

func (s *{{.NameCamel}}Service) List(ctx context.Context, p pagination.Pagination) ([]{{.NameCamel}}Response, int64, error) {
	entities, total, err := s.repo.List(ctx, p.GetOffset(), p.GetLimit(), p.Sort, p.Order)
	if err != nil {
		return nil, 0, errors.Wrap(errors.CodeInternalError, "failed to list {{.Name}}", err)
	}
	return To{{.NameCamel}}Responses(entities), total, nil
}

func (s *{{.NameCamel}}Service) Update(ctx context.Context, id uint64, req Update{{.NameCamel}}Request) error {
	entity, err := s.repo.FindByID(ctx, id)
	if err != nil {
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			return errors.Wrap(errors.CodeNotFound, "{{.Name}} not found", err)
		}
		return errors.Wrap(errors.CodeInternalError, "failed to get {{.Name}}", err)
	}
	entity.Name = req.Name
	entity.Description = req.Description
	if err := s.repo.Update(ctx, entity); err != nil {
		if isDuplicateKey(err) {
			return errors.Wrap(errors.CodeConflict, "{{.Name}} already exists", err)
		}
		return errors.Wrap(errors.CodeInternalError, "failed to update {{.Name}}", err)
	}
	return nil
}

func (s *{{.NameCamel}}Service) Delete(ctx context.Context, id uint64) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		return errors.Wrap(errors.CodeInternalError, "failed to delete {{.Name}}", err)
	}
	return nil
}
`

const mysqlRepoTemplate = `package infrastructure

import (
	"context"

	"jimu/internal/modules/{{.Name}}/domain"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
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
	if err != nil {
		return nil, err
	}
	return &entity, nil
}

func (r *mysql{{.NameCamel}}Repository) List(ctx context.Context, offset, limit int, sort, order string) ([]domain.{{.NameCamel}}, int64, error) {
	var items []domain.{{.NameCamel}}
	var total int64
	db := r.db.WithContext(ctx).Model(&domain.{{.NameCamel}}{})
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	err := db.Order(clause.OrderByColumn{
		Column: clause.Column{Name: sort},
		Desc:   order == "desc",
	}).Offset(offset).Limit(limit).Find(&items).Error
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
	"jimu/internal/shared/errors"
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
	req := c.MustGet("validated_req").(*application.Create{{.NameCamel}}Request)
	entity, err := h.service.Create(c.Request.Context(), *req)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.Created(c, entity)
}

func (h *{{.NameCamel}}Handler) Get(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.Fail(c, errors.New(errors.CodeInvalidParam, "invalid id"))
		return
	}
	entity, err := h.service.Get(c.Request.Context(), id)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, entity)
}

func (h *{{.NameCamel}}Handler) List(c *gin.Context) {
	p := c.MustGet("validated_query").(*pagination.Pagination)
	if err := p.Normalize("id", "name", "created_at"); err != nil {
		response.Fail(c, errors.New(errors.CodeInvalidParam, err.Error()))
		return
	}
	items, total, err := h.service.List(c.Request.Context(), *p)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.Page(c, items, total, p.Page, p.PageSize)
}

func (h *{{.NameCamel}}Handler) Update(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.Fail(c, errors.New(errors.CodeInvalidParam, "invalid id"))
		return
	}
	req := c.MustGet("validated_req").(*application.Update{{.NameCamel}}Request)
	if err := h.service.Update(c.Request.Context(), id, *req); err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, nil)
}

func (h *{{.NameCamel}}Handler) Delete(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.Fail(c, errors.New(errors.CodeInvalidParam, "invalid id"))
		return
	}
	if err := h.service.Delete(c.Request.Context(), id); err != nil {
		response.Fail(c, err)
		return
	}
	response.NoContent(c)
}
`

const routerTemplate = `package interfaces

import (
	"jimu/internal/modules/{{.Name}}/application"
	"jimu/internal/platform/http/middleware"
	"jimu/internal/shared/pagination"

	"github.com/gin-gonic/gin"
)

func Register{{.NameCamel}}Routes(r *gin.RouterGroup, service *application.{{.NameCamel}}Service) {
	handler := New{{.NameCamel}}Handler(service)
	group := r.Group("/{{.RouteName}}")
	{
		group.POST("", middleware.ValidateJSON(&application.Create{{.NameCamel}}Request{}), handler.Create)
		group.GET("", middleware.ValidateQuery(&pagination.Pagination{}), handler.List)
		group.GET("/:id", handler.Get)
		group.PUT("/:id", middleware.ValidateJSON(&application.Update{{.NameCamel}}Request{}), handler.Update)
		group.DELETE("/:id", handler.Delete)
	}
}
`

const serviceTestTemplate = `package application

import (
	"context"
	stderrors "errors"
	"testing"

	"jimu/internal/modules/{{.Name}}/domain"
	apperrors "jimu/internal/shared/errors"
	"jimu/internal/shared/pagination"

	"gorm.io/gorm"
)

func Test{{.NameCamel}}ServiceListPassesPagination(t *testing.T) {
	seed := []domain.{{.NameCamel}}{
		{ID: 1, Name: "one"},
	}
	repo := &fake{{.NameCamel}}Repository{items: seed, total: 3}
	service := New{{.NameCamel}}Service(repo)

	items, total, err := service.List(context.Background(), pagination.Pagination{Page: 2, PageSize: 5, Sort: "created_at", Order: "asc"})
	if err != nil {
		t.Fatal(err)
	}
	if repo.offset != 5 || repo.limit != 5 || repo.sort != "created_at" || repo.order != "asc" {
		t.Fatalf("pagination = offset:%d limit:%d sort:%q order:%q", repo.offset, repo.limit, repo.sort, repo.order)
	}
	if total != 3 || len(items) != 1 || items[0].Name != "one" {
		t.Fatalf("items = %#v total = %d", items, total)
	}
}

func Test{{.NameCamel}}ServiceGetMapsNotFound(t *testing.T) {
	service := New{{.NameCamel}}Service(&fake{{.NameCamel}}Repository{findErr: gorm.ErrRecordNotFound})

	_, err := service.Get(context.Background(), 9)
	if appCode(err) != apperrors.CodeNotFound {
		t.Fatalf("code = %d, want %d", appCode(err), apperrors.CodeNotFound)
	}
}

type fake{{.NameCamel}}Repository struct {
	item      *domain.{{.NameCamel}}
	items     []domain.{{.NameCamel}}
	total     int64
	findErr   error
	createErr error
	updateErr error
	deleteErr error
	offset    int
	limit     int
	sort      string
	order     string
}

func (r *fake{{.NameCamel}}Repository) FindByID(context.Context, uint64) (*domain.{{.NameCamel}}, error) {
	if r.item != nil {
		return r.item, r.findErr
	}
	return &domain.{{.NameCamel}}{}, r.findErr
}

func (r *fake{{.NameCamel}}Repository) List(_ context.Context, offset, limit int, sort, order string) ([]domain.{{.NameCamel}}, int64, error) {
	r.offset = offset
	r.limit = limit
	r.sort = sort
	r.order = order
	return r.items, r.total, nil
}

func (r *fake{{.NameCamel}}Repository) Create(context.Context, *domain.{{.NameCamel}}) error {
	return r.createErr
}
func (r *fake{{.NameCamel}}Repository) Update(context.Context, *domain.{{.NameCamel}}) error {
	return r.updateErr
}
func (r *fake{{.NameCamel}}Repository) Delete(context.Context, uint64) error {
	return r.deleteErr
}

func appCode(err error) int {
	var appErr *apperrors.AppError
	if stderrors.As(err, &appErr) {
		return appErr.Code
	}
	return 0
}
`

const handlerTestTemplate = `package interfaces

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"jimu/internal/modules/{{.Name}}/application"
	"jimu/internal/modules/{{.Name}}/domain"
	"jimu/internal/platform/http/middleware"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func Test{{.NameCamel}}CreateReturnsCreated(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	handler := New{{.NameCamel}}Handler(application.New{{.NameCamel}}Service(&fake{{.NameCamel}}Repository{}))
	r.POST("/{{.RouteName}}", middleware.ValidateJSON(&application.Create{{.NameCamel}}Request{}), handler.Create)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/{{.RouteName}}", strings.NewReader(` + "`" + `{"name":"one","description":"desc"}` + "`" + `)))

	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusCreated)
	}
}

func Test{{.NameCamel}}DeleteReturnsNoContent(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	handler := New{{.NameCamel}}Handler(application.New{{.NameCamel}}Service(&fake{{.NameCamel}}Repository{}))
	r.DELETE("/{{.RouteName}}/:id", handler.Delete)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodDelete, "/{{.RouteName}}/7", nil))

	if w.Code != http.StatusNoContent || w.Body.Len() != 0 {
		t.Fatalf("status = %d body = %q", w.Code, w.Body.String())
	}
}

type fake{{.NameCamel}}Repository struct{}

func (r *fake{{.NameCamel}}Repository) FindByID(context.Context, uint64) (*domain.{{.NameCamel}}, error) {
	return &domain.{{.NameCamel}}{ID: 7, Name: "one"}, nil
}
func (r *fake{{.NameCamel}}Repository) List(context.Context, int, int, string, string) ([]domain.{{.NameCamel}}, int64, error) {
	return nil, 0, nil
}
func (r *fake{{.NameCamel}}Repository) Create(_ context.Context, entity *domain.{{.NameCamel}}) error {
	entity.ID = 7
	return nil
}
func (r *fake{{.NameCamel}}Repository) Update(context.Context, *domain.{{.NameCamel}}) error {
	return nil
}
func (r *fake{{.NameCamel}}Repository) Delete(context.Context, uint64) error {
	return nil
}

var _ = gorm.ErrRecordNotFound
`

const migrationTemplate = `-- +goose Up
CREATE TABLE IF NOT EXISTS {{.TableName}} (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    name VARCHAR(128) NOT NULL,
    description VARCHAR(255) DEFAULT '',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP NULL DEFAULT NULL,
    PRIMARY KEY (id),
    UNIQUE KEY uk_{{.TableName}}_name (name),
    INDEX idx_{{.TableName}}_deleted_at (deleted_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- +goose Down
DROP TABLE IF EXISTS {{.TableName}};
`
