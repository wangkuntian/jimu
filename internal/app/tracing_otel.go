//go:build !jimu_no_otel

package app

import (
	platformhttp "jimu/internal/platform/http"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
)

func tracingMiddleware() platformhttp.TracingMiddleware {
	return func(serviceName string) gin.HandlerFunc {
		return otelgin.Middleware(serviceName)
	}
}
