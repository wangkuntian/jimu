package interfaces

import (
	"jimu/internal/modules/auth/application"
	"jimu/internal/platform/auth"

	"github.com/gin-gonic/gin"
)

func RegisterAuthRoutes(r *gin.RouterGroup, service *application.AuthService, jwtUtil *auth.JWT) {
	handler := NewAuthHandler(service)
	auth := r.Group("/auth")
	{
		auth.POST("/login", handler.Login)
		auth.POST("/register", handler.Register)
		auth.POST("/refresh", handler.RefreshToken)
	}
	_ = jwtUtil // used by middleware, not directly here
}
