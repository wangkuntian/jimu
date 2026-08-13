package domain

import (
	"context"

	"jimu/internal/modules/user/domain"
)

type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
}

type AuthServiceInterface interface {
	Login(ctx context.Context, username, password string) (*TokenPair, error)
	Register(ctx context.Context, username, password, email, phone string) (*domain.User, error)
	Refresh(ctx context.Context, refreshToken string) (*TokenPair, error)
	Logout(ctx context.Context, userID uint64, sessionID string) error
	LogoutAll(ctx context.Context, userID uint64) error
}
