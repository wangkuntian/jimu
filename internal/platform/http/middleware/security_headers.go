package middleware

import (
	"github.com/gin-gonic/gin"
)

// SecurityHeadersConfig 安全头配置
type SecurityHeadersConfig struct {
	ContentTypeOptions    string // X-Content-Type-Options, 默认 "nosniff"
	FrameOptions          string // X-Frame-Options, 默认 "DENY"
	XSSProtection         string // X-XSS-Protection, 默认 "1; mode=block"
	StrictTransport       string // Strict-Transport-Transport-Security, 默认 "max-age=31536000; includeSubDomains"
	ContentSecurityPolicy string // Content-Security-Policy, 默认 "default-src 'self'"
	ReferrerPolicy        string // Referrer-Policy, 默认 "strict-origin-when-cross-origin"
	PermissionsPolicy     string // Permissions-Policy, 默认 "camera=(), microphone=(), geolocation=()"
}

// DefaultSecurityHeadersConfig 返回默认安全头配置
func DefaultSecurityHeadersConfig() SecurityHeadersConfig {
	return SecurityHeadersConfig{
		ContentTypeOptions:    "nosniff",
		FrameOptions:          "DENY",
		XSSProtection:         "1; mode=block",
		StrictTransport:       "max-age=31536000; includeSubDomains",
		ContentSecurityPolicy: "default-src 'self'",
		ReferrerPolicy:        "strict-origin-when-cross-origin",
		PermissionsPolicy:     "camera=(), microphone=(), geolocation=()",
	}
}

// SecurityHeaders 设置安全响应头
func SecurityHeaders(cfg SecurityHeadersConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
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
		c.Next()
	}
}
