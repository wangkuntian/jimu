package generator

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestGeneratedModuleCompiles(t *testing.T) {
	root := newTestRepository(t)
	copyRootFile(t, root, "go.mod")
	copyGoSum(t, root)
	writeStubPackages(t, root)

	if err := GenerateModuleAt(root, "product"); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command("go", "test", "./internal/modules/product/...")
	cmd.Dir = root
	cmd.Env = append(os.Environ(), "GOWORK=off", "GOCACHE="+filepath.Join(os.TempDir(), "jimu-go-build-cache"))
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("generated module does not compile: %v\n%s", err, output)
	}
}

func writeStubPackages(t *testing.T, root string) {
	t.Helper()
	writeFileForTest(t, root, "internal/contract/contract.go", `package contract

import "github.com/gin-gonic/gin"

type Router interface {
	Group(string, ...gin.HandlerFunc) *gin.RouterGroup
}

type JobRegistry interface{}
type EventBus interface{}
`)
	writeFileForTest(t, root, "internal/shared/errors/errors.go", `package errors

import "fmt"

const (
	CodeInvalidParam = 1001
	CodeNotFound = 1004
	CodeInternalError = 1005
	CodeConflict = 1009
)

type AppError struct {
	Code int
	Message string
	Cause error
}

func (e *AppError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("[%d] %s: %v", e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("[%d] %s", e.Code, e.Message)
}

func (e *AppError) Unwrap() error { return e.Cause }
func New(code int, message string) *AppError { return &AppError{Code: code, Message: message} }
func Wrap(code int, message string, cause error) *AppError {
	return &AppError{Code: code, Message: message, Cause: cause}
}
`)
	writeFileForTest(t, root, "internal/shared/pagination/pagination.go", `package pagination

import "strings"

type Pagination struct {
	Page int `+"`form:\"page\"`"+`
	PageSize int `+"`form:\"page_size\"`"+`
	Sort string `+"`form:\"sort\"`"+`
	Order string `+"`form:\"order\"`"+`
}

func (p *Pagination) Normalize(allowedSorts ...string) error {
	if p.Page == 0 { p.Page = 1 }
	if p.PageSize == 0 { p.PageSize = 20 }
	if p.Sort == "" { p.Sort = "id" }
	p.Order = strings.ToLower(strings.TrimSpace(p.Order))
	if p.Order == "" { p.Order = "desc" }
	return nil
}

func (p Pagination) GetOffset() int { return (p.Page - 1) * p.PageSize }
func (p Pagination) GetLimit() int { return p.PageSize }
`)
	writeFileForTest(t, root, "internal/shared/response/response.go", `package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type Body struct {
	Code int `+"`json:\"code\"`"+`
	Message string `+"`json:\"message\"`"+`
	Data interface{} `+"`json:\"data,omitempty\"`"+`
}

func OK(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Body{Code: 0, Message: "ok", Data: data})
}

func Created(c *gin.Context, data interface{}) {
	c.JSON(http.StatusCreated, Body{Code: 0, Message: "ok", Data: data})
}

func NoContent(c *gin.Context) { c.Status(http.StatusNoContent) }
func Fail(c *gin.Context, err error) {
	c.JSON(http.StatusInternalServerError, Body{Code: 1005, Message: err.Error()})
}
func Page(c *gin.Context, data interface{}, total int64, page, pageSize int) {
	c.JSON(http.StatusOK, data)
}
`)
	writeFileForTest(t, root, "internal/shared/errors/errors_test.go", `package errors
`)
	writeFileForTest(t, root, "internal/platform/http/middleware/middleware.go", `package middleware

import (
	stderrors "errors"
	"fmt"
	"strings"

	"jimu/internal/shared/errors"
	"jimu/internal/shared/response"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

func ValidateJSON(dst interface{}) gin.HandlerFunc {
	return func(c *gin.Context) {
		if err := c.ShouldBindJSON(dst); err != nil {
			response.Fail(c, errors.New(errors.CodeInvalidParam, translateValidationError(err)))
			c.Abort()
			return
		}
		c.Set("validated_req", dst)
		c.Next()
	}
}

func ValidateQuery(dst interface{}) gin.HandlerFunc {
	return func(c *gin.Context) {
		if err := c.ShouldBindQuery(dst); err != nil {
			response.Fail(c, errors.New(errors.CodeInvalidParam, translateValidationError(err)))
			c.Abort()
			return
		}
		c.Set("validated_query", dst)
		c.Next()
	}
}

func translateValidationError(err error) string {
	var verr validator.ValidationErrors
	if stderrors.As(err, &verr) && len(verr) > 0 {
		e := verr[0]
		field := e.Field()
		switch e.Tag() {
		case "required":
			return fmt.Sprintf("%s 不能为空", field)
		default:
			return fmt.Sprintf("%s 校验失败", field)
		}
	}
	if strings.Contains(err.Error(), "invalid") {
		return "请求体格式错误"
	}
	return "请求参数错误"
}
`)
}

func writeFileForTest(t *testing.T, root, rel, content string) {
	t.Helper()
	path := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func copyGoSum(t *testing.T, root string) {
	copyRootFile(t, root, "go.sum")
}

func copyRootFile(t *testing.T, root, name string) {
	t.Helper()
	content, err := os.ReadFile(filepath.Join("..", "..", name))
	if err != nil {
		t.Fatal(err)
	}
	writeFileForTest(t, root, name, string(content))
}
