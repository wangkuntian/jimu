package middleware

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"jimu/internal/platform/logger"
	"jimu/internal/shared/errors"
	"jimu/internal/shared/response"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// LogConfig 日志中间件配置
type LogConfig struct {
	LogRequestBody  bool // 是否记录请求体
	LogResponseBody bool // 是否记录响应体
	MaxBodyLogSize  int  // 最大记录体字节数
}

// DefaultLogConfig 返回默认日志配置
func DefaultLogConfig() LogConfig {
	return LogConfig{
		LogRequestBody:  false,
		LogResponseBody: false,
		MaxBodyLogSize:  1024,
	}
}

func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = uuid.New().String()
		}
		c.Set("request_id", requestID)
		c.Header("X-Request-ID", requestID)
		c.Next()
	}
}

func Logger(log *logger.Logger, cfg LogConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		method := c.Request.Method

		// 捕获请求体
		var requestBody string
		if cfg.LogRequestBody && shouldLogBody(c.Request.Header.Get("Content-Type")) {
			body, _ := c.GetRawData()
			if len(body) > 0 {
				// 重新设置 body 以便后续 handler 可以读取
				c.Request.Body = io.NopCloser(bytes.NewBuffer(body))
				if len(body) > cfg.MaxBodyLogSize {
					requestBody = sanitizeBody(string(body[:cfg.MaxBodyLogSize])) + "...[truncated]"
				} else {
					requestBody = sanitizeBody(string(body))
				}
			}
		}

		// 捕获响应体
		var responseBody string
		var truncated bool
		if cfg.LogResponseBody {
			w := newResponseBodyWriter(c.Writer, cfg.MaxBodyLogSize)
			c.Writer = w
		}

		c.Next()

		latency := time.Since(start)
		status := c.Writer.Status()

		// 获取捕获的响应体
		if cfg.LogResponseBody {
			if w, ok := c.Writer.(*responseBodyWriter); ok {
				responseBody = w.capturedBody()
				truncated = w.isTruncated()
			}
		}

		// 构建日志字段
		fields := []interface{}{
			"method", method,
			"path", path,
			"status", status,
			"latency", latency.String(),
			"request_id", c.GetString("request_id"),
			"client_ip", c.ClientIP(),
			"user_agent", c.Request.UserAgent(),
		}

		if requestBody != "" {
			fields = append(fields, "request_body", requestBody)
		}
		if responseBody != "" {
			fields = append(fields, "response_body", responseBody)
			if truncated {
				fields = append(fields, "response_truncated", true)
			}
		}

		log.WithContext(c.Request.Context()).Infow("request", fields...)
	}
}

// shouldLogBody 判断是否应该记录该 Content-Type 的 body
func shouldLogBody(contentType string) bool {
	if contentType == "" {
		return false
	}
	// 只记录文本类请求体
	textTypes := []string{
		"application/json",
		"application/xml",
		"application/x-www-form-urlencoded",
		"text/",
	}
	for _, t := range textTypes {
		if strings.Contains(contentType, t) {
			return true
		}
	}
	return false
}

// Recovery 返回 panic 恢复中间件。传入 reporter 时，panic 会上报到错误追踪平台
// （Sentry 等），同时返回统一 500 响应；不传则仅返回统一响应。
func Recovery(reporters ...Reporter) gin.HandlerFunc {
	rep := firstReporter(reporters)
	return gin.CustomRecovery(func(c *gin.Context, err interface{}) {
		if rep != nil {
			rep.Report(c.Request.Context(), fmt.Errorf("panic recovered: %v", err),
				"method", c.Request.Method, "path", c.Request.URL.Path, "client_ip", c.ClientIP())
		}
		response.Fail(c, errors.New(errors.CodeInternalError, "internal server error"))
	})
}

// Reporter 错误上报接口（避免反向依赖 platform/reporter 包）。
type Reporter interface {
	Report(ctx context.Context, err error, attrs ...string)
}

func firstReporter(reporters []Reporter) Reporter {
	for _, r := range reporters {
		if r != nil {
			return r
		}
	}
	return nil
}

// CORSConfig CORS 配置
type CORSConfig struct {
	AllowedOrigins   []string // 允许的来源，["*"] 表示允许所有
	AllowedMethods   []string // 允许的方法
	AllowedHeaders   []string // 允许的头部
	AllowCredentials bool     // 是否允许携带凭证
	MaxAge           int      // 预请求缓存秒数
}

// CORSMiddleware 基于配置的 CORS 中间件
func CORSMiddleware(cfg CORSConfig) gin.HandlerFunc {
	origins := cfg.AllowedOrigins
	methods := strings.Join(cfg.AllowedMethods, ",")
	headers := strings.Join(cfg.AllowedHeaders, ",")
	credentials := "false"
	if cfg.AllowCredentials {
		credentials = "true"
	}
	maxAge := strconv.Itoa(cfg.MaxAge)

	// 是否允许所有来源
	allOrigins := len(origins) == 1 && origins[0] == "*"

	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if allOrigins {
			c.Header("Access-Control-Allow-Origin", "*")
		} else if isAllowedOrigin(origin, origins) {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Vary", "Origin")
		}
		c.Header("Access-Control-Allow-Methods", methods)
		c.Header("Access-Control-Allow-Headers", headers)
		c.Header("Access-Control-Allow-Credentials", credentials)
		c.Header("Access-Control-Max-Age", maxAge)

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	}
}

func isAllowedOrigin(origin string, allowed []string) bool {
	for _, o := range allowed {
		if o == origin {
			return true
		}
	}
	return false
}
