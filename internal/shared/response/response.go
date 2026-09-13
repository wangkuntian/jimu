package response

import (
	"context"
	"errors"
	"net/http"

	appErrs "jimu/internal/shared/errors"
	"jimu/internal/shared/i18n"

	"github.com/gin-gonic/gin"
)

type Body struct {
	Code      int         `json:"code"`
	Message   string      `json:"message"`
	Data      interface{} `json:"data,omitempty"`
	RequestID string      `json:"request_id,omitempty"`
	Details   interface{} `json:"details,omitempty"`
}

type Paginated struct {
	Code      int         `json:"code"`
	Message   string      `json:"message"`
	Data      interface{} `json:"data"`
	Total     int64       `json:"total"`
	Page      int         `json:"page"`
	PageSize  int         `json:"page_size"`
	RequestID string      `json:"request_id,omitempty"`
}

func OK(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Body{
		Code:      0,
		Message:   "ok",
		Data:      data,
		RequestID: requestID(c),
	})
}

func Created(c *gin.Context, data interface{}) {
	c.JSON(http.StatusCreated, Body{
		Code:      0,
		Message:   "ok",
		Data:      data,
		RequestID: requestID(c),
	})
}

func NoContent(c *gin.Context) {
	c.Status(http.StatusNoContent)
}

func Fail(c *gin.Context, err error) {
	failWithDetails(c, err, nil)
}

// FailWithDetails 返回带字段级错误详情的失败响应
func FailWithDetails(c *gin.Context, err error, details interface{}) {
	failWithDetails(c, err, details)
}

// codeToKey 错误码 → i18n key（新增错误码需在此登记，i18n 测试会强制校验）
var codeToKey = map[int]string{
	appErrs.CodeInvalidParam:               "invalid_param",
	appErrs.CodeUnauthorized:               "unauthorized",
	appErrs.CodeForbidden:                  "forbidden",
	appErrs.CodeNotFound:                   "not_found",
	appErrs.CodeInternalError:              "internal_error",
	appErrs.CodeInvalidCredentials:         "invalid_credentials",
	appErrs.CodeRateLimited:                "rate_limit_exceeded",
	appErrs.CodeTimeout:                    "timeout",
	appErrs.CodeConflict:                   "conflict",
	appErrs.CodeServiceUnavailable:         "service_unavailable",
	appErrs.CodeUserNotFound:               "user_not_found",
	appErrs.CodeUserExists:                 "user_exists",
	appErrs.CodeInvalidPassword:            "invalid_credentials",
	appErrs.CodeRoleNotFound:               "role_not_found",
	appErrs.CodeInvalidResetCode:           "invalid_reset_code",
	appErrs.CodeMFARequired:                "mfa_required",
	appErrs.CodeInvalidMFA:                 "invalid_mfa",
	appErrs.CodePasswordReused:             "password_reused",
	appErrs.CodePasswordBreached:           "password_breached",
	appErrs.CodeWebAuthnNoCredential:       "webauthn_no_credential",
	appErrs.CodeWebAuthnVerificationFailed: "webauthn_verification_failed",
	appErrs.CodeOAuthProviderNotFound:      "oauth_provider_not_found",
	appErrs.CodeCaptchaRequired:            "captcha_required",
	appErrs.CodeCaptchaInvalid:             "captcha_invalid",
	appErrs.CodeTenantNotFound:             "tenant_not_found",
	appErrs.CodeTenantExists:               "tenant_exists",
	appErrs.CodeTenantProtected:            "tenant_protected",
	appErrs.CodeTenantCodeFormat:           "tenant_code_format",
	appErrs.CodeQuotaExceeded:              "quota_exceeded",
}

// localeFrom 从 gin context 读取语言，缺省 zh
func localeFrom(c *gin.Context) string {
	if c != nil {
		if l := c.GetString("locale"); l != "" {
			return l
		}
	}
	return i18n.LangZH
}

func failWithDetails(c *gin.Context, err error, details interface{}) {
	var appErr *appErrs.AppError
	if errors.As(err, &appErr) {
		// 上下文超时（含被包装成内部错误的情况）统一按 1008/504 返回，避免误报 500
		if errors.Is(appErr, context.DeadlineExceeded) && appErr.Code != appErrs.CodeTimeout {
			appErr = appErrs.New(appErrs.CodeTimeout, appErr.Message)
		}
		// 内部错误隐藏具体原因，只返回翻译后的通用消息；
		// 已登记的错误码返回本地化文案；未登记的错误码保留调用方消息（避免出现
		// 「HTTP 403 + 服务器内部错误」这类状态与文案矛盾的响应）。
		locale := localeFrom(c)
		message := appErr.Message
		if key, ok := codeToKey[appErr.Code]; ok {
			message = i18n.T(key, locale)
		} else if appErr.Code == appErrs.CodeInternalError || message == "" {
			message = i18n.T("internal_error", locale)
		}
		c.JSON(StatusForCode(appErr.Code), Body{
			Code:      appErr.Code,
			Message:   message,
			RequestID: requestID(c),
			Details:   details,
		})
		return
	}
	if errors.Is(err, context.DeadlineExceeded) {
		c.JSON(StatusForCode(appErrs.CodeTimeout), Body{
			Code:      appErrs.CodeTimeout,
			Message:   i18n.T("timeout", localeFrom(c)),
			RequestID: requestID(c),
			Details:   details,
		})
		return
	}
	c.JSON(http.StatusInternalServerError, Body{
		Code:      appErrs.CodeInternalError,
		Message:   i18n.T("internal_error", localeFrom(c)),
		RequestID: requestID(c),
		Details:   details,
	})
}

func Page(c *gin.Context, data interface{}, total int64, page, pageSize int) {
	c.JSON(http.StatusOK, Paginated{
		Code:      0,
		Message:   "ok",
		Data:      data,
		Total:     total,
		Page:      page,
		PageSize:  pageSize,
		RequestID: requestID(c),
	})
}

// StatusForCode 返回错误码对应的 HTTP 状态码
func StatusForCode(code int) int {
	return appErrs.HTTPStatus(code)
}

func requestID(c *gin.Context) string {
	if c == nil {
		return ""
	}
	return c.GetString("request_id")
}
