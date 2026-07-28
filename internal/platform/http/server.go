package http

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"jimu/internal/config"
	"jimu/internal/platform/http/middleware"
	"jimu/internal/platform/logger"

	"github.com/gin-gonic/gin"
)

type Server struct {
	*http.Server
}

func NewServer(cfg config.HTTPConfig, r *gin.Engine) *Server {
	srv := &http.Server{
		Addr:    fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Handler: r,
	}

	return &Server{Server: srv}
}

func (s *Server) Run() error {
	go func() {
		if err := s.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Printf("server failed: %v\n", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return s.Shutdown(ctx)
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
	)

	return r
}
