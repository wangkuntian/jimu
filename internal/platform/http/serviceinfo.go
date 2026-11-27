package http

import (
	"jimu/internal/config"
	"jimu/internal/shared/response"

	"github.com/gin-gonic/gin"
)

// 服务自述里可被生成器裁剪的入口；框架仓保持全部可用。
var (
	serviceName            = "jimu"
	serviceAPIDocs         = "/swagger/index.html"
	serviceMetrics         = "management 端口 /metrics（默认 9090）"
	serviceObservabilityUI = "OpenObserve http://localhost:5080（make up-observability 后可用）"
)

// serviceInfoHandler 返回 / 的服务自述。
func serviceInfoHandler(otelCfg config.OTELConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		serviceVersion := otelCfg.ServiceVersion
		if serviceVersion == "" {
			serviceVersion = "dev"
		}
		body := gin.H{
			"service":  serviceName,
			"version":  serviceVersion,
			"api_base": "/api/v1",
			"health":   "management 端口 /livez、/readyz（默认 9090）",
		}
		if serviceAPIDocs != "" {
			body["api_docs"] = serviceAPIDocs
		}
		if serviceMetrics != "" {
			body["metrics"] = serviceMetrics
		}
		if serviceObservabilityUI != "" {
			body["observability_ui"] = serviceObservabilityUI
		}
		response.OK(c, body)
	}
}
