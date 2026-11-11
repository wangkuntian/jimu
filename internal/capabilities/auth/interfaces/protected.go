package interfaces

import (
	"jimu/internal/platform/authz"
	"jimu/internal/platform/identity"
	"jimu/internal/platform/tenant"

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
