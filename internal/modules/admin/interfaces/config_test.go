package interfaces

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"jimu/internal/modules/admin/application"

	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
)

func newConfigHandler(t *testing.T) (*AdminConfigHandler, *miniredis.Miniredis, *redis.Client) {
	t.Helper()
	mr, err := miniredis.Run()
	assert.NoError(t, err)
	t.Cleanup(mr.Close)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	svc := application.NewAdminConfigService(client, &fakeEventBus{}, "jimu")
	return NewAdminConfigHandler(svc), mr, client
}

func TestAdminConfigHandlerGet(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h, mr, _ := newConfigHandler(t)
	assert.NoError(t, mr.Set("jimu:config:log_level", "debug"))

	r := gin.New()
	r.GET("/config", h.Get)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/config", nil))
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "log_level")

	// 服务错误
	r2 := gin.New()
	svc := application.NewAdminConfigService(redis.NewClient(&redis.Options{Addr: mr.Addr()}), &fakeEventBus{}, "jimu")
	r2.GET("/config", NewAdminConfigHandler(svc).Get)
	mr.Close()
	w2 := httptest.NewRecorder()
	r2.ServeHTTP(w2, httptest.NewRequest(http.MethodGet, "/config", nil))
	assert.Equal(t, http.StatusInternalServerError, w2.Code)
}

func TestAdminConfigHandlerUpdate(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h, mr, _ := newConfigHandler(t)

	r := gin.New()
	r.PUT("/config/:key", h.Update)

	// 非法 key
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPut, "/config/nope",
		strings.NewReader(`{"value":"1"}`)))
	assert.Equal(t, http.StatusBadRequest, w.Code)

	// 非法 JSON（handler 透传绑定错误 -> 500）
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, httptest.NewRequest(http.MethodPut, "/config/log_level", strings.NewReader(`{bad`)))
	assert.Equal(t, http.StatusInternalServerError, w2.Code)

	// 成功
	w3 := httptest.NewRecorder()
	r.ServeHTTP(w3, httptest.NewRequest(http.MethodPut, "/config/log_level",
		strings.NewReader(`{"value":"debug"}`)))
	assert.Equal(t, http.StatusOK, w3.Code)
	val, err := mr.Get("jimu:config:log_level")
	assert.NoError(t, err)
	assert.Equal(t, "debug", val)

	// 服务错误
	r2 := gin.New()
	svc := application.NewAdminConfigService(redis.NewClient(&redis.Options{Addr: mr.Addr()}), &fakeEventBus{}, "jimu")
	r2.PUT("/config/:key", NewAdminConfigHandler(svc).Update)
	mr.Close()
	w4 := httptest.NewRecorder()
	r2.ServeHTTP(w4, httptest.NewRequest(http.MethodPut, "/config/log_level",
		strings.NewReader(`{"value":"debug"}`)))
	assert.Equal(t, http.StatusInternalServerError, w4.Code)
}

func TestAdminConfigHandlerReload(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h, mr, _ := newConfigHandler(t)
	assert.NoError(t, mr.Set("jimu:config:log_level", "info"))

	r := gin.New()
	r.POST("/config/reload", h.Reload)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/config/reload", nil))
	assert.Equal(t, http.StatusOK, w.Code)

	// 服务错误
	r2 := gin.New()
	svc := application.NewAdminConfigService(redis.NewClient(&redis.Options{Addr: mr.Addr()}), &fakeEventBus{}, "jimu")
	r2.POST("/config/reload", NewAdminConfigHandler(svc).Reload)
	mr.Close()
	w2 := httptest.NewRecorder()
	r2.ServeHTTP(w2, httptest.NewRequest(http.MethodPost, "/config/reload", nil))
	assert.Equal(t, http.StatusInternalServerError, w2.Code)
}
