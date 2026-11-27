//go:build jimu_lean

package app

import "net/http"

// lean 构建不暴露 /metrics。
var metricsHandler func() http.Handler
