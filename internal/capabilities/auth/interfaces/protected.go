package interfaces

import (
	"jimu/internal/kernel/access"
	"jimu/internal/kernel/auth"
	"jimu/internal/kernel/tenant"

	"github.com/casbin/casbin/v3"
	"github.com/gin-gonic/gin"
)

func ProtectedMiddleware(jwtUtil *auth.JWT, store access.AuthorizationStore, enforcer *casbin.Enforcer) []gin.HandlerFunc {
	return []gin.HandlerFunc{
		auth.AuthMiddleware(jwtUtil),
		access.AuthorizationMiddleware(store, enforcer),
		tenant.Middleware(),
	}
}
