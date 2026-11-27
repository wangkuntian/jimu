package errors

import (
	"fmt"
	"time"
)

// 能力错误码定义（2xxx-5xxx）。
// 0 能力生成项目不会复制本文件，核心错误码仍可独立编译。
const (
	// 用户/认证模块 (2xxx)
	CodeUserNotFound               = 2001 // 用户不存在
	CodeUserExists                 = 2002 // 用户已存在
	CodeInvalidPassword            = 2003 // 密码错误
	CodeRoleNotFound               = 2004 // 角色不存在
	CodeInvalidResetCode           = 2005 // 密码重置验证码无效或已过期
	CodeMFARequired                = 2006 // 需要提供 TOTP 二次验证码
	CodeInvalidMFA                 = 2007 // TOTP 验证码无效
	CodePasswordReused             = 2008 // 新密码与当前或近期使用过的密码重复
	CodePasswordBreached           = 2009 // 新密码出现在已知数据泄露集合中
	CodeWebAuthnNoCredential       = 2010 // 该用户未注册通行密钥
	CodeWebAuthnVerificationFailed = 2011 // 通行密钥校验失败（挑战过期/签名无效）

	// OAuth 模块 (3xxx)
	CodeOAuthProviderNotFound = 3001 // 第三方登录提供商不存在

	// 验证码模块 (4xxx)
	CodeCaptchaRequired = 4001 // 缺少验证码
	CodeCaptchaInvalid  = 4002 // 验证码无效

	// 租户模块 (5xxx)
	CodeTenantNotFound   = 5001 // 租户不存在
	CodeTenantExists     = 5002 // 租户已存在
	CodeTenantProtected  = 5003 // 默认租户受保护，不可停用或删除
	CodeTenantCodeFormat = 5004 // 租户编码格式无效
	CodeQuotaExceeded    = 5005 // 租户资源配额已用尽
	CodePlanNotFound     = 5006 // 套餐不存在
)

// AccountLocked 返回账号锁定错误，message 包含剩余锁定时间（分钟）。
func AccountLocked(remaining time.Duration) *AppError {
	minutes := int(remaining.Minutes())
	if minutes < 1 {
		minutes = 1
	}
	return New(CodeForbidden, fmt.Sprintf("account locked due to too many failed attempts, try again in %d minutes", minutes))
}

func init() {
	statuses := map[int]int{
		CodeUserNotFound:               404,
		CodeUserExists:                 409,
		CodeInvalidPassword:            401,
		CodeRoleNotFound:               404,
		CodeInvalidResetCode:           400,
		CodeMFARequired:                401,
		CodeInvalidMFA:                 401,
		CodePasswordReused:             400,
		CodePasswordBreached:           400,
		CodeWebAuthnNoCredential:       404,
		CodeWebAuthnVerificationFailed: 400,
		CodeOAuthProviderNotFound:      404,
		CodeCaptchaRequired:            400,
		CodeCaptchaInvalid:             400,
		CodeTenantNotFound:             404,
		CodeTenantExists:               409,
		CodeTenantProtected:            409,
		CodeTenantCodeFormat:           400,
		CodeQuotaExceeded:              403,
		CodePlanNotFound:               404,
	}
	for code, status := range statuses {
		capabilityHTTPStatus[code] = status
	}
	capabilityErrorCodes = []ErrorInfo{
		{CodeUserNotFound, "用户不存在", 404, "用户"},
		{CodeUserExists, "用户已存在", 409, "用户"},
		{CodeInvalidPassword, "密码错误", 401, "用户"},
		{CodeRoleNotFound, "角色不存在", 404, "角色"},
		{CodeInvalidResetCode, "密码重置验证码无效或已过期", 400, "用户"},
		{CodeMFARequired, "需要提供 TOTP 二次验证码", 401, "用户"},
		{CodeInvalidMFA, "TOTP 验证码无效", 401, "用户"},
		{CodePasswordReused, "新密码与当前或近期使用过的密码重复", 400, "用户"},
		{CodePasswordBreached, "新密码出现在已知数据泄露集合中", 400, "用户"},
		{CodeOAuthProviderNotFound, "第三方登录提供商不存在", 404, "OAuth"},
		{CodeCaptchaRequired, "缺少验证码", 400, "验证码"},
		{CodeCaptchaInvalid, "验证码无效", 400, "验证码"},
		{CodeTenantNotFound, "租户不存在", 404, "租户"},
		{CodeTenantExists, "租户已存在", 409, "租户"},
		{CodeTenantProtected, "默认租户受保护，不可停用或删除", 409, "租户"},
		{CodeTenantCodeFormat, "租户编码格式无效", 400, "租户"},
		{CodeQuotaExceeded, "租户资源配额已用尽", 403, "租户"},
		{CodePlanNotFound, "套餐不存在", 404, "租户"},
		{CodeWebAuthnNoCredential, "该用户未注册通行密钥", 404, "用户"},
		{CodeWebAuthnVerificationFailed, "通行密钥校验失败", 400, "用户"},
	}
}
