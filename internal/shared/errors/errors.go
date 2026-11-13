package errors

import (
	"fmt"
)

// 错误码定义
// 编码规则：
//
//	1xxx - 通用错误
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
//	1010 -> 503 Service Unavailable
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
	CodeServiceUnavailable = 1010 // 服务繁忙（负载保护拒绝）
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

var (
	capabilityHTTPStatus = map[int]int{}
	capabilityErrorCodes []ErrorInfo
)

// HTTPStatus 返回错误码对应的 HTTP 状态码
func HTTPStatus(code int) int {
	switch code {
	case CodeOK:
		return 200
	case CodeInvalidParam:
		return 400
	case CodeUnauthorized, CodeInvalidCredentials:
		return 401
	case CodeForbidden:
		return 403
	case CodeNotFound:
		return 404
	case CodeConflict:
		return 409
	case CodeRateLimited:
		return 429
	case CodeTimeout:
		return 504
	case CodeServiceUnavailable:
		return 503
	case CodeInternalError:
		return 500
	}
	if status, ok := capabilityHTTPStatus[code]; ok {
		return status
	}
	return 500
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
	infos := []ErrorInfo{
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
		{CodeServiceUnavailable, "服务繁忙", 503, "通用"},
	}
	return append(infos, capabilityErrorCodes...)
}
