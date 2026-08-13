package interfaces

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	qdomain "jimu/internal/platform/queue/domain"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestAdminJobHandlerSubmit(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/jobs", NewAdminJobHandler(&fakeJobRepo{}, &fakeDeadLetterRepo{}).Submit)

	// 成功（默认优先级与最大尝试次数）
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/jobs",
		strings.NewReader(`{"type":"email","payload":"x"}`)))
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "job_id")

	// 成功（自定义优先级）
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, httptest.NewRequest(http.MethodPost, "/jobs",
		strings.NewReader(`{"type":"email","priority":1,"max_attempts":5}`)))
	assert.Equal(t, http.StatusOK, w2.Code)

	// 非法 JSON（handler 透传绑定错误 -> 500）
	w3 := httptest.NewRecorder()
	r.ServeHTTP(w3, httptest.NewRequest(http.MethodPost, "/jobs", strings.NewReader(`{bad`)))
	assert.Equal(t, http.StatusInternalServerError, w3.Code)

	// 仓储错误
	r2 := gin.New()
	r2.POST("/jobs", NewAdminJobHandler(&fakeJobRepo{create: func(ctx context.Context, job *qdomain.Job) error {
		return errors.New("db down")
	}}, &fakeDeadLetterRepo{}).Submit)
	w4 := httptest.NewRecorder()
	r2.ServeHTTP(w4, httptest.NewRequest(http.MethodPost, "/jobs",
		strings.NewReader(`{"type":"email"}`)))
	assert.Equal(t, http.StatusInternalServerError, w4.Code)
}

func TestAdminJobHandlerList(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/jobs", NewAdminJobHandler(&fakeJobRepo{}, &fakeDeadLetterRepo{}).List)

	// 无过滤
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/jobs", nil))
	assert.Equal(t, http.StatusOK, w.Code)

	// 带 status 过滤
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, httptest.NewRequest(http.MethodGet, "/jobs?status=pending", nil))
	assert.Equal(t, http.StatusOK, w2.Code)

	// 仓储错误
	r2 := gin.New()
	r2.GET("/jobs", NewAdminJobHandler(&fakeJobRepo{list: func(ctx context.Context, offset, limit int, filters map[string]interface{}) ([]qdomain.Job, int64, error) {
		return nil, 0, errors.New("db down")
	}}, &fakeDeadLetterRepo{}).List)
	w3 := httptest.NewRecorder()
	r2.ServeHTTP(w3, httptest.NewRequest(http.MethodGet, "/jobs", nil))
	assert.Equal(t, http.StatusInternalServerError, w3.Code)
}

func TestAdminJobHandlerGet(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/jobs/:id", NewAdminJobHandler(&fakeJobRepo{}, &fakeDeadLetterRepo{}).Get)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/jobs/1", nil))
	assert.Equal(t, http.StatusOK, w.Code)

	// 非法 id
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, httptest.NewRequest(http.MethodGet, "/jobs/abc", nil))
	assert.Equal(t, http.StatusBadRequest, w2.Code)

	// 未找到
	r2 := gin.New()
	r2.GET("/jobs/:id", NewAdminJobHandler(&fakeJobRepo{findByID: func(ctx context.Context, id uint64) (*qdomain.Job, error) {
		return nil, errors.New("not found")
	}}, &fakeDeadLetterRepo{}).Get)
	w3 := httptest.NewRecorder()
	r2.ServeHTTP(w3, httptest.NewRequest(http.MethodGet, "/jobs/1", nil))
	assert.Equal(t, http.StatusNotFound, w3.Code)
}

