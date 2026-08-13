package interfaces

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"jimu/internal/modules/admin/application"

	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
)

func newMonitoringHandler(t *testing.T) *AdminMonitoringHandler {
	t.Helper()
	mr, err := miniredis.Run()
	assert.NoError(t, err)
	t.Cleanup(mr.Close)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	svc := application.NewAdminMonitoringService("v1", "test", client)
	return NewAdminMonitoringHandler(svc)
}

func TestAdminMonitoringHandlerStatus(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/status", newMonitoringHandler(t).Status)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/status", nil))
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "v1")
}

func TestAdminMonitoringHandlerHealth(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/health", newMonitoringHandler(t).Health)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/health", nil))
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "\"redis\":true")
}

func TestAdminMonitoringHandlerMetrics(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/metrics", newMonitoringHandler(t).Metrics)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "goroutines")
}
