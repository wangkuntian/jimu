package interfaces

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"jimu/internal/modules/admin/application"
	admindomain "jimu/internal/modules/admin/domain"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestAdminAPIKeyHandlerList(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/apikeys", NewAdminAPIKeyHandler(application.NewAdminAPIKeyService(&fakeAPIKeyRepo{})).List)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/apikeys", nil))
	assert.Equal(t, http.StatusOK, w.Code)

	r2 := gin.New()
	r2.GET("/apikeys", NewAdminAPIKeyHandler(application.NewAdminAPIKeyService(&fakeAPIKeyRepo{list: func(ctx context.Context, offset, limit int) ([]admindomain.APIKey, int64, error) {
		return nil, 0, errors.New("db down")
	}})).List)
	w2 := httptest.NewRecorder()
	r2.ServeHTTP(w2, httptest.NewRequest(http.MethodGet, "/apikeys", nil))
	assert.Equal(t, http.StatusInternalServerError, w2.Code)
}

func TestAdminAPIKeyHandlerCreate(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/apikeys", func(c *gin.Context) {
		c.Set("user_id", uint64(3))
		NewAdminAPIKeyHandler(application.NewAdminAPIKeyService(&fakeAPIKeyRepo{})).Create(c)
	})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/apikeys",
		strings.NewReader(`{"name":"web","scopes":["read"],"expires_in":30}`)))
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "store this key safely")

	// 非法 JSON（handler 透传绑定错误 -> 500）
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, httptest.NewRequest(http.MethodPost, "/apikeys", strings.NewReader(`{bad`)))
	assert.Equal(t, http.StatusInternalServerError, w2.Code)

	// 仓储错误
	r2 := gin.New()
	r2.POST("/apikeys", func(c *gin.Context) {
		c.Set("user_id", uint64(3))
		NewAdminAPIKeyHandler(application.NewAdminAPIKeyService(&fakeAPIKeyRepo{create: func(ctx context.Context, key *admindomain.APIKey) error {
			return errors.New("db down")
		}})).Create(c)
	})
	w3 := httptest.NewRecorder()
	r2.ServeHTTP(w3, httptest.NewRequest(http.MethodPost, "/apikeys",
		strings.NewReader(`{"name":"web"}`)))
	assert.Equal(t, http.StatusInternalServerError, w3.Code)
}

func TestAdminAPIKeyHandlerGet(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/apikeys/:id", NewAdminAPIKeyHandler(application.NewAdminAPIKeyService(&fakeAPIKeyRepo{})).Get)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/apikeys/1", nil))
	assert.Equal(t, http.StatusOK, w.Code)

	// 非法 id
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, httptest.NewRequest(http.MethodGet, "/apikeys/abc", nil))
	assert.Equal(t, http.StatusBadRequest, w2.Code)

	// 未找到
	r2 := gin.New()
	r2.GET("/apikeys/:id", NewAdminAPIKeyHandler(application.NewAdminAPIKeyService(&fakeAPIKeyRepo{findByID: func(ctx context.Context, id uint64) (*admindomain.APIKey, error) {
		return nil, errors.New("not found")
	}})).Get)
	w3 := httptest.NewRecorder()
	r2.ServeHTTP(w3, httptest.NewRequest(http.MethodGet, "/apikeys/1", nil))
	assert.Equal(t, http.StatusInternalServerError, w3.Code)
}

func TestAdminAPIKeyHandlerRevoke(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.DELETE("/apikeys/:id", NewAdminAPIKeyHandler(application.NewAdminAPIKeyService(&fakeAPIKeyRepo{})).Revoke)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodDelete, "/apikeys/1", nil))
	assert.Equal(t, http.StatusOK, w.Code)

	// 非法 id
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, httptest.NewRequest(http.MethodDelete, "/apikeys/abc", nil))
	assert.Equal(t, http.StatusBadRequest, w2.Code)

	// 仓储错误
	r2 := gin.New()
	r2.DELETE("/apikeys/:id", NewAdminAPIKeyHandler(application.NewAdminAPIKeyService(&fakeAPIKeyRepo{delete: func(ctx context.Context, id uint64) error {
		return errors.New("db down")
	}})).Revoke)
	w3 := httptest.NewRecorder()
	r2.ServeHTTP(w3, httptest.NewRequest(http.MethodDelete, "/apikeys/1", nil))
	assert.Equal(t, http.StatusInternalServerError, w3.Code)
}
