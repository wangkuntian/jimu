package middleware

import (
	"jimu/internal/config"

	"github.com/gin-gonic/gin"
)

// SecurityHeadersFromConfig 从项目配置创建安全头中间件
func SecurityHeadersFromConfig(cfg config.SecurityConfig) gin.HandlerFunc {
	// 空值填充默认值
	d := config.DefaultSecurityConfig()
	if cfg.ContentTypeOptions == "" {
		cfg.ContentTypeOptions = d.ContentTypeOptions
	}
	if cfg.FrameOptions == "" {
		cfg.FrameOptions = d.FrameOptions
	}
	if cfg.XSSProtection == "" {
		cfg.XSSProtection = d.XSSProtection
	}
	if cfg.StrictTransport == "" {
		cfg.StrictTransport = d.StrictTransport
	}
	if cfg.ContentSecurityPolicy == "" {
		cfg.ContentSecurityPolicy = d.ContentSecurityPolicy
	}
	if cfg.ReferrerPolicy == "" {
		cfg.ReferrerPolicy = d.ReferrerPolicy
	}
	if cfg.PermissionsPolicy == "" {
		cfg.PermissionsPolicy = d.PermissionsPolicy
	}

	return func(c *gin.Context) {
		writeSecurityHeaders(c, cfg)
		c.Next()
	}
}

// writeSecurityHeaders 写入安全头
func writeSecurityHeaders(c *gin.Context, cfg config.SecurityConfig) {
	if cfg.ContentTypeOptions != "" {
		c.Header("X-Content-Type-Options", cfg.ContentTypeOptions)
	}
	if cfg.FrameOptions != "" {
		c.Header("X-Frame-Options", cfg.FrameOptions)
	}
	if cfg.XSSProtection != "" {
		c.Header("X-XSS-Protection", cfg.XSSProtection)
	}
	if cfg.StrictTransport != "" {
		c.Header("Strict-Transport-Security", cfg.StrictTransport)
	}
	if cfg.ContentSecurityPolicy != "" {
		c.Header("Content-Security-Policy", cfg.ContentSecurityPolicy)
	}
	if cfg.ReferrerPolicy != "" {
		c.Header("Referrer-Policy", cfg.ReferrerPolicy)
	}
	if cfg.PermissionsPolicy != "" {
		c.Header("Permissions-Policy", cfg.PermissionsPolicy)
	}
}
