package interfaces

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"jimu/internal/platform/feature"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestAdminFeatureHandlerList(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// manager 为 nil
	r := gin.New()
	r.GET("/features", NewAdminFeatureHandler(nil).List)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/features", nil))
	assert.Equal(t, http.StatusOK, w.Code)

	// 有 flag
	mgr := feature.NewManager()
	mgr.Register(feature.Flag{Name: "dark_mode", Enabled: true})
	r2 := gin.New()
	r2.GET("/features", NewAdminFeatureHandler(mgr).List)
	w2 := httptest.NewRecorder()
	r2.ServeHTTP(w2, httptest.NewRequest(http.MethodGet, "/features", nil))
	assert.Equal(t, http.StatusOK, w2.Code)
	assert.Contains(t, w2.Body.String(), "dark_mode")
}

func TestAdminFeatureHandlerUpdate(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mgr := feature.NewManager()
	mgr.Register(feature.Flag{Name: "dark_mode", Enabled: false, Percentage: 0})
	h := NewAdminFeatureHandler(mgr)
	r := gin.New()
	r.PUT("/features/:name", h.Update)

	// 成功更新
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPut, "/features/dark_mode",
		strings.NewReader(`{"enabled":true,"percentage":50}`)))
	assert.Equal(t, http.StatusOK, w.Code)
	flag, ok := mgr.Get("dark_mode")
	assert.True(t, ok)
	assert.True(t, flag.Enabled)
	assert.Equal(t, 50, flag.Percentage)

	// 只更新 enabled
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, httptest.NewRequest(http.MethodPut, "/features/dark_mode",
		strings.NewReader(`{"enabled":false}`)))
	assert.Equal(t, http.StatusOK, w2.Code)

	// flag 不存在
	w3 := httptest.NewRecorder()
	r.ServeHTTP(w3, httptest.NewRequest(http.MethodPut, "/features/nope",
		strings.NewReader(`{"enabled":true}`)))
	assert.Equal(t, http.StatusNotFound, w3.Code)

	// 非法 JSON
	w4 := httptest.NewRecorder()
	r.ServeHTTP(w4, httptest.NewRequest(http.MethodPut, "/features/dark_mode", strings.NewReader(`{bad`)))
	assert.Equal(t, http.StatusBadRequest, w4.Code)

	// manager 为 nil
	r2 := gin.New()
	r2.PUT("/features/:name", NewAdminFeatureHandler(nil).Update)
	w5 := httptest.NewRecorder()
	r2.ServeHTTP(w5, httptest.NewRequest(http.MethodPut, "/features/dark_mode",
		strings.NewReader(`{"enabled":true}`)))
	assert.Equal(t, http.StatusInternalServerError, w5.Code)
}
