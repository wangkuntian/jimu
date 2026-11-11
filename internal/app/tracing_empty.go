//go:build jimu_no_otel

package app

import platformhttp "jimu/internal/platform/http"

func tracingMiddleware() platformhttp.TracingMiddleware {
	return nil
}
