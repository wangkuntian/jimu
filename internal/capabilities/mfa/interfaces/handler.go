package interfaces

import (
	"strconv"

	"jimu/internal/capabilities/mfa/application"
	"jimu/internal/kernel/http/middleware"
	"jimu/internal/shared/errors"
	"jimu/internal/shared/response"

	"github.com/gin-gonic/gin"
)

// DeviceTokenHeader 可信设备令牌请求头：登录时携带可在密码正确的前提下跳过 TOTP
const DeviceTokenHeader = "X-Device-Token"

// enableTOTPRequest 启用 TOTP 请求参数（先用 setup 获取密钥，再用本接口确认）
type enableTOTPRequest struct {
	Code string `json:"code" binding:"required,len=6"`
}

// disableTOTPRequest 关闭 TOTP 请求参数
type disableTOTPRequest struct {
	Code string `json:"code" binding:"required,len=6"`
}

// MFAHandler MFA / 可信设备 HTTP 处理器
type MFAHandler struct {
	service *application.MFAService
}

// NewMFAHandler 创建 MFA 处理器
func NewMFAHandler(service *application.MFAService) *MFAHandler {
	return &MFAHandler{service: service}
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
func (h *MFAHandler) SetupTOTP(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		response.Fail(c, errors.New(errors.CodeUnauthorized, "authentication required"))
		return
	}
	secret, uri, err := h.service.SetupTOTP(c.Request.Context(), userID, c.GetString("username"))
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
func (h *MFAHandler) EnableTOTP(c *gin.Context) {
	req, _ := c.MustGet("validated_req").(*enableTOTPRequest)
	userID, ok := currentUserID(c)
	if !ok {
		response.Fail(c, errors.New(errors.CodeUnauthorized, "authentication required"))
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
func (h *MFAHandler) DisableTOTP(c *gin.Context) {
	req, _ := c.MustGet("validated_req").(*disableTOTPRequest)
	userID, ok := currentUserID(c)
	if !ok {
		response.Fail(c, errors.New(errors.CodeUnauthorized, "authentication required"))
		return
	}
	if err := h.service.DisableTOTP(c.Request.Context(), userID, req.Code); err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, gin.H{})
}

// ListDevices godoc
// @Summary      获取可信设备列表
// @Description  返回当前用户的可信设备（登录时可跳过 TOTP 的设备）。密码始终必需，设备令牌仅替代 TOTP 因子；改密或登出全部设备会吊销全部可信设备。
// @Tags         认证
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  response.Body  "成功，返回可信设备列表"
// @Failure      401  {object}  contract.ErrorResponse  "未认证"
// @Router       /auth/devices [get]
func (h *MFAHandler) ListDevices(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		response.Fail(c, errors.New(errors.CodeUnauthorized, "authentication required"))
		return
	}
	devices, err := h.service.ListTrustedDevices(c.Request.Context(), userID)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, devices)
}

// RevokeDevice godoc
// @Summary      注销可信设备
// @Description  按 ID 注销当前用户的一个可信设备，注销后该设备登录需重新提供 TOTP。
// @Tags         认证
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      int  true  "设备 ID"
// @Success      200  {object}  response.Body  "成功，返回已注销设备 ID"
// @Failure      400  {object}  contract.ErrorResponse  "参数错误（设备 ID 非法）"
// @Failure      401  {object}  contract.ErrorResponse  "未认证"
// @Router       /auth/devices/{id} [delete]
func (h *MFAHandler) RevokeDevice(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		response.Fail(c, errors.New(errors.CodeUnauthorized, "authentication required"))
		return
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.Fail(c, errors.New(errors.CodeInvalidParam, "invalid device id"))
		return
	}
	if err := h.service.RevokeTrustedDevice(c.Request.Context(), userID, id); err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, gin.H{"revoked": id})
}

// RevokeAllDevices godoc
// @Summary      注销全部可信设备
// @Description  注销当前用户全部可信设备，用于设备丢失或异常登录后的止损。
// @Tags         认证
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  response.Body  "成功"
// @Failure      401  {object}  contract.ErrorResponse  "未认证"
// @Router       /auth/devices [delete]
func (h *MFAHandler) RevokeAllDevices(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		response.Fail(c, errors.New(errors.CodeUnauthorized, "authentication required"))
		return
	}
	if err := h.service.RevokeAllTrustedDevices(c.Request.Context(), userID); err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, gin.H{"revoked": "all"})
}

func currentUserID(c *gin.Context) (uint64, bool) {
	v, exists := c.Get("user_id")
	if !exists {
		return 0, false
	}
	id, ok := v.(uint64)
	return id, ok
}

// RegisterMFARoutes 在 /auth 分组上注册 MFA 与可信设备路由（调用方负责套 JWT 中间件）。
func RegisterMFARoutes(authGroup *gin.RouterGroup, service *application.MFAService) {
	handler := NewMFAHandler(service)
	authGroup.POST("/mfa/setup", handler.SetupTOTP)
	authGroup.POST("/mfa/enable", middleware.ValidateJSON(&enableTOTPRequest{}), handler.EnableTOTP)
	authGroup.POST("/mfa/disable", middleware.ValidateJSON(&disableTOTPRequest{}), handler.DisableTOTP)
	authGroup.GET("/devices", handler.ListDevices)
	authGroup.DELETE("/devices", handler.RevokeAllDevices)
	authGroup.DELETE("/devices/:id", handler.RevokeDevice)
}
