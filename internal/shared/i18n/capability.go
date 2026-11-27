package i18n

var capabilityMessages = map[string]map[string]string{
	LangZH: {
		"role_not_found":               "角色不存在",
		"invalid_reset_code":           "重置验证码无效或已过期",
		"mfa_required":                 "需要提供二次验证码",
		"invalid_mfa":                  "二次验证码无效",
		"password_breached":            "该密码已在数据泄露中出现，请更换",
		"webauthn_no_credential":       "该账号未注册通行密钥",
		"webauthn_verification_failed": "通行密钥校验失败，请重试",
		"oauth_provider_not_found":     "第三方登录提供商不存在",
		"captcha_required":             "缺少验证码",
		"captcha_invalid":              "验证码无效",
		"tenant_not_found":             "租户不存在",
		"tenant_exists":                "租户编码已存在",
		"tenant_protected":             "默认租户受保护，不可删除",
		"tenant_code_format":           "租户编码格式无效",
		"quota_exceeded":               "资源配额已用尽，请联系管理员升级套餐",
		"plan_not_found":               "套餐不存在",
		"user_not_found":               "用户不存在",
		"user_exists":                  "用户名已存在",
		"account_locked":               "账号已被锁定，请 %d 分钟后重试",
		"password_reused":              "新密码不能与当前或近期使用过的密码相同",
	},
	LangEN: {
		"role_not_found":               "role not found",
		"invalid_reset_code":           "reset code is invalid or expired",
		"mfa_required":                 "second factor code is required",
		"invalid_mfa":                  "invalid second factor code",
		"password_breached":            "this password has appeared in a data breach, choose another one",
		"webauthn_no_credential":       "no passkey registered for this account",
		"webauthn_verification_failed": "passkey verification failed, please retry",
		"oauth_provider_not_found":     "oauth provider not found",
		"captcha_required":             "captcha is required",
		"captcha_invalid":              "captcha is invalid",
		"tenant_not_found":             "tenant not found",
		"tenant_exists":                "tenant code already exists",
		"tenant_protected":             "the default tenant is protected",
		"tenant_code_format":           "invalid tenant code format",
		"quota_exceeded":               "resource quota exceeded, ask an administrator to upgrade the plan",
		"plan_not_found":               "plan not found",
		"user_not_found":               "user not found",
		"user_exists":                  "username already exists",
		"account_locked":               "account locked, try again in %d minutes",
		"password_reused":              "new password must differ from current or recent passwords",
	},
}

func init() {
	for lang, entries := range capabilityMessages {
		RegisterMessages(lang, entries)
	}
}
