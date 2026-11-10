package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestConcurrencyLimitDisabled(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(ConcurrencyLimit(0, 0))
	r.GET("/", func(c *gin.Context) { c.Status(http.StatusNoContent) })

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))
	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestConcurrencyLimitShedsWhenFull(t *testing.T) {
	gin.SetMode(gin.TestMode)
	release := make(chan struct{})
	inHandler := make(chan struct{}, 1)

	r := gin.New()
	r.Use(ConcurrencyLimit(1, 0))
	r.GET("/", func(c *gin.Context) {
		inHandler <- struct{}{}
		<-release
		c.Status(http.StatusNoContent)
	})

	// 占满唯一名额
	go func() {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))
	}()
	<-inHandler

	// 第二个请求立即被拒绝（不排队）
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))
	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
	assert.Contains(t, w.Body.String(), "1010")

	close(release)
}

func TestConcurrencyLimitWaitsThenSucceeds(t *testing.T) {
	gin.SetMode(gin.TestMode)
	release := make(chan struct{})
	inHandler := make(chan struct{}, 1)

	r := gin.New()
	r.Use(ConcurrencyLimit(1, 300*time.Millisecond))
	r.GET("/", func(c *gin.Context) {
		inHandler <- struct{}{}
		<-release
		c.Status(http.StatusNoContent)
	})

	go func() {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))
	}()
	<-inHandler

	// 排队期间名额释放 → 第二个请求正常处理
	go func() {
		time.Sleep(20 * time.Millisecond)
		close(release)
	}()
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))
	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestConcurrencyLimitShedsAfterWaitTimeout(t *testing.T) {
	gin.SetMode(gin.TestMode)
	release := make(chan struct{})
	inHandler := make(chan struct{}, 1)

	r := gin.New()
	r.Use(ConcurrencyLimit(1, 20*time.Millisecond))
	r.GET("/", func(c *gin.Context) {
		inHandler <- struct{}{}
		<-release
		c.Status(http.StatusNoContent)
	})

	go func() {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))
	}()
	<-inHandler

	// 排队等待超时 → 拒绝
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))
	assert.Equal(t, http.StatusServiceUnavailable, w.Code)

	close(release)
}
