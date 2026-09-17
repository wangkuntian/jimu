package domain

import (
	"context"

	"jimu/internal/modules/user/domain"
)

type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	// DeviceToken 仅在「记住此设备」成功签发时返回，客户端需安全保存；
	// 后续登录携带它可在密码正确的前提下跳过 TOTP。
	DeviceToken string `json:"device_token,omitempty"`
}

type AuthServiceInterface interface {
	Login(ctx context.Context, username, password string) (*TokenPair, error)
	LoginWithTOTP(ctx context.Context, username, password, totpCode string) (*TokenPair, error)
	Register(ctx context.Context, username, password, email, phone string) (*domain.User, error)
	Refresh(ctx context.Context, refreshToken string) (*TokenPair, error)
	Logout(ctx context.Context, userID uint64, sessionID string) error
	LogoutAll(ctx context.Context, userID uint64) error
	SetupTOTP(ctx context.Context, userID uint64, account string) (secret string, uri string, err error)
	EnableTOTP(ctx context.Context, userID uint64, code string) error
	DisableTOTP(ctx context.Context, userID uint64, code string) error
}
