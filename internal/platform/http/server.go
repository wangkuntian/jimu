package http

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"

	"jimu/internal/config"
	"jimu/internal/platform/http/middleware"
	"jimu/internal/platform/logger"

	"github.com/gin-gonic/gin"
)

type Server struct {
	*http.Server
	activeRequests int64 // 活跃请求计数
}

func NewServer(cfg config.HTTPConfig, r *gin.Engine) *Server {
	srv := &http.Server{
		Addr:    fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Handler: r,
	}

	return &Server{Server: srv}
}

// Run 启动服务并等待退出信号
func (s *Server) Run() error {
	// 包装 handler 统计活跃请求
	originalHandler := s.Handler
	s.Handler = &requestTracker{handler: originalHandler, counter: &s.activeRequests}

	go func() {
		if err := s.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Printf("server failed: %v\n", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit
	fmt.Printf("received signal: %v, shutting down...\n", sig)

	// 给负载均衡器/反向代理一点时间停止发送新请求
	time.Sleep(1 * time.Second)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 等待活跃请求完成（最多等 shutdown timeout）
	go func() {
		ticker := time.NewTicker(500 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if atomic.LoadInt64(&s.activeRequests) == 0 {
					cancel()
					return
				}
			}
		}
	}()

	return s.Shutdown(ctx)
}

// requestTracker 包装 http.Handler 统计活跃请求
type requestTracker struct {
	handler http.Handler
	counter *int64
}

func (rt *requestTracker) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	atomic.AddInt64(rt.counter, 1)
	defer atomic.AddInt64(rt.counter, -1)
	rt.handler.ServeHTTP(w, r)
}

func SetupRouter(log *logger.Logger, cfg config.HTTPConfig) *gin.Engine {
	if cfg.Mode == "release" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	r.Use(
		middleware.RequestID(),
		middleware.Logger(log),
		middleware.Recovery(),
		middleware.CORS(),
		middleware.Timeout(30*time.Second),
		middleware.GlobalRateLimit(),
	)

	return r
}
