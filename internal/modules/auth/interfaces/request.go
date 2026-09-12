package interfaces

// loginRequest 登录/注册请求参数
type loginRequest struct {
	// 用户名（4-20 位字母、数字或下划线）
	Username string `json:"username" binding:"required,min=4,max=20"`
	// 密码（8-32 位，必须包含字母和数字）
	Password string `json:"password" binding:"required,min=8,max=32"`
	// 邮箱（注册时可选收集，用于密码重置与通知；登录忽略）
	Email string `json:"email,omitempty" binding:"omitempty,email"`
	// 手机号（注册时可选收集；登录忽略）
	Phone string `json:"phone,omitempty" binding:"omitempty"`
	// 验证码（登录时可选，注册时如果开启验证码则必填）
	CaptchaID   string `json:"captcha_id,omitempty"`
	CaptchaCode string `json:"captcha_code,omitempty"`
	// 记住此设备：登录成功后签发设备令牌，后续登录携带 X-Device-Token 可跳过 TOTP（密码仍必需）
	RememberDevice bool `json:"remember_device,omitempty"`
	// TOTP 二次验证码（用户启用 TOTP 后登录必填，6 位数字）
	TOTPCode string `json:"totp_code,omitempty" binding:"omitempty,len=6"`
	// 租户名称（开通式注册时必填：注册即开通新租户，注册者成为 owner）
	TenantName string `json:"tenant_name,omitempty" binding:"omitempty,min=1,max=128"`
	// 租户编码（开通式注册时可选；不传自动生成，仅限字母/数字/短横线/下划线）
	TenantCode string `json:"tenant_code,omitempty" binding:"omitempty,max=64"`
}

// enableTOTPRequest 启用 TOTP 请求参数（先用 SetupTOTP 获取密钥，再用本接口确认）
type enableTOTPRequest struct {
	Code string `json:"code" binding:"required,len=6"`
}

// disableTOTPRequest 关闭 TOTP 请求参数
type disableTOTPRequest struct {
	Code string `json:"code" binding:"required,len=6"`
}

// refreshRequest 刷新 Token 请求参数
type refreshRequest struct {
	// 刷新令牌（从登录或刷新接口获取）
	RefreshToken string `json:"refresh_token" binding:"required"`
}

// forgotPasswordRequest 忘记密码请求参数
type forgotPasswordRequest struct {
	Email string `json:"email" binding:"required,email"`
}

// resetPasswordRequest 重置密码请求参数
type resetPasswordRequest struct {
	Email       string `json:"email" binding:"required,email"`
	Code        string `json:"code" binding:"required,len=6"`
	NewPassword string `json:"new_password" binding:"required,min=8,max=32"`
}
