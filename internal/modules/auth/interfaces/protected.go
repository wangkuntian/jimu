package interfaces

import (
	platformauth "jimu/internal/platform/auth"

	"github.com/casbin/casbin/v2"
	"github.com/gin-gonic/gin"
)

func ProtectedMiddleware(jwtUtil *platformauth.JWT, store platformauth.AuthorizationStore, enforcer *casbin.Enforcer) []gin.HandlerFunc {
	return []gin.HandlerFunc{
		platformauth.AuthMiddleware(jwtUtil),
		platformauth.AuthorizationMiddleware(store, enforcer),
	}
}
