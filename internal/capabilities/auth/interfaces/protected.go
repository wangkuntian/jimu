package interfaces

import (
	platformauth "jimu/internal/kernel/auth"
	"jimu/internal/kernel/tenant"

	"github.com/casbin/casbin/v3"
	"github.com/gin-gonic/gin"
)

func ProtectedMiddleware(jwtUtil *platformauth.JWT, store platformauth.AuthorizationStore, enforcer *casbin.Enforcer) []gin.HandlerFunc {
	return []gin.HandlerFunc{
		platformauth.AuthMiddleware(jwtUtil),
		platformauth.AuthorizationMiddleware(store, enforcer),
		tenant.Middleware(),
	}
}
