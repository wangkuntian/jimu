package interfaces

import (
	"fmt"
	"strings"
	"time"

	"jimu/internal/config"
	"jimu/internal/modules/auth/application"
	platformauth "jimu/internal/platform/auth"
	"jimu/internal/platform/captcha"
	"jimu/internal/shared/errors"
	"jimu/internal/shared/response"

	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	service    *application.AuthService
	cfg        config.AuthConfig
	limiter    *platformauth.Limiter
	captcha    *captcha.Service
	captchaCfg config.CaptchaConfig
}

func NewAuthHandler(service *application.AuthService, cfg config.AuthConfig, limiter *platformauth.Limiter, captchaSvc *captcha.Service, captchaCfg config.CaptchaConfig) *AuthHandler {
	return &AuthHandler{service: service, cfg: cfg, limiter: limiter, captcha: captchaSvc, captchaCfg: captchaCfg}
}

// Login godoc
// @Summary      用户登录
// @Description  使用用户名和密码进行身份验证，成功返回访问令牌和刷新令牌。支持 IP 和用户名维度的限流保护。
// @Tags         认证
// @Accept       json
// @Produce      json
// @Param        body  body      loginRequest  true  "登录信息"
// @Success      200   {object}  response.Body  "成功，返回 accessToken 和 refreshToken"
// @Failure      400   {object}  contract.ErrorResponse  "参数错误（用户名或密码格式不符）"
// @Failure      401   {object}  contract.ErrorResponse  "认证失败（用户名或密码错误）"
// @Failure      429   {object}  contract.ErrorResponse  "请求过于频繁，请稍后再试"
// @Router       /auth/login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	req, _ := c.MustGet("validated_req").(*loginRequest)
	if !h.allow(c, "login", "ip:"+c.ClientIP(), h.cfg.LoginRateLimit, time.Duration(h.cfg.LoginRateWindowSec)*time.Second) {
		return
	}
	if !h.allow(c, "login", "username:"+normalizeUsername(req.Username), h.cfg.LoginRateLimit, time.Duration(h.cfg.LoginRateWindowSec)*time.Second) {
		return
	}
	if !h.verifyCaptcha(c, req) {
		return
	}
	tokenPair, err := h.service.LoginWithTOTP(c.Request.Context(), req.Username, req.Password, req.TOTPCode)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, tokenPair)
}

// Register godoc
// @Summary      用户注册
// @Description  注册新用户账户。仅当系统配置中 public_registration 为 true 时可用。支持 IP 维度的限流保护。
// @Tags         认证
// @Accept       json
// @Produce      json
// @Param        body  body      loginRequest  true  "注册信息（用户名和密码）"
// @Success      200   {object}  response.Body  "成功，返回用户信息"
// @Failure      400   {object}  contract.ErrorResponse  "参数错误（如用户名已存在）"
// @Failure      429   {object}  contract.ErrorResponse  "请求过于频繁"
// @Router       /auth/register [post]
func (h *AuthHandler) Register(c *gin.Context) {
	req, _ := c.MustGet("validated_req").(*loginRequest)
	if !h.allow(c, "register", "ip:"+c.ClientIP(), h.cfg.RegisterRateLimit, time.Duration(h.cfg.RegisterRateWindowSec)*time.Second) {
		return
	}
	if !h.verifyCaptcha(c, req) {
		return
	}
	user, err := h.service.Register(c.Request.Context(), req.Username, req.Password, req.Email, req.Phone)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, user)
}

// ForgotPassword godoc
// @Summary      发送密码重置验证码
// @Description  向指定邮箱发送 6 位数字验证码。用户不存在也返回成功，防止邮箱枚举探测。
// @Tags         认证
// @Accept       json
// @Produce      json
// @Param        body  body      forgotPasswordRequest  true  "邮箱"
// @Success      200   {object}  response.Body  "成功（无论邮箱是否存在）"
// @Failure      400   {object}  contract.ErrorResponse  "参数错误"
// @Failure      429   {object}  contract.ErrorResponse  "请求过于频繁"
// @Router       /auth/forgot-password [post]
func (h *AuthHandler) ForgotPassword(c *gin.Context) {
	req, _ := c.MustGet("validated_req").(*forgotPasswordRequest)
	if !h.allow(c, "forgot-password", "ip:"+c.ClientIP(), h.cfg.LoginRateLimit, time.Duration(h.cfg.LoginRateWindowSec)*time.Second) {
		return
	}
	if err := h.service.ForgotPassword(c.Request.Context(), req.Email); err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, gin.H{})
}

