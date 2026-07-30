package http

import (
	"net/http"
	"net/http/pprof"
	"time"

	"jimu/internal/config"
	"jimu/internal/platform/observability"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func HealthRouter(readiness *observability.Readiness, enablePprof bool) http.Handler {
	mux := http.NewServeMux()
	observability.RegisterHealth(mux, readiness)
	mux.Handle("/metrics", promhttp.Handler())
	if enablePprof {
		mux.HandleFunc("/debug/pprof/", pprof.Index)
		mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
		mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
		mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
		mux.HandleFunc("/debug/pprof/trace", pprof.Trace)
		for _, name := range []string{"allocs", "block", "goroutine", "heap", "mutex", "threadcreate"} {
			mux.Handle("/debug/pprof/"+name, pprof.Handler(name))
		}
	}
	return mux
}

func NewManagementServer(cfg config.ManagementConfig, handler http.Handler) *Server {
	return newServer(&http.Server{
		Addr:              formatAddr(cfg.Host, cfg.Port),
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       30 * time.Second,
	})
}
