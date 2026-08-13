package interfaces

import (
	"jimu/internal/config"
	"jimu/internal/modules/auth/application"
	"jimu/internal/platform/auth"
	"jimu/internal/platform/captcha"
	"jimu/internal/platform/http/middleware"

	"github.com/gin-gonic/gin"
)

func RegisterAuthRoutes(r *gin.RouterGroup, service *application.AuthService, jwtUtil *auth.JWT, cfg config.AuthConfig, limiter *auth.Limiter, captchaSvc *captcha.Service, captchaCfg config.CaptchaConfig) {
	handler := NewAuthHandler(service, cfg, limiter, captchaSvc, captchaCfg)
	authGroup := r.Group("/auth")
	{
		authGroup.POST("/login", middleware.ValidateJSON(&loginRequest{}), handler.Login)
		if cfg.PublicRegistration {
			authGroup.POST("/register", middleware.ValidateJSON(&loginRequest{}), handler.Register)
		}
		authGroup.POST("/refresh", middleware.ValidateJSON(&refreshRequest{}), handler.RefreshToken)
		authGroup.POST("/forgot-password", middleware.ValidateJSON(&forgotPasswordRequest{}), handler.ForgotPassword)
		authGroup.POST("/reset-password", middleware.ValidateJSON(&resetPasswordRequest{}), handler.ResetPassword)

		protected := authGroup.Group("")
		protected.Use(auth.AuthMiddleware(jwtUtil))
		protected.POST("/logout", handler.Logout)
		protected.POST("/logout-all", handler.LogoutAll)
	}
}

// RegisterCaptchaRoute 注册验证码路由
func RegisterCaptchaRoute(r *gin.RouterGroup, svc *captcha.Service) {
	if svc == nil {
		return
	}
	handler := captcha.NewCaptchaHandler(svc)
	r.GET("/captcha", handler.Generate)
}
