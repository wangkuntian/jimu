package middleware

import (
	"strings"

	"jimu/internal/shared/errors"
	"jimu/internal/shared/response"

	"github.com/gin-gonic/gin"
)

// APIVersion API 版本信息
type APIVersion struct {
	Version    string // 版本号，如 "v1"
	Deprecated bool   // 是否已弃用
	Sunset     string // 弃用后的截止日期（RFC3339 格式）
}

// VersionConfig API 版本配置
type VersionConfig struct {
	// 支持的版本列表
	Versions map[string]APIVersion
	// 默认版本（未指定版本时使用）
	DefaultVersion string
	// 版本提取方式: "header", "path", "query"
	ExtractFrom string
	// Header 名称（ExtractFrom="header" 时）
	HeaderName string
	// Query 参数名（ExtractFrom="query" 时）
	QueryParam string
	// 是否要求客户端必须指定版本
	Required bool
	// 跳过版本检查的路径
	Skipper func(*gin.Context) bool
}

// DefaultVersionConfig 返回默认版本配置
func DefaultVersionConfig() VersionConfig {
	return VersionConfig{
		Versions: map[string]APIVersion{
			"v1": {Version: "v1", Deprecated: false},
		},
		DefaultVersion: "v1",
		ExtractFrom:    "path", // /api/v1/...
		HeaderName:     "X-Api-Version",
		QueryParam:     "api-version",
		Required:       false,
	}
}

// APIVersionMiddleware API 版本协商中间件
func APIVersionMiddleware(cfg VersionConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		if cfg.Skipper != nil && cfg.Skipper(c) {
			c.Next()
			return
		}

		version := extractVersion(c, cfg)

		// 检查版本是否有效
		if version == "" {
			if cfg.Required {
				response.Fail(c, errors.New(errors.CodeInvalidParam, "API version is required"))
				c.Abort()
				return
			}
			version = cfg.DefaultVersion
		}

		v, ok := cfg.Versions[version]
		if !ok {
			response.Fail(c, errors.New(errors.CodeInvalidParam, "unsupported API version: "+version))
			c.Abort()
			return
		}

		// 设置版本信息到 context
		c.Set("api_version", version)

		// 设置响应头
		c.Header("X-Api-Version", version)
		c.Header("X-Api-Latest-Version", cfg.DefaultVersion)

		if v.Deprecated {
			c.Header("Deprecation", "true")
			if v.Sunset != "" {
				c.Header("Sunset", v.Sunset)
			}
			// 添加 Warning 头（RFC 7234）
			c.Header("Warning", `299 - "API version `+version+` is deprecated, please use `+cfg.DefaultVersion+`"`)
		}

		c.Next()
	}
}

// extractVersion 从请求中提取版本号
func extractVersion(c *gin.Context, cfg VersionConfig) string {
	switch cfg.ExtractFrom {
	case "header":
		return c.GetHeader(cfg.HeaderName)
	case "query":
		return c.Query(cfg.QueryParam)
	case "path":
		return extractVersionFromPath(c.Request.URL.Path)
	default:
		return extractVersionFromPath(c.Request.URL.Path)
	}
}

// extractVersionFromPath 从 URL 路径提取版本号
// 支持格式: /api/v1/users, /v1/users
func extractVersionFromPath(path string) string {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	for _, part := range parts {
		if strings.HasPrefix(part, "v") && len(part) > 1 {
			// 简单验证：v 后跟数字
			if isNumeric(part[1:]) {
				return part
			}
		}
	}
	return ""
}

// isNumeric 检查字符串是否为数字
func isNumeric(s string) bool {
	if s == "" {
		return false
	}
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}
