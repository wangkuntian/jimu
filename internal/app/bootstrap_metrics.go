//go:build !jimu_lean

package app

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var metricsHandler = func() http.Handler { return promhttp.Handler() }
