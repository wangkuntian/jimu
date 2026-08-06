package http

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"time"

	"jimu/internal/config"
	"jimu/internal/platform/http/middleware"
	"jimu/internal/platform/logger"
	"jimu/internal/platform/observability"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
)

type Server struct {
	*http.Server
	errors chan error
}

func NewServer(cfg config.HTTPConfig, r *gin.Engine) *Server {
	return newServer(&http.Server{
		Addr:              formatAddr(cfg.Host, cfg.Port),
		Handler:           r,
		ReadHeaderTimeout: time.Duration(cfg.ReadHeaderTimeoutSec) * time.Second,
		ReadTimeout:       time.Duration(cfg.ReadTimeoutSec) * time.Second,
		WriteTimeout:      time.Duration(cfg.WriteTimeoutSec) * time.Second,
		IdleTimeout:       time.Duration(cfg.IdleTimeoutSec) * time.Second,
	})
}

func newServer(server *http.Server) *Server {
	return &Server{Server: server, errors: make(chan error, 1)}
}

func formatAddr(host string, port int) string {
	return fmt.Sprintf("%s:%d", host, port)
}

func (s *Server) Start(ctx context.Context) error {
	var listener net.ListenConfig
	ln, err := listener.Listen(ctx, "tcp", s.Addr)
	if err != nil {
		return err
	}
	go func() {
		var serveErr error
		if s.TLSConfig != nil {
			serveErr = s.ServeTLS(ln, "", "")
		} else {
			serveErr = s.Serve(ln)
		}
		if serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) {
			s.errors <- serveErr
		}
	}()
	return nil
}

func (s *Server) Stop(ctx context.Context) error {
	return s.Shutdown(ctx)
}

func (s *Server) Errors() <-chan error {
	return s.errors
}

func SetupRouter(log *logger.Logger, cfg config.HTTPConfig, serverCfg config.ServerConfig, securityCfg config.SecurityConfig, otelCfg observability.TracingConfig) *gin.Engine {
	switch cfg.Mode {
	case config.HTTPModeRelease:
		gin.SetMode(gin.ReleaseMode)
	case config.HTTPModeTest:
		gin.SetMode(gin.TestMode)
	}

	r := gin.New()
	r.Use(
		middleware.RequestID(),
	)
	// OpenTelemetry 追踪中间件（在 Logger 之前，确保 span 可记录到日志）
	if otelCfg.Enabled {
		serviceName := otelCfg.ServiceName
		if serviceName == "" {
			serviceName = "jimu"
		}
		r.Use(otelgin.Middleware(serviceName))
	}
	r.Use(
		middleware.CORSMiddleware(middleware.CORSConfig{
			AllowedOrigins: cfg.AllowedOrigins,
			MaxAge:        86400,
		}),
		middleware.SecurityHeadersFromConfig(securityCfg),
		middleware.GzipCompression(),
		middleware.Logger(log, middleware.DefaultLogConfig()),
		middleware.Security(cfg),
		middleware.Recovery(),
		middleware.Timeout(time.Duration(serverCfg.TimeoutSec)*time.Second),
		middleware.GlobalRateLimit(serverCfg.RateLimitRate, serverCfg.RateLimitBurst),
	)

	return r
}

func ConfigureTrustedProxies(engine *gin.Engine, proxies []string) error {
	if len(proxies) == 0 {
		proxies = nil
	}
	return engine.SetTrustedProxies(proxies)
}
