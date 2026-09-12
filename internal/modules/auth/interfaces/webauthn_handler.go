package interfaces

import (
	"strconv"
	"time"

	"jimu/internal/modules/auth/application"
	"jimu/internal/shared/errors"
	"jimu/internal/shared/response"

	"github.com/gin-gonic/gin"
)

// webAuthnBeginRequest 开始注册时可选填写凭证名称（便于用户识别设备）
type webAuthnBeginRequest struct {
	Name string `json:"name" binding:"omitempty,max=64"`
}

// webAuthnLoginBeginRequest 无密码登录开始请求
type webAuthnLoginBeginRequest struct {
	Username string `json:"username" binding:"required,min=3,max=64"`
}

// webAuthnRenameRequest 重命名通行密钥
type webAuthnRenameRequest struct {
	Name string `json:"name" binding:"required,min=1,max=64"`
}

// webAuthnSessionResponse 挑战会话响应：options 交给浏览器 navigator.credentials，
// session_id 由客户端在 finish 步骤原样回传（服务端一次性存储，防挑战替换/重放）。
type webAuthnSessionResponse struct {
	SessionID string      `json:"session_id"`
	Options   interface{} `json:"options"`
}

// BeginWebAuthnRegistration godoc
// @Summary      开始注册通行密钥
// @Description  为当前登录用户生成 WebAuthn 注册选项（publicKey）与一次性 session_id。客户端调用 navigator.credentials.create() 后，把返回的凭证 JSON 原样提交到 finish 接口。需要浏览器与 HTTPS（localhost 除外）。
// @Tags         认证
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body  body      webAuthnBeginRequest  false  "凭证名称（可选，便于识别设备）"
// @Success      200  {object}  response.Body  "成功，返回 session_id 与注册选项"
// @Failure      401  {object}  contract.ErrorResponse  "未认证"
// @Failure      500  {object}  contract.ErrorResponse  "WebAuthn 未配置"
// @Router       /auth/webauthn/register/begin [post]
func (h *AuthHandler) BeginWebAuthnRegistration(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		response.Fail(c, errors.New(errors.CodeUnauthorized, "authentication required"))
		return
	}
	var req webAuthnBeginRequest
	if c.Request.ContentLength > 0 {
		if err := c.ShouldBindJSON(&req); err != nil {
			response.Fail(c, errors.New(errors.CodeInvalidParam, "invalid request body"))
			return
		}
	}
	creation, sessionID, err := h.service.BeginWebAuthnRegistration(c.Request.Context(), userID, req.Name)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, webAuthnSessionResponse{SessionID: sessionID, Options: creation})
}

// FinishWebAuthnRegistration godoc
// @Summary      完成注册通行密钥
// @Description  校验浏览器返回的注册凭证（证明与挑战），通过后保存公钥。session_id 取自 begin 响应；请求体为 navigator.credentials.create() 的原始 JSON。
// @Tags         认证
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        session_id  query  string  true  "begin 接口返回的 session_id"
// @Success      200  {object}  response.Body  "成功，返回已注册凭证信息"
// @Failure      400  {object}  contract.ErrorResponse  "凭证校验失败（2011）"
// @Failure      401  {object}  contract.ErrorResponse  "未认证"
// @Router       /auth/webauthn/register/finish [post]
func (h *AuthHandler) FinishWebAuthnRegistration(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		response.Fail(c, errors.New(errors.CodeUnauthorized, "authentication required"))
		return
	}
	body, err := c.GetRawData()
	if err != nil {
		response.Fail(c, errors.New(errors.CodeInvalidParam, "failed to read request body"))
		return
	}
	info, err := h.service.FinishWebAuthnRegistration(c.Request.Context(), userID, c.Query("session_id"), body)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, info)
}

// ListWebAuthnCredentials godoc
// @Summary      获取通行密钥列表
// @Description  返回当前用户已注册的通行密钥（不含公钥等敏感字段），用于设备管理与注销。
// @Tags         认证
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  response.Body  "成功，返回通行密钥列表"
// @Failure      401  {object}  contract.ErrorResponse  "未认证"
// @Router       /auth/webauthn/credentials [get]
func (h *AuthHandler) ListWebAuthnCredentials(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		response.Fail(c, errors.New(errors.CodeUnauthorized, "authentication required"))
		return
	}
	credentials, err := h.service.ListWebAuthnCredentials(c.Request.Context(), userID)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, credentials)
}

