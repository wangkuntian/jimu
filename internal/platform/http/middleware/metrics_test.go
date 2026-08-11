package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupMetricsEngine 构建带 Metrics 中间件的测试引擎
func setupMetricsEngine() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(Metrics())
	r.GET("/api/v1/users/:id", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})
	return r
}

// gatherHTTPMetric 从 default registry 收集 jimu_http 指标并按 path 汇总
func gatherHTTPMetric(t *testing.T) map[string]float64 {
	t.Helper()
	families, err := prometheus.DefaultGatherer.Gather()
	require.NoError(t, err)
	byPath := make(map[string]float64)
	for _, mf := range families {
		if mf.GetName() != "jimu_http_requests_total" {
			continue
		}
		for _, m := range mf.GetMetric() {
			for _, lv := range m.GetLabel() {
				if lv.GetName() == "path" {
					byPath[lv.GetValue()] += m.GetCounter().GetValue()
				}
			}
		}
	}
	return byPath
}

// TestMetricsLabelsUseRouteTemplate 验证 path 标签使用路由模板而非实际路径
func TestMetricsLabelsUseRouteTemplate(t *testing.T) {
	httpRequestsTotal.Reset()
	r := setupMetricsEngine()
	for _, id := range []string{"1", "2"} {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/users/"+id, nil))
		assert.Equal(t, http.StatusOK, w.Code)
	}

	byPath := gatherHTTPMetric(t)
	assert.Equal(t, float64(2), byPath["/api/v1/users/:id"], "两个不同 ID 应合并为同一路由模板序列")
	assert.NotContains(t, byPath, "/api/v1/users/1")
	assert.NotContains(t, byPath, "/api/v1/users/2")
}

// TestMetricsRecordsUnmatchedPath 验证 404 未匹配路由回退到原始路径
func TestMetricsRecordsUnmatchedPath(t *testing.T) {
	httpRequestsTotal.Reset()
	r := setupMetricsEngine()
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/no/such/route", nil))
	assert.Equal(t, http.StatusNotFound, w.Code)

	byPath := gatherHTTPMetric(t)
	assert.Equal(t, float64(1), byPath["/no/such/route"], "未匹配路由应记录原始路径")
}
