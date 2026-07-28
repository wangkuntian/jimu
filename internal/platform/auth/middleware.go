package auth

import (
	"strings"

	"jimu/internal/shared/errors"
	"jimu/internal/shared/response"

	"github.com/gin-gonic/gin"
)

func AuthMiddleware(jwtUtil *JWT) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			response.Fail(c, errors.New(errors.CodeUnauthorized, "missing authorization header"))
			c.Abort()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			response.Fail(c, errors.New(errors.CodeUnauthorized, "invalid authorization format"))
			c.Abort()
			return
		}

		claims, err := jwtUtil.Parse(parts[1])
		if err != nil {
			response.Fail(c, errors.New(errors.CodeUnauthorized, "invalid token"))
			c.Abort()
			return
		}

		c.Set("user_id", claims.UserID)
		c.Next()
	}
}