// RenameWebAuthnCredential godoc
// @Summary      重命名通行密钥
// @Description  修改当前用户某个通行密钥的名称。
// @Tags         认证
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id    path      int                    true  "凭证 ID"
// @Param        body  body      webAuthnRenameRequest  true  "新名称"
// @Success      200  {object}  response.Body  "更新成功"
// @Failure      400  {object}  contract.ErrorResponse  "参数错误"
// @Failure      401  {object}  contract.ErrorResponse  "未认证"
// @Router       /auth/webauthn/credentials/{id} [put]
func (h *AuthHandler) RenameWebAuthnCredential(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		response.Fail(c, errors.New(errors.CodeUnauthorized, "authentication required"))
		return
	}
	id, err := parseCredentialID(c)
	if err != nil {
		response.Fail(c, err)
		return
	}
	var req webAuthnRenameRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, errors.New(errors.CodeInvalidParam, "invalid request body"))
		return
	}
	if err := h.service.RenameWebAuthnCredential(c.Request.Context(), userID, id, req.Name); err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, nil)
}

// DeleteWebAuthnCredential godoc
// @Summary      删除通行密钥
// @Description  删除当前用户的某个通行密钥；删除最后一个凭证后该账号不能再使用通行密钥登录（密码登录不受影响）。
// @Tags         认证
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      int  true  "凭证 ID"
// @Success      200  {object}  response.Body  "删除成功"
// @Failure      401  {object}  contract.ErrorResponse  "未认证"
// @Router       /auth/webauthn/credentials/{id} [delete]
func (h *AuthHandler) DeleteWebAuthnCredential(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		response.Fail(c, errors.New(errors.CodeUnauthorized, "authentication required"))
		return
	}
	id, err := parseCredentialID(c)
	if err != nil {
		response.Fail(c, err)
		return
	}
	if err := h.service.DeleteWebAuthnCredential(c.Request.Context(), userID, id); err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, nil)
}

// BeginWebAuthnLogin godoc
// @Summary      开始通行密钥登录
// @Description  按用户名生成 WebAuthn 登录断言（allowCredentials 为该用户已注册凭证）与一次性 session_id。用户不存在或未注册凭证时返回错误，不泄漏账号是否存在之外的信息。支持 IP 与用户名维度限流。
// @Tags         认证
// @Accept       json
// @Produce      json
// @Param        body  body      webAuthnLoginBeginRequest  true  "用户名"
// @Success      200  {object}  response.Body  "成功，返回 session_id 与登录选项"
// @Failure      401  {object}  contract.ErrorResponse  "认证失败（用户不存在或已禁用）"
// @Failure      404  {object}  contract.ErrorResponse  "该用户未注册通行密钥（2010）"
// @Failure      429  {object}  contract.ErrorResponse  "请求过于频繁"
// @Router       /auth/webauthn/login/begin [post]
func (h *AuthHandler) BeginWebAuthnLogin(c *gin.Context) {
	req, _ := c.MustGet("validated_req").(*webAuthnLoginBeginRequest)
	if !h.allow(c, "login", "ip:"+c.ClientIP(), h.cfg.LoginRateLimit, time.Duration(h.cfg.LoginRateWindowSec)*time.Second) {
		return
	}
	if !h.allow(c, "login", "username:"+normalizeUsername(req.Username), h.cfg.LoginRateLimit, time.Duration(h.cfg.LoginRateWindowSec)*time.Second) {
		return
	}
	ctx := application.WithClientInfo(c.Request.Context(), c.ClientIP(), c.Request.UserAgent())
	assertion, sessionID, err := h.service.BeginWebAuthnLogin(ctx, req.Username)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, webAuthnSessionResponse{SessionID: sessionID, Options: assertion})
}

// FinishWebAuthnLogin godoc
// @Summary      完成通行密钥登录
// @Description  校验浏览器返回的登录断言（签名、挑战、计数器），通过后签发 access/refresh token。通行密钥是抗钓鱼的强因子，因此不再要求密码，也不叠加 TOTP。session_id 取自 begin 响应；请求体为 navigator.credentials.get() 的原始 JSON。
// @Tags         认证
// @Accept       json
// @Produce      json
// @Param        session_id  query  string  true  "begin 接口返回的 session_id"
// @Success      200  {object}  response.Body  "成功，返回 accessToken 和 refreshToken"
// @Failure      400  {object}  contract.ErrorResponse  "断言校验失败（2011）"
// @Failure      429  {object}  contract.ErrorResponse  "请求过于频繁"
// @Router       /auth/webauthn/login/finish [post]
func (h *AuthHandler) FinishWebAuthnLogin(c *gin.Context) {
	sessionID := c.Query("session_id")
	if !h.allow(c, "login", "ip:"+c.ClientIP(), h.cfg.LoginRateLimit, time.Duration(h.cfg.LoginRateWindowSec)*time.Second) {
		return
	}
	body, err := c.GetRawData()
	if err != nil {
		response.Fail(c, errors.New(errors.CodeInvalidParam, "failed to read request body"))
		return
	}
	ctx := application.WithClientInfo(c.Request.Context(), c.ClientIP(), c.Request.UserAgent())
	pair, err := h.service.FinishWebAuthnLogin(ctx, sessionID, body)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, pair)
}

func parseCredentialID(c *gin.Context) (uint64, error) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		return 0, errors.New(errors.CodeInvalidParam, "invalid credential id")
	}
	return id, nil
}