// ResetPassword godoc
// @Summary      重置密码
// @Description  用邮箱验证码设置新密码。验证码一次性，重置成功后强制登出该用户全部会话。
// @Tags         认证
// @Accept       json
// @Produce      json
// @Param        body  body      resetPasswordRequest  true  "邮箱、验证码、新密码"
// @Success      200   {object}  response.Body  "成功"
// @Failure      400   {object}  contract.ErrorResponse  "参数错误或验证码无效/已过期"
// @Failure      429   {object}  contract.ErrorResponse  "请求过于频繁"
// @Router       /auth/reset-password [post]
func (h *AuthHandler) ResetPassword(c *gin.Context) {
	req, _ := c.MustGet("validated_req").(*resetPasswordRequest)
	if !h.allow(c, "reset-password", "ip:"+c.ClientIP(), h.cfg.LoginRateLimit, time.Duration(h.cfg.LoginRateWindowSec)*time.Second) {
		return
	}
	if err := h.service.ResetPassword(c.Request.Context(), req.Email, req.Code, req.NewPassword); err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, gin.H{})
}

// RefreshToken godoc
// @Summary      刷新访问令牌
// @Description  使用有效的刷新令牌获取新的访问令牌。刷新令牌轮换机制会同时生成新的刷新令牌，旧的刷新令牌失效。
// @Tags         认证
// @Accept       json
// @Produce      json
// @Param        body  body      refreshRequest  true  "刷新令牌"
// @Success      200   {object}  response.Body  "成功，返回新的 token 对"
// @Failure      400   {object}  contract.ErrorResponse  "参数错误"
// @Failure      401   {object}  contract.ErrorResponse  "刷新令牌无效或已过期"
// @Router       /auth/refresh [post]
func (h *AuthHandler) RefreshToken(c *gin.Context) {
	req, _ := c.MustGet("validated_req").(*refreshRequest)
	tokenPair, err := h.service.Refresh(c.Request.Context(), req.RefreshToken)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, tokenPair)
}

// Logout godoc
// @Summary      退出当前会话
// @Description  撤销当前会话的刷新令牌，使其失效。需要有效的访问令牌进行身份验证。
// @Tags         认证
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  response.Body  "成功"
// @Failure      401  {object}  contract.ErrorResponse  "未认证或会话无效"
// @Router       /auth/logout [post]
func (h *AuthHandler) Logout(c *gin.Context) {
	userID, sessionID, ok := authContext(c)
	if !ok {
		response.Fail(c, errors.New(errors.CodeUnauthorized, "invalid session"))
		return
	}
	if err := h.service.Logout(c.Request.Context(), userID, sessionID); err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, gin.H{})
}

// LogoutAll godoc
// @Summary      退出所有会话
// @Description  撤销该用户的所有刷新会话，强制所有设备重新登录。需要有效的访问令牌进行身份验证。
// @Tags         认证
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  response.Body  "成功"
// @Failure      401  {object}  contract.ErrorResponse  "未认证或会话无效"
// @Router       /auth/logout-all [post]
func (h *AuthHandler) LogoutAll(c *gin.Context) {
	userID, _, ok := authContext(c)
	if !ok {
		response.Fail(c, errors.New(errors.CodeUnauthorized, "invalid session"))
		return
	}
	if err := h.service.LogoutAll(c.Request.Context(), userID); err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, gin.H{})
}

// SetupTOTP godoc
// @Summary      生成 TOTP 绑定密钥
// @Description  生成新的 TOTP 密钥并返回 otpauth URI（用于二维码/认证器绑定）。重复调用会轮换密钥。绑定后需调用启用接口用首次验证码确认。
// @Tags         认证
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  response.Body  "成功，返回 secret 与 otpauth URI"
// @Failure      401  {object}  contract.ErrorResponse  "未认证或会话无效"
// @Router       /auth/mfa/setup [post]
func (h *AuthHandler) SetupTOTP(c *gin.Context) {
	userID, _, ok := authContext(c)
	if !ok {
		response.Fail(c, errors.New(errors.CodeUnauthorized, "invalid session"))
		return
	}
	username := c.GetString("username")
	secret, uri, err := h.service.SetupTOTP(c.Request.Context(), userID, username)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, gin.H{"secret": secret, "otpauth_uri": uri})
}

