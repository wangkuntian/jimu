package application

import (
	"context"
	stderrors "errors"
	"fmt"
	"strings"
	"time"

	authdomain "jimu/internal/modules/auth/domain"
	userdomain "jimu/internal/modules/user/domain"
	"jimu/internal/platform/auth"
	"jimu/internal/shared/errors"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type AuthService struct {
	userRepo  userdomain.UserRepository
	jwtUtil   *auth.JWT
	sessions  auth.SessionStore
	lockout   *auth.LoginFailureTracker
	accessMin int
}

func NewAuthService(userRepo userdomain.UserRepository, jwtUtil *auth.JWT, sessions auth.SessionStore, lockout *auth.LoginFailureTracker, accessMin int) *AuthService {
	return &AuthService{
		userRepo:  userRepo,
		jwtUtil:   jwtUtil,
		sessions:  sessions,
		lockout:   lockout,
		accessMin: accessMin,
	}
}

func (s *AuthService) Login(ctx context.Context, username, password string) (*authdomain.TokenPair, error) {
	normalized := normalizeUsername(username)

	// 检查账号是否被锁定
	if s.lockout != nil {
		locked, remaining, err := s.lockout.CheckLocked(ctx, normalized)
		if err != nil {
			return nil, errors.Wrap(errors.CodeInternalError, "lockout check failed", err)
		}
		if locked {
			return nil, ErrAccountLocked(remaining)
		}
	}

	user, err := s.userRepo.FindByUsername(ctx, normalized)
	if err != nil {
		s.recordFailure(ctx, normalized)
		return nil, invalidCredentials()
	}
	if user.Status != 1 {
		s.recordFailure(ctx, normalized)
		return nil, invalidCredentials()
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		s.recordFailure(ctx, normalized)
		return nil, invalidCredentials()
	}

	// 登录成功，清除失败计数
	if s.lockout != nil {
		_ = s.lockout.Reset(ctx, normalized)
	}

	sessionID := uuid.NewString()
	accessToken, refreshToken, refreshClaims, err := s.issueTokenPair(user.ID, sessionID)
	if err != nil {
		return nil, err
	}
	if err := s.sessions.Create(ctx, user.ID, sessionID, refreshClaims.ID, refreshTTL(refreshClaims)); err != nil {
		return nil, errors.Wrap(errors.CodeInternalError, "failed to create session", err)
	}

	return &authdomain.TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    s.accessMin * 60,
	}, nil
}

func (s *AuthService) recordFailure(ctx context.Context, username string) {
	if s.lockout == nil {
		return
	}
	// 锁定记录失败不影响主流程，仅忽略
	_, _ = s.lockout.RecordFailure(ctx, username)
}

func (s *AuthService) Register(ctx context.Context, username, password string) (*userdomain.User, error) {
	username = normalizeUsername(username)
	existing, _ := s.userRepo.FindByUsername(ctx, username)
	if existing != nil {
		return nil, errors.New(errors.CodeUserExists, "username already exists")
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, errors.Wrap(errors.CodeInternalError, "failed to hash password", err)
	}

	user := &userdomain.User{
		Username: username,
		Password: string(hashedPassword),
		Status:   1,
	}
	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, errors.Wrap(errors.CodeInternalError, "failed to create user", err)
	}
	return user, nil
}

func (s *AuthService) Refresh(ctx context.Context, refreshToken string) (*authdomain.TokenPair, error) {
	claims, err := s.jwtUtil.Parse(refreshToken, auth.TokenTypeRefresh)
	if err != nil {
		return nil, errors.New(errors.CodeUnauthorized, "invalid refresh token")
	}

	accessToken, newRefreshToken, newRefreshClaims, err := s.issueTokenPair(claims.UserID, claims.SessionID)
	if err != nil {
		return nil, err
	}
	err = s.sessions.Rotate(ctx, claims.UserID, claims.SessionID, claims.ID, newRefreshClaims.ID, refreshTTL(newRefreshClaims))
	if err != nil {
		if stderrors.Is(err, auth.ErrSessionNotFound) || stderrors.Is(err, auth.ErrTokenReuse) {
			return nil, errors.New(errors.CodeUnauthorized, "invalid refresh token")
		}
		return nil, errors.Wrap(errors.CodeInternalError, "failed to rotate session", err)
	}

	return &authdomain.TokenPair{
		AccessToken:  accessToken,
		RefreshToken: newRefreshToken,
		ExpiresIn:    s.accessMin * 60,
	}, nil
}

func (s *AuthService) Logout(ctx context.Context, userID uint64, sessionID string) error {
	if sessionID == "" {
		return errors.New(errors.CodeUnauthorized, "invalid session")
	}
	if err := s.sessions.Revoke(ctx, userID, sessionID); err != nil {
		return errors.Wrap(errors.CodeInternalError, "failed to revoke session", err)
	}
	return nil
}

func (s *AuthService) LogoutAll(ctx context.Context, userID uint64) error {
	if err := s.sessions.RevokeAll(ctx, userID); err != nil {
		return errors.Wrap(errors.CodeInternalError, "failed to revoke sessions", err)
	}
	return nil
}

func (s *AuthService) issueTokenPair(userID uint64, sessionID string) (string, string, auth.Claims, error) {
	accessToken, err := s.jwtUtil.GenerateAccess(userID, sessionID)
	if err != nil {
		return "", "", auth.Claims{}, errors.Wrap(errors.CodeInternalError, "failed to generate access token", err)
	}
	refreshToken, refreshClaims, err := s.jwtUtil.GenerateRefresh(userID, sessionID)
	if err != nil {
		return "", "", auth.Claims{}, errors.Wrap(errors.CodeInternalError, "failed to generate refresh token", err)
	}
	return accessToken, refreshToken, refreshClaims, nil
}

func refreshTTL(claims auth.Claims) time.Duration {
	if claims.ExpiresAt == nil {
		return 0
	}
	return time.Until(claims.ExpiresAt.Time)
}

func normalizeUsername(username string) string {
	return strings.ToLower(strings.TrimSpace(username))
}

func invalidCredentials() error {
	return errors.New(errors.CodeInvalidCredentials, "invalid credentials")
}

// ErrAccountLocked 返回账号锁定错误，message 包含剩余锁定时间
func ErrAccountLocked(remaining time.Duration) error {
	minutes := int(remaining.Minutes())
	if minutes < 1 {
		minutes = 1
	}
	return errors.New(errors.CodeForbidden, fmt.Sprintf("account locked due to too many failed attempts, try again in %d minutes", minutes))
}
