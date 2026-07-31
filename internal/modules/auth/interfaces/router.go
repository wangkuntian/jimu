package interfaces

import (
	"jimu/internal/config"
	"jimu/internal/modules/auth/application"
	"jimu/internal/platform/auth"

	"github.com/gin-gonic/gin"
)

func RegisterAuthRoutes(r *gin.RouterGroup, service *application.AuthService, jwtUtil *auth.JWT, cfg config.AuthConfig, limiter *auth.Limiter) {
	handler := NewAuthHandler(service, cfg, limiter)
	authGroup := r.Group("/auth")
	{
		authGroup.POST("/login", handler.Login)
		if cfg.PublicRegistration {
			authGroup.POST("/register", handler.Register)
		}
		authGroup.POST("/refresh", handler.RefreshToken)

		protected := authGroup.Group("")
		protected.Use(auth.AuthMiddleware(jwtUtil))
		protected.POST("/logout", handler.Logout)
		protected.POST("/logout-all", handler.LogoutAll)
	}
}
