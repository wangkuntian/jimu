// internal/modules/oauth/application/service.go
package application

import (
	"context"
	"fmt"
	"time"

	"jimu/internal/modules/auth/domain"
	oauthdomain "jimu/internal/modules/oauth/domain"
	userdomain "jimu/internal/modules/user/domain"
	"jimu/internal/platform/auth"
	oauthplatform "jimu/internal/platform/oauth"
	"jimu/internal/shared/errors"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// OAuthService OAuth 登录服务
type OAuthService struct {
	userRepo    userdomain.UserRepository
	bindingRepo oauthdomain.BindingRepository
	jwtUtil     *auth.JWT
	sessions    auth.SessionStore
	providers   map[string]oauthplatform.Provider
	accessMin   int
}

// NewOAuthService 创建 OAuth 服务
func NewOAuthService(userRepo userdomain.UserRepository, bindingRepo oauthdomain.BindingRepository, jwtUtil *auth.JWT, sessions auth.SessionStore, providers map[string]oauthplatform.Provider, accessMin int) *OAuthService {
	return &OAuthService{
		userRepo:    userRepo,
		bindingRepo: bindingRepo,
		jwtUtil:     jwtUtil,
		sessions:    sessions,
		providers:   providers,
		accessMin:   accessMin,
	}
}

// AuthURL 构造授权跳转 URL
func (s *OAuthService) AuthURL(providerName, state string) (string, error) {
	p, ok := s.providers[providerName]
	if !ok {
		return "", errors.New(errors.CodeOAuthProviderNotFound, "unknown oauth provider: "+providerName)
	}
	return p.AuthURL(state), nil
}

// Login 处理 OAuth 回调，匹配/创建用户并签发 token
func (s *OAuthService) Login(ctx context.Context, providerName, code string) (*domain.TokenPair, error) {
	p, ok := s.providers[providerName]
	if !ok {
		return nil, errors.New(errors.CodeOAuthProviderNotFound, "unknown oauth provider: "+providerName)
	}
	info, err := p.Exchange(ctx, code)
	if err != nil {
		return nil, errors.Wrap(errors.CodeInternalError, "oauth exchange failed", err)
	}

	// 匹配绑定
	binding, err := s.bindingRepo.FindByProviderSubject(ctx, providerName, info.Subject)
	var userID uint64
	if err == nil {
		userID = binding.UserID
	} else {
		// 创建用户 + 绑定
		username := fmt.Sprintf("%s_%s", providerName, info.Subject)
		if len(username) > 64 {
			username = username[:64]
		}
		randPwd := uuid.NewString()[:16]
		hashed, err := bcrypt.GenerateFromPassword([]byte(randPwd), bcrypt.DefaultCost)
		if err != nil {
			return nil, errors.Wrap(errors.CodeInternalError, "hash oauth password", err)
		}
		user := &userdomain.User{
			Username: username,
			Password: string(hashed),
			Status:   1,
		}
		if err := s.userRepo.Create(ctx, user); err != nil {
			return nil, errors.Wrap(errors.CodeInternalError, "create oauth user", err)
		}
		userID = user.ID
		if err := s.bindingRepo.Create(ctx, &oauthdomain.OAuthBinding{
			UserID:   userID,
			Provider: providerName,
			Subject:  info.Subject,
		}); err != nil {
			return nil, errors.Wrap(errors.CodeInternalError, "create oauth binding", err)
		}
	}

	// 签发 token
	sessionID := uuid.NewString()
	accessToken, err := s.jwtUtil.GenerateAccess(userID, sessionID)
	if err != nil {
		return nil, errors.Wrap(errors.CodeInternalError, "generate access token", err)
	}
	refreshToken, refreshClaims, err := s.jwtUtil.GenerateRefresh(userID, sessionID)
	if err != nil {
		return nil, errors.Wrap(errors.CodeInternalError, "generate refresh token", err)
	}
	if err := s.sessions.Create(ctx, userID, sessionID, refreshClaims.ID, refreshTTL(refreshClaims)); err != nil {
		return nil, errors.Wrap(errors.CodeInternalError, "create session", err)
	}

	return &domain.TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    s.accessMin * 60,
	}, nil
}

// refreshTTL 计算 refresh token 剩余有效期（与 auth 模块一致）
func refreshTTL(claims auth.Claims) time.Duration {
	if claims.ExpiresAt == nil {
		return 0
	}
	return time.Until(claims.ExpiresAt.Time)
}
