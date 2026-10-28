package auth

import (
	"context"
	"strings"

	"jimu/internal/shared/errors"
	"jimu/internal/shared/response"

	"github.com/gin-gonic/gin"
)

// TenantStateChecker 由租户能力实现；内核只依赖最小只读能力，不感知租户存储。
type TenantStateChecker interface {
	TenantActive(ctx context.Context, tenantID uint64) (bool, error)
}

// UserStateChecker 由用户能力实现；内核只依赖最小只读能力，不感知用户存储。
type UserStateChecker interface {
	UserActive(ctx context.Context, userID uint64) (bool, error)
}

// StateCheckers 为 JWT 鉴权链可选的实时身份状态检查。
type StateCheckers struct {
	Tenant TenantStateChecker
	User   UserStateChecker
}

func AuthMiddleware(jwtUtil *JWT, stateCheckers ...StateCheckers) gin.HandlerFunc {
	var tenantChecker TenantStateChecker
	var userChecker UserStateChecker
	if len(stateCheckers) > 0 {
		tenantChecker = stateCheckers[0].Tenant
		userChecker = stateCheckers[0].User
	}
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

		claims, err := jwtUtil.Parse(parts[1], TokenTypeAccess)
		if err != nil {
			response.Fail(c, errors.New(errors.CodeUnauthorized, "invalid token"))
			c.Abort()
			return
		}

		if userChecker != nil {
			active, err := userChecker.UserActive(c.Request.Context(), claims.UserID)
			if err != nil {
				response.Fail(c, errors.Wrap(errors.CodeInternalError, "failed to check user state", err))
				c.Abort()
				return
			}
			if !active {
				response.Fail(c, errors.New(errors.CodeUnauthorized, "user is disabled or deleted"))
				c.Abort()
				return
			}
		}

		// tid=0 是已定义的可信平台级上下文，没有具体租户可校验。
		if tenantChecker != nil && claims.TenantID != 0 {
			active, err := tenantChecker.TenantActive(c.Request.Context(), claims.TenantID)
			if err != nil {
				response.Fail(c, errors.Wrap(errors.CodeInternalError, "failed to check tenant state", err))
				c.Abort()
				return
			}
			if !active {
				response.Fail(c, errors.New(errors.CodeUnauthorized, "tenant is disabled or deleted"))
				c.Abort()
				return
			}
		}

		c.Set("user_id", claims.UserID)
		c.Set("session_id", claims.SessionID)
		c.Set("tenant_id", claims.TenantID)
		c.Next()
	}
}
