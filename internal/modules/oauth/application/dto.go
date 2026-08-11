// internal/modules/oauth/application/dto.go
package application

// LoginRequest 回调登录请求
type LoginRequest struct {
	Code  string `json:"code"`
	State string `json:"state"`
}
