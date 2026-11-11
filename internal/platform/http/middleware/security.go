package middleware

import (
	"net/http"
	"strings"

	"jimu/internal/config"

	"github.com/gin-gonic/gin"
)

func Security(cfg config.HTTPConfig) gin.HandlerFunc {
	allowedOrigins := make(map[string]struct{}, len(cfg.AllowedOrigins))
	for _, origin := range cfg.AllowedOrigins {
		allowedOrigins[origin] = struct{}{}
	}

	return func(c *gin.Context) {
		if cfg.MaxBodyBytes > 0 {
			if c.Request.ContentLength > cfg.MaxBodyBytes {
				c.AbortWithStatus(http.StatusRequestEntityTooLarge)
				return
			}
			c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, cfg.MaxBodyBytes)
		}

		origin := c.GetHeader("Origin")
		if origin != "" {
			addVary(c, "Origin")
			if _, ok := allowedOrigins[origin]; ok {
				c.Header("Access-Control-Allow-Origin", origin)
				c.Header("Access-Control-Allow-Methods", "GET,POST,PUT,PATCH,DELETE,OPTIONS")
				c.Header("Access-Control-Allow-Headers", "Authorization,Content-Type,X-Request-ID")
			}
		}

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

func addVary(c *gin.Context, value string) {
	current := c.Writer.Header().Get("Vary")
	for _, item := range strings.Split(current, ",") {
		if strings.TrimSpace(item) == value {
			return
		}
	}
	if current == "" {
		c.Header("Vary", value)
		return
	}
	c.Header("Vary", current+", "+value)
}
