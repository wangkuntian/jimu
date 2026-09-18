package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestTimeoutPropagatesContextDeadline(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(Timeout(5 * time.Millisecond))
	r.GET("/", func(c *gin.Context) {
		if _, ok := c.Request.Context().Deadline(); !ok {
			t.Fatal("request context missing deadline")
		}
		time.Sleep(20 * time.Millisecond)
		c.Status(http.StatusNoContent)
	})

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))
	// handler 已产出响应（204）时不被超时中间件覆盖
	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestTimeoutWritesGatewayTimeoutWhenNoResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(Timeout(5 * time.Millisecond))
	r.GET("/", func(c *gin.Context) {
		// 忽略 ctx 且不写响应：中间件应补 504
		time.Sleep(20 * time.Millisecond)
	})

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))
	assert.Equal(t, http.StatusGatewayTimeout, w.Code)
	assert.Contains(t, w.Body.String(), "1008")
}

func TestTimeoutDisabledWhenZero(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(Timeout(0))
	r.GET("/", func(c *gin.Context) {
		if _, ok := c.Request.Context().Deadline(); ok {
			t.Fatal("zero timeout should not set deadline")
		}
		c.Status(http.StatusNoContent)
	})

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))
	assert.Equal(t, http.StatusNoContent, w.Code)
}
