package interfaces

import (
	"jimu/internal/capabilities/passkey/application"
	"jimu/internal/config"
	"jimu/internal/kernel/auth"
	"jimu/internal/kernel/http/middleware"

	"github.com/gin-gonic/gin"
)

// RegisterPasskeyRoutes 注册通行密钥路由：
// login/begin、login/finish 公开（无密码登录）；register/* 与 credentials/* 受 JWT 保护。
func RegisterPasskeyRoutes(rg *gin.RouterGroup, service *application.PasskeyService, cfg config.AuthConfig, limiter *auth.Limiter, jwtUtil *auth.JWT) {
	handler := NewPasskeyHandler(service, cfg, limiter)

	// 公开：无密码登录
	rg.POST("/auth/webauthn/login/begin", middleware.ValidateJSON(&webAuthnLoginBeginRequest{}), handler.BeginWebAuthnLogin)
	rg.POST("/auth/webauthn/login/finish", handler.FinishWebAuthnLogin)

	// 受保护：凭证注册与管理
	protected := rg.Group("/auth/webauthn")
	protected.Use(auth.AuthMiddleware(jwtUtil))
	protected.POST("/register/begin", handler.BeginWebAuthnRegistration)
	protected.POST("/register/finish", handler.FinishWebAuthnRegistration)
	protected.GET("/credentials", handler.ListWebAuthnCredentials)
	protected.PUT("/credentials/:id", middleware.ValidateJSON(&webAuthnRenameRequest{}), handler.RenameWebAuthnCredential)
	protected.DELETE("/credentials/:id", handler.DeleteWebAuthnCredential)
}