// EnableTOTP godoc
// @Summary      启用 TOTP 二次验证
// @Description  用认证器生成的首次验证码确认启用 TOTP。验证码通过后该用户登录必须提供 TOTP 码。
// @Tags         认证
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body  body      enableTOTPRequest  true  "首次验证码"
// @Success      200  {object}  response.Body  "成功"
// @Failure      400  {object}  contract.ErrorResponse  "参数错误"
// @Failure      401  {object}  contract.ErrorResponse  "未认证或验证码无效"
// @Router       /auth/mfa/enable [post]
func (h *AuthHandler) EnableTOTP(c *gin.Context) {
	req, _ := c.MustGet("validated_req").(*enableTOTPRequest)
	userID, _, ok := authContext(c)
	if !ok {
		response.Fail(c, errors.New(errors.CodeUnauthorized, "invalid session"))
		return
	}
	if err := h.service.EnableTOTP(c.Request.Context(), userID, req.Code); err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, gin.H{})
}

// DisableTOTP godoc
// @Summary      关闭 TOTP 二次验证
// @Description  校验当前验证码后关闭 TOTP 并清除密钥。关闭后登录不再要求 TOTP 码。
// @Tags         认证
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body  body      disableTOTPRequest  true  "当前验证码"
// @Success      200  {object}  response.Body  "成功"
// @Failure      400  {object}  contract.ErrorResponse  "参数错误"
// @Failure      401  {object}  contract.ErrorResponse  "未认证或验证码无效"
// @Router       /auth/mfa/disable [post]
func (h *AuthHandler) DisableTOTP(c *gin.Context) {
	req, _ := c.MustGet("validated_req").(*disableTOTPRequest)
	userID, _, ok := authContext(c)
	if !ok {
		response.Fail(c, errors.New(errors.CodeUnauthorized, "invalid session"))
		return
	}
	if err := h.service.DisableTOTP(c.Request.Context(), userID, req.Code); err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, gin.H{})
}

func authContext(c *gin.Context) (uint64, string, bool) {
	userID, ok := c.Get("user_id")
	if !ok {
		return 0, "", false
	}
	id, ok := userID.(uint64)
	if !ok {
		return 0, "", false
	}
	return id, c.GetString("session_id"), c.GetString("session_id") != ""
}

// verifyCaptcha 校验验证码（启用时）。放在限流检查之后，先限流防刷验证码暴力重试，再校验验证码。
func (h *AuthHandler) verifyCaptcha(c *gin.Context, req *loginRequest) bool {
	if !h.captchaCfg.Enabled || h.captcha == nil {
		return true
	}
	if req.CaptchaID == "" || req.CaptchaCode == "" {
		response.Fail(c, errors.New(errors.CodeCaptchaRequired, "captcha required"))
		return false
	}
	if err := h.captcha.Verify(c.Request.Context(), req.CaptchaID, req.CaptchaCode); err != nil {
		response.Fail(c, errors.New(errors.CodeCaptchaInvalid, "invalid captcha"))
		return false
	}
	return true
}

func (h *AuthHandler) allow(c *gin.Context, scope, key string, limit int, window time.Duration) bool {
	if h.limiter == nil {
		return true
	}
	ok, err := h.limiter.Allow(c.Request.Context(), scope, key, limit, window)
	if err != nil && ok {
		return true
	}
	if err != nil || !ok {
		writeAuthRateLimitHeaders(c, limit, window)
		response.Fail(c, errors.New(errors.CodeRateLimited, "too many requests"))
		return false
	}
	return true
}

// writeAuthRateLimitHeaders 写入认证维度限流响应头
func writeAuthRateLimitHeaders(c *gin.Context, limit int, window time.Duration) {
	c.Header("X-RateLimit-Limit", fmt.Sprintf("%d", limit))
	c.Header("X-RateLimit-Remaining", "0")
	if window > 0 {
		c.Header("Retry-After", fmt.Sprintf("%d", int(window.Seconds())))
	}
}

func normalizeUsername(username string) string {
	return strings.ToLower(strings.TrimSpace(username))
}
