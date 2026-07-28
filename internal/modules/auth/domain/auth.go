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
	Register(ctx context.Context, username, password string) (*domain.User, error)
	RefreshToken(ctx context.Context, refreshToken string) (*TokenPair, error)
}
