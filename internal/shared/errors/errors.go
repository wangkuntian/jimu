package errors

import "fmt"

// 错误码定义
// 编码规则：
//
//	1xxx - 通用错误
//	2xxx - 用户/认证模块
//	3xxx - OAuth 模块
//	4xxx - 验证码模块
//	9xxx - 系统级错误
//
// HTTP 映射：
//
//	1001 -> 400 Bad Request
//	1002 -> 401 Unauthorized
//	1003 -> 403 Forbidden
//	1004 -> 404 Not Found
//	1005 -> 500 Internal Server Error
//	1006 -> 401 Unauthorized
//	1007 -> 429 Too Many Requests
//	1008 -> 504 Gateway Timeout
//	1009 -> 409 Conflict
//	2001 -> 404 Not Found
//	2002 -> 409 Conflict
//	2003 -> 401 Unauthorized
//	2004 -> 404 Not Found
//	4001 -> 400 Bad Request
//	4002 -> 400 Bad Request
const (
	// 通用错误 (1xxx)
	CodeOK                 = 0    // 成功
	CodeInvalidParam       = 1001 // 参数错误
	CodeUnauthorized       = 1002 // 未认证
	CodeForbidden          = 1003 // 无权限
	CodeNotFound           = 1004 // 资源不存在
	CodeInternalError      = 1005 // 服务器内部错误
	CodeInvalidCredentials = 1006 // 认证信息无效
	CodeRateLimited        = 1007 // 请求过于频繁
	CodeTimeout            = 1008 // 请求超时
	CodeConflict           = 1009 // 资源冲突

	// 用户/认证模块 (2xxx)
	CodeUserNotFound    = 2001 // 用户不存在
	CodeUserExists      = 2002 // 用户已存在
	CodeInvalidPassword = 2003 // 密码错误
	CodeRoleNotFound    = 2004 // 角色不存在

	// OAuth 模块 (3xxx)
	CodeOAuthProviderNotFound = 3001 // 第三方登录提供商不存在

	// 验证码模块 (4xxx)
	CodeCaptchaRequired = 4001 // 缺少验证码
	CodeCaptchaInvalid  = 4002 // 验证码无效
)

type AppError struct {
	Code    int
	Message string
	Cause   error
}

func (e *AppError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("[%d] %s: %v", e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("[%d] %s", e.Code, e.Message)
}

func (e *AppError) Unwrap() error {
	return e.Cause
}

func New(code int, message string) *AppError {
	return &AppError{Code: code, Message: message}
}

func Wrap(code int, message string, cause error) *AppError {
	return &AppError{Code: code, Message: message, Cause: cause}
}

// HTTPStatus 返回错误码对应的 HTTP 状态码
func HTTPStatus(code int) int {
	switch code {
	case CodeOK:
		return 200
	case CodeInvalidParam:
		return 400
	case CodeUnauthorized, CodeInvalidCredentials, CodeInvalidPassword:
		return 401
	case CodeForbidden:
		return 403
	case CodeNotFound, CodeUserNotFound, CodeRoleNotFound, CodeOAuthProviderNotFound:
		return 404
	case CodeConflict, CodeUserExists:
		return 409
	case CodeCaptchaRequired, CodeCaptchaInvalid:
		return 400
	case CodeRateLimited:
		return 429
	case CodeTimeout:
		return 504
	case CodeInternalError:
		return 500
	default:
		return 500
	}
}

// IsCode 判断错误码是否匹配
func IsCode(err error, code int) bool {
	var appErr *AppError
	if As(err, &appErr) {
		return appErr.Code == code
	}
	return false
}

// As 类型断言
func As(err error, target **AppError) bool {
	for err != nil {
		if e, ok := err.(*AppError); ok {
			*target = e
			return true
		}
		type unwrapper interface{ Unwrap() error }
		if u, ok := err.(unwrapper); ok {
			err = u.Unwrap()
		} else {
			return false
		}
	}
	return false
}

// ErrorInfo 错误码信息（用于文档生成）
type ErrorInfo struct {
	Code       int    `json:"code"`
	Message    string `json:"message"`
	HTTPStatus int    `json:"http_status"`
	Category   string `json:"category"`
}

// AllErrorCodes 返回所有错误码信息（用于 Swagger 文档）
func AllErrorCodes() []ErrorInfo {
	return []ErrorInfo{
		{CodeOK, "成功", 200, "通用"},
		{CodeInvalidParam, "参数错误", 400, "通用"},
		{CodeUnauthorized, "未认证", 401, "通用"},
		{CodeForbidden, "无权限", 403, "通用"},
		{CodeNotFound, "资源不存在", 404, "通用"},
		{CodeInternalError, "服务器内部错误", 500, "通用"},
		{CodeInvalidCredentials, "认证信息无效", 401, "通用"},
		{CodeRateLimited, "请求过于频繁", 429, "通用"},
		{CodeTimeout, "请求超时", 504, "通用"},
		{CodeConflict, "资源冲突", 409, "通用"},
		{CodeUserNotFound, "用户不存在", 404, "用户"},
		{CodeUserExists, "用户已存在", 409, "用户"},
		{CodeInvalidPassword, "密码错误", 401, "用户"},
		{CodeRoleNotFound, "角色不存在", 404, "角色"},
		{CodeOAuthProviderNotFound, "第三方登录提供商不存在", 404, "OAuth"},
		{CodeCaptchaRequired, "缺少验证码", 400, "验证码"},
		{CodeCaptchaInvalid, "验证码无效", 400, "验证码"},
	}
}