func TestAdminJobHandlerRetry(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/jobs/:id/retry", NewAdminJobHandler(&fakeJobRepo{}, &fakeDeadLetterRepo{}).Retry)

	// 成功
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/jobs/1/retry", nil))
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "retried")

	// 非法 id
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, httptest.NewRequest(http.MethodPost, "/jobs/abc/retry", nil))
	assert.Equal(t, http.StatusBadRequest, w2.Code)

	// 未找到
	r2 := gin.New()
	r2.POST("/jobs/:id/retry", NewAdminJobHandler(&fakeJobRepo{findByID: func(ctx context.Context, id uint64) (*qdomain.Job, error) {
		return nil, errors.New("not found")
	}}, &fakeDeadLetterRepo{}).Retry)
	w3 := httptest.NewRecorder()
	r2.ServeHTTP(w3, httptest.NewRequest(http.MethodPost, "/jobs/1/retry", nil))
	assert.Equal(t, http.StatusNotFound, w3.Code)

	// 更新失败
	r3 := gin.New()
	r3.POST("/jobs/:id/retry", NewAdminJobHandler(&fakeJobRepo{update: func(ctx context.Context, job *qdomain.Job) error {
		return errors.New("db down")
	}}, &fakeDeadLetterRepo{}).Retry)
	w4 := httptest.NewRecorder()
	r3.ServeHTTP(w4, httptest.NewRequest(http.MethodPost, "/jobs/1/retry", nil))
	assert.Equal(t, http.StatusInternalServerError, w4.Code)
}

func TestAdminJobHandlerListDeadLetters(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// 死信仓储未配置
	r := gin.New()
	r.GET("/jobs/dead-letters", NewAdminJobHandler(&fakeJobRepo{}, nil).ListDeadLetters)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/jobs/dead-letters", nil))
	assert.Equal(t, http.StatusInternalServerError, w.Code)

	// 成功
	r2 := gin.New()
	r2.GET("/jobs/dead-letters", NewAdminJobHandler(&fakeJobRepo{}, &fakeDeadLetterRepo{}).ListDeadLetters)
	w2 := httptest.NewRecorder()
	r2.ServeHTTP(w2, httptest.NewRequest(http.MethodGet, "/jobs/dead-letters?resolved=true", nil))
	assert.Equal(t, http.StatusOK, w2.Code)

	// 仓储错误
	r3 := gin.New()
	r3.GET("/jobs/dead-letters", NewAdminJobHandler(&fakeJobRepo{}, &fakeDeadLetterRepo{list: func(ctx context.Context, offset, limit int, resolved bool) ([]qdomain.DeadLetter, int64, error) {
		return nil, 0, errors.New("db down")
	}}).ListDeadLetters)
	w3 := httptest.NewRecorder()
	r3.ServeHTTP(w3, httptest.NewRequest(http.MethodGet, "/jobs/dead-letters", nil))
	assert.Equal(t, http.StatusInternalServerError, w3.Code)
}

func TestAdminJobHandlerResolveDeadLetter(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// 死信仓储未配置
	r := gin.New()
	r.POST("/jobs/dead-letters/:id/resolve", NewAdminJobHandler(&fakeJobRepo{}, nil).ResolveDeadLetter)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/jobs/dead-letters/1/resolve", nil))
	assert.Equal(t, http.StatusInternalServerError, w.Code)

	// 非法 id
	r2 := gin.New()
	r2.POST("/jobs/dead-letters/:id/resolve", NewAdminJobHandler(&fakeJobRepo{}, &fakeDeadLetterRepo{}).ResolveDeadLetter)
	w2 := httptest.NewRecorder()
	r2.ServeHTTP(w2, httptest.NewRequest(http.MethodPost, "/jobs/dead-letters/abc/resolve", nil))
	assert.Equal(t, http.StatusBadRequest, w2.Code)

	// 成功
	w3 := httptest.NewRecorder()
	r2.ServeHTTP(w3, httptest.NewRequest(http.MethodPost, "/jobs/dead-letters/1/resolve", nil))
	assert.Equal(t, http.StatusOK, w3.Code)

	// 仓储错误
	r3 := gin.New()
	r3.POST("/jobs/dead-letters/:id/resolve", NewAdminJobHandler(&fakeJobRepo{}, &fakeDeadLetterRepo{markResolved: func(ctx context.Context, id uint64) error {
		return errors.New("db down")
	}}).ResolveDeadLetter)
	w4 := httptest.NewRecorder()
	r3.ServeHTTP(w4, httptest.NewRequest(http.MethodPost, "/jobs/dead-letters/1/resolve", nil))
	assert.Equal(t, http.StatusInternalServerError, w4.Code)
}
