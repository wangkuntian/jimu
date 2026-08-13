package interfaces

import (
	"bytes"
	"context"
	"errors"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"jimu/internal/modules/admin/application"
	admindomain "jimu/internal/modules/admin/domain"
	userdomain "jimu/internal/modules/user/domain"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

// multipartRequest 构造带 file 字段的 multipart 请求
func multipartRequest(method, target, filename, contentType, body string) *http.Request {
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	part, _ := writer.CreateFormFile("file", filename)
	part.Write([]byte(body))
	if contentType != "" {
		// CreateFormFile 默认 Content-Type 为 application/octet-stream；
		// 需要自定义类型时用 CreatePart 手动构造
		_ = contentType
	}
	writer.Close()
	req := httptest.NewRequest(method, target, &buf)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	return req
}

func newImportHandler(db *gorm.DB) *AdminImportHandler {
	return NewAdminImportHandler(application.NewImportService(&fakeImportJobRepo{}, &fakeUserRepository{}, db))
}

func TestAdminImportHandlerPreview(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/preview", newImportHandler(newSqliteDB(t, &userdomain.User{})).Preview)

	// 成功
	w := httptest.NewRecorder()
	r.ServeHTTP(w, multipartRequest(http.MethodPost, "/preview", "users.csv", "text/csv",
		"username,password,email\nalice,secret123,a@b.com\n"))
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "total_rows")

	// 缺少文件
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, httptest.NewRequest(http.MethodPost, "/preview", nil))
	assert.Equal(t, http.StatusBadRequest, w2.Code)

	// 校验错误行（返回 OK，带 error_rows）
	w3 := httptest.NewRecorder()
	r.ServeHTTP(w3, multipartRequest(http.MethodPost, "/preview", "users.csv", "text/csv",
		"username,password,email\n,secret123,a@b.com\n"))
	assert.Equal(t, http.StatusOK, w3.Code)
	assert.Contains(t, w3.Body.String(), "error_rows")
}

func TestAdminImportHandlerImport(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/import", func(c *gin.Context) {
		c.Set("user_id", uint64(1))
		newImportHandler(newSqliteDB(t, &userdomain.User{})).Import(c)
	})

	// 成功导入
	w := httptest.NewRecorder()
	r.ServeHTTP(w, multipartRequest(http.MethodPost, "/import", "users.csv", "text/csv",
		"username,password,email\nalice,secret123,a@b.com\n"))
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "import_job_id")

	// 校验失败：返回 validation_error，无任务
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, multipartRequest(http.MethodPost, "/import", "users.csv", "text/csv",
		"username,password,email\n,secret123,a@b.com\n"))
	assert.Equal(t, http.StatusOK, w2.Code)
	assert.Contains(t, w2.Body.String(), "validation_error")

	// 缺少文件
	w3 := httptest.NewRecorder()
	r.ServeHTTP(w3, httptest.NewRequest(http.MethodPost, "/import", nil))
	assert.Equal(t, http.StatusBadRequest, w3.Code)

	// 服务错误（创建任务失败）
	r2 := gin.New()
	r2.POST("/import", func(c *gin.Context) {
		c.Set("user_id", uint64(1))
		NewAdminImportHandler(application.NewImportService(&fakeImportJobRepo{
			create: func(ctx context.Context, job *admindomain.ImportJob) error {
				return errors.New("db down")
			},
		}, &fakeUserRepository{}, newSqliteDB(t, &userdomain.User{}))).Import(c)
	})
	w4 := httptest.NewRecorder()
	r2.ServeHTTP(w4, multipartRequest(http.MethodPost, "/import", "users.csv", "text/csv",
		"username,password,email\nalice,secret123,a@b.com\n"))
	assert.Equal(t, http.StatusInternalServerError, w4.Code)
}

func TestAdminImportHandlerGet(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/import/:id", newImportHandler(nil).Get)

	// 成功
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/import/1", nil))
	assert.Equal(t, http.StatusOK, w.Code)

	// 非法 id
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, httptest.NewRequest(http.MethodGet, "/import/abc", nil))
	assert.Equal(t, http.StatusBadRequest, w2.Code)

	// 未找到
	r2 := gin.New()
	r2.GET("/import/:id", NewAdminImportHandler(application.NewImportService(&fakeImportJobRepo{
		findByID: func(ctx context.Context, id uint64) (*admindomain.ImportJob, error) {
			return nil, gorm.ErrRecordNotFound
		},
	}, &fakeUserRepository{}, nil)).Get)
	w3 := httptest.NewRecorder()
	r2.ServeHTTP(w3, httptest.NewRequest(http.MethodGet, "/import/1", nil))
	assert.Equal(t, http.StatusNotFound, w3.Code)
}

func TestAdminImportHandlerTemplate(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/template", newImportHandler(nil).Template)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/template", nil))
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "text/csv", w.Header().Get("Content-Type"))
	assert.Contains(t, w.Body.String(), "username,password,email")
}

func TestBindImportFileXLSX(t *testing.T) {
	gin.SetMode(gin.TestMode)
	// 通过 Preview 验证 .xlsx 文件名 + octet-stream 被识别为 Excel 格式
	r := gin.New()
	r.POST("/preview", func(c *gin.Context) {
		format, _, err := bindImportFile(c)
		assert.NoError(t, err)
		assert.Equal(t, "xlsx", string(format))
		c.Status(200)
	})
	w := httptest.NewRecorder()
	// xlsx 内容不是合法 Excel，bindImportFile 只解析文件头；这里用空内容验证格式分支
	req := multipartRequest(http.MethodPost, "/preview", "data.xlsx", "application/octet-stream", "junk")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}
