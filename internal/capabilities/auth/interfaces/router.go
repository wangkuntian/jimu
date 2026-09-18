package interfaces

import (
	"jimu/internal/capabilities/auth/application"
	"jimu/internal/capabilities/captcha"
	"jimu/internal/config"
	"jimu/internal/kernel/auth"
	"jimu/internal/kernel/http/middleware"
	"jimu/internal/shared/pagination"

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

		// WebAuthn/通行密钥：begin 为公开端点（无密码登录），finish 校验断言后签发 token
		authGroup.POST("/webauthn/login/begin", middleware.ValidateJSON(&webAuthnLoginBeginRequest{}), handler.BeginWebAuthnLogin)
		authGroup.POST("/webauthn/login/finish", handler.FinishWebAuthnLogin)

		protected := authGroup.Group("")
		protected.Use(auth.AuthMiddleware(jwtUtil))
		protected.POST("/logout", handler.Logout)
		protected.POST("/logout-all", handler.LogoutAll)
		protected.POST("/mfa/setup", handler.SetupTOTP)
		protected.POST("/mfa/enable", middleware.ValidateJSON(&enableTOTPRequest{}), handler.EnableTOTP)
		protected.POST("/mfa/disable", middleware.ValidateJSON(&disableTOTPRequest{}), handler.DisableTOTP)
		protected.GET("/login-history", middleware.ValidateQuery(&pagination.Pagination{}), handler.LoginHistory)
		protected.GET("/devices", handler.ListDevices)
		protected.DELETE("/devices", handler.RevokeAllDevices)
		protected.DELETE("/devices/:id", handler.RevokeDevice)
		protected.POST("/webauthn/register/begin", handler.BeginWebAuthnRegistration)
		protected.POST("/webauthn/register/finish", handler.FinishWebAuthnRegistration)
		protected.GET("/webauthn/credentials", handler.ListWebAuthnCredentials)
		protected.PUT("/webauthn/credentials/:id", middleware.ValidateJSON(&webAuthnRenameRequest{}), handler.RenameWebAuthnCredential)
		protected.DELETE("/webauthn/credentials/:id", handler.DeleteWebAuthnCredential)
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
