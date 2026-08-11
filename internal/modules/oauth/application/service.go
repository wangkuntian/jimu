// internal/modules/oauth/application/service.go
package application

import (
	"context"
	stderrors "errors"
	"fmt"
	"time"

	"jimu/internal/modules/auth/domain"
	oauthdomain "jimu/internal/modules/oauth/domain"
	userdomain "jimu/internal/modules/user/domain"
	"jimu/internal/platform/auth"
	oauthplatform "jimu/internal/platform/oauth"
	"jimu/internal/shared/errors"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// oauthStateTTL OAuth state 有效期
const oauthStateTTL = 10 * time.Minute

// oauthStateKey OAuth state 的 Redis key
func oauthStateKey(state string) string { return "jimu:oauth:state:" + state }

// OAuthService OAuth 登录服务
type OAuthService struct {
	userRepo    userdomain.UserRepository
	bindingRepo oauthdomain.BindingRepository
	jwtUtil     *auth.JWT
	sessions    auth.SessionStore
	providers   map[string]oauthplatform.Provider
	rdb         *redis.Client
	db          *gorm.DB
	accessMin   int
}

// NewOAuthService 创建 OAuth 服务
func NewOAuthService(userRepo userdomain.UserRepository, bindingRepo oauthdomain.BindingRepository, jwtUtil *auth.JWT, sessions auth.SessionStore, providers map[string]oauthplatform.Provider, rdb *redis.Client, db *gorm.DB, accessMin int) *OAuthService {
	return &OAuthService{
		userRepo:    userRepo,
		bindingRepo: bindingRepo,
		jwtUtil:     jwtUtil,
		sessions:    sessions,
		providers:   providers,
		rdb:         rdb,
		db:          db,
		accessMin:   accessMin,
	}
}

// AuthURL 构造授权跳转 URL 并存储 state（防 CSRF）
func (s *OAuthService) AuthURL(ctx context.Context, providerName, state string) (string, error) {
	p, ok := s.providers[providerName]
	if !ok {
		return "", errors.New(errors.CodeOAuthProviderNotFound, "unknown oauth provider: "+providerName)
	}
	if err := s.rdb.Set(ctx, oauthStateKey(state), providerName, oauthStateTTL).Err(); err != nil {
		return "", errors.Wrap(errors.CodeInternalError, "store oauth state", err)
	}
	return p.AuthURL(state), nil
}

// BeginLogin 生成 state 并返回授权跳转 URL
func (s *OAuthService) BeginLogin(ctx context.Context, providerName string) (string, string, error) {
	state := uuid.NewString()
	url, err := s.AuthURL(ctx, providerName, state)
	if err != nil {
		return "", "", err
	}
	return url, state, nil
}

// consumeState 校验并一次性消费 state（防 CSRF / 登录会话固定）
func (s *OAuthService) consumeState(ctx context.Context, state, providerName string) error {
	if state == "" {
		return errors.New(errors.CodeInvalidParam, "invalid oauth state")
	}
	key := oauthStateKey(state)
	val, err := s.rdb.Get(ctx, key).Result()
	if err == redis.Nil || val != providerName {
		return errors.New(errors.CodeInvalidParam, "invalid oauth state")
	}
	if err != nil {
		return errors.Wrap(errors.CodeInternalError, "read oauth state", err)
	}
	if err := s.rdb.Del(ctx, key).Err(); err != nil {
		return errors.Wrap(errors.CodeInternalError, "delete oauth state", err)
	}
	return nil
}

// Login 处理 OAuth 回调，匹配/创建用户并签发 token
func (s *OAuthService) Login(ctx context.Context, providerName, code, state string) (*domain.TokenPair, error) {
	if err := s.consumeState(ctx, state, providerName); err != nil {
		return nil, err
	}
	p, ok := s.providers[providerName]
	if !ok {
		return nil, errors.New(errors.CodeOAuthProviderNotFound, "unknown oauth provider: "+providerName)
	}
	info, err := p.Exchange(ctx, code)
	if err != nil {
		return nil, errors.Wrap(errors.CodeInternalError, "oauth exchange failed", err)
	}

	// 匹配绑定（非 RecordNotFound 错误直接返回，不误入创建分支）
	binding, err := s.bindingRepo.FindByProviderSubject(ctx, providerName, info.Subject)
	var userID uint64
	if err != nil {
		if !stderrors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.Wrap(errors.CodeInternalError, "find oauth binding", err)
		}
		userID, err = s.createUserWithBinding(ctx, providerName, info)
		if err != nil {
			// 并发冲突（重复用户/绑定，事务回滚）：回退重查绑定
			binding, findErr := s.bindingRepo.FindByProviderSubject(ctx, providerName, info.Subject)
			if findErr != nil {
				return nil, errors.Wrap(errors.CodeInternalError, "create oauth user binding", err)
			}
			userID = binding.UserID
		}
	} else {
		userID = binding.UserID
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

// createUserWithBinding 在同一事务内创建用户与绑定，保证原子性
func (s *OAuthService) createUserWithBinding(ctx context.Context, providerName string, info *oauthplatform.UserInfo) (uint64, error) {
	username := fmt.Sprintf("%s_%s", providerName, info.Subject)
	if len(username) > 64 {
		username = username[:64]
	}
	randPwd := uuid.NewString()[:16]
	hashed, err := bcrypt.GenerateFromPassword([]byte(randPwd), bcrypt.DefaultCost)
	if err != nil {
		return 0, err
	}
	user := &userdomain.User{
		Username: username,
		Password: string(hashed),
		Status:   1,
	}
	var userID uint64
	if err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(user).Error; err != nil {
			return err
		}
		userID = user.ID
		return tx.Create(&oauthdomain.OAuthBinding{
			UserID:   userID,
			Provider: providerName,
			Subject:  info.Subject,
		}).Error
	}); err != nil {
		return 0, err
	}
	return userID, nil
}

// refreshTTL 计算 refresh token 剩余有效期（与 auth 模块一致）
func refreshTTL(claims auth.Claims) time.Duration {
	if claims.ExpiresAt == nil {
		return 0
	}
	return time.Until(claims.ExpiresAt.Time)
}
