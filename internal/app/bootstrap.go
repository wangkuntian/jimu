package app

import (
	"jimu/internal/config"
	"jimu/internal/contract"
	"jimu/internal/platform/http"
	"jimu/internal/platform/observability"
)

func Bootstrap(modules ...contract.Module) *http.Server {
	cfg, err := config.Load()
	if err != nil {
		panic("failed to load config: " + err.Error())
	}

	container, err := NewContainer(cfg)
	if err != nil {
		panic("failed to create container: " + err.Error())
	}

	r := http.SetupRouter(container.Logger, cfg.HTTP, cfg.Server)

	// Health check (no auth required)
	healthGroup := r.Group("/")
	observability.Register(healthGroup, container.DB, container.Redis)

	// Debug routes (pprof + metrics)
	observability.RegisterDebugRoutes(r.Group("/debug"))

	// Swagger UI
	http.RegisterSwagger(r.Group("/swagger"))

	// Register modules
	for _, m := range modules {
		m.RegisterHTTP(r)
		container.Logger.Info("module registered", "name", m.Name())
	}

	return http.NewServer(cfg.HTTP, r)
}
