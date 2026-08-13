package interfaces

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"jimu/internal/config"
	"jimu/internal/modules/admin/application"
	"jimu/internal/platform/logger"
	"jimu/internal/platform/scheduler"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func newTaskHandler(t *testing.T) *AdminTaskHandler {
	t.Helper()
	s := scheduler.NewWithStore(logger.New(config.LogConfig{Level: "warn", Format: "console", Output: "stdout"}), scheduler.NewMemoryStore(), nil)
	if err := s.AddNamedFunc("j1", "Job One", "@every 10s", func() {}); err != nil {
		t.Fatalf("AddNamedFunc error: %v", err)
	}
	return NewAdminTaskHandler(application.NewAdminTaskService(s))
}

func TestAdminTaskHandlerList(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/tasks", newTaskHandler(t).List)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/tasks", nil))
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "j1")
}

func TestAdminTaskHandlerTrigger(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/tasks/:id/run", newTaskHandler(t).Trigger)

	// 成功
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/tasks/j1/run", nil))
	assert.Equal(t, http.StatusOK, w.Code)

	// 任务不存在
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, httptest.NewRequest(http.MethodPost, "/tasks/nope/run", nil))
	assert.Equal(t, http.StatusInternalServerError, w2.Code)
}

func TestAdminTaskHandlerToggle(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/tasks/:id/toggle", newTaskHandler(t).Toggle)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/tasks/j1/toggle", nil))
	assert.Equal(t, http.StatusOK, w.Code)

	// 任务不存在
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, httptest.NewRequest(http.MethodPost, "/tasks/nope/toggle", nil))
	assert.Equal(t, http.StatusNotFound, w2.Code)
}

func TestAdminTaskHandlerHistory(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/tasks/:id/history", newTaskHandler(t).History)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/tasks/j1/history", nil))
	assert.Equal(t, http.StatusOK, w.Code)
}
