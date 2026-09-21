package interfaces

import (
	"jimu/internal/capabilities/auth/application"
	"jimu/internal/config"
	"jimu/internal/contract"
	"jimu/internal/kernel/auth"
	"jimu/internal/kernel/http/middleware"
	"jimu/internal/shared/pagination"

	"github.com/gin-gonic/gin"
)

// RegisterAuthRoutes 注册认证路由：公开（登录/注册/刷新/找回重置）+ 受保护（登出/登录历史）。
// 验证码路由由 captcha 能力自挂（GET /api/v1/captcha）。
func RegisterAuthRoutes(r *gin.RouterGroup, service *application.AuthService, jwtUtil *auth.JWT, cfg config.AuthConfig, limiter *auth.Limiter, captchaVerifier contract.CaptchaVerifier) {
	handler := NewAuthHandler(service, cfg, limiter, captchaVerifier)
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
		protected.GET("/login-history", middleware.ValidateQuery(&pagination.Pagination{}), handler.LoginHistory)
	}
}
