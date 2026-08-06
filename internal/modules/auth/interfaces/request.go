package interfaces

// loginRequest 登录/注册请求参数
type loginRequest struct {
	// 用户名（4-20 位字母、数字或下划线）
	Username string `json:"username" binding:"required,min=4,max=20"`
	// 密码（8-32 位，必须包含字母和数字）
	Password string `json:"password" binding:"required,min=8,max=32"`
}

// refreshRequest 刷新 Token 请求参数
type refreshRequest struct {
	// 刷新令牌（从登录或刷新接口获取）
	RefreshToken string `json:"refresh_token" binding:"required"`
}
