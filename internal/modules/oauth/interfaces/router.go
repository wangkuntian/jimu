// internal/modules/oauth/interfaces/router.go
package interfaces

import (
	"jimu/internal/modules/oauth/application"

	"github.com/gin-gonic/gin"
)

// RegisterOAuthRoutes 注册 OAuth 路由
func RegisterOAuthRoutes(r *gin.RouterGroup, service *application.OAuthService) {
	handler := NewOAuthHandler(service)
	group := r.Group("/oauth")
	{
		group.GET("/:provider/login", handler.Login)
		group.GET("/:provider/callback", handler.Callback)
	}
}
