package interfaces

// loginRequest 登录/注册请求
type loginRequest struct {
	Username string `json:"username" binding:"required,min=4,max=20"`
	Password string `json:"password" binding:"required,min=8,max=32"`
}

// refreshRequest 刷新 Token 请求
type refreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}
