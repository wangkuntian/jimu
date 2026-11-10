package interfaces

import (
	"jimu/internal/kernel/tenant"
	"jimu/internal/platform/authz"
	"jimu/internal/platform/identity"

	"github.com/casbin/casbin/v3"
	"github.com/gin-gonic/gin"
)

func ProtectedMiddleware(jwtUtil *identity.JWT, store authz.AuthorizationStore, enforcer *casbin.Enforcer, stateCheckers ...identity.StateCheckers) []gin.HandlerFunc {
	return []gin.HandlerFunc{
		identity.AuthMiddleware(jwtUtil, stateCheckers...),
		authz.AuthorizationMiddleware(store, enforcer),
		tenant.Middleware(),
	}
}
