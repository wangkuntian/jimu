package application

import (
	"context"
	"crypto/rand"
	"encoding/json"
	stderrors "errors"
	"fmt"
	"log"
	"strings"
	"time"

	"jimu/internal/contract"
	authdomain "jimu/internal/modules/auth/domain"
	userdomain "jimu/internal/modules/user/domain"
	"jimu/internal/platform/auth"
	"jimu/internal/platform/encryption"
	"jimu/internal/platform/notification"
	"jimu/internal/platform/outbox"
	"jimu/internal/shared/errors"
	"jimu/internal/shared/totp"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type AuthService struct {
	userRepo   userdomain.UserRepository
	jwtUtil    *auth.JWT
	sessions   auth.SessionStore
	lockout    *auth.LoginFailureTracker
	accessMin  int
	outbox     *outbox.Outbox
	cipher     *encryption.Cipher
	notifier   notification.Dispatcher
	resetStore *ResetStore
	resetGen   func() string // 验证码生成器（测试注入）
	issuer     string        // TOTP otpauth URI 的 issuer
}

func NewAuthService(userRepo userdomain.UserRepository, jwtUtil *auth.JWT, sessions auth.SessionStore, lockout *auth.LoginFailureTracker, accessMin int, deps ...interface{}) *AuthService {
	s := &AuthService{
		userRepo:  userRepo,
		jwtUtil:   jwtUtil,
		sessions:  sessions,
		lockout:   lockout,
		accessMin: accessMin,
	}
	for _, dep := range deps {
		switch d := dep.(type) {
		case *outbox.Outbox:
			s.outbox = d
		case *encryption.Cipher:
			s.cipher = d
		case notification.Dispatcher:
			s.notifier = d
		case *ResetStore:
			s.resetStore = d
		case totpIssuer:
			s.issuer = string(d)
		}
	}
	return s
}

// totpIssuer 注入 TOTP otpauth URI 的 issuer（默认 jimu）。
type totpIssuer string

// WithIssuer 返回注入 issuer 的 dep，供 NewAuthService 使用。
func WithIssuer(issuer string) interface{} { return totpIssuer(issuer) }

func (s *AuthService) Login(ctx context.Context, username, password string) (*authdomain.TokenPair, error) {
	// 兼容入口：不提供 TOTP 码。用户启用 TOTP 时返回 CodeMFARequired 提示二次验证。
	return s.LoginWithTOTP(ctx, username, password, "")
}

// LoginWithTOTP 支持 TOTP 二次验证的登录。用户启用 TOTP 时校验验证码：
// 码缺失返回 CodeMFARequired，码无效返回 CodeInvalidMFA；未启用 TOTP 时与旧登录等价。
func (s *AuthService) LoginWithTOTP(ctx context.Context, username, password, totpCode string) (*authdomain.TokenPair, error) {
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

	// TOTP 校验：用户启用后必须提供有效验证码
	if user.TOTPEnabled {
		if totpCode == "" {
			return nil, errors.New(errors.CodeMFARequired, "TOTP code required")
		}
		if !totp.Validate(user.TOTPSecret, totpCode, time.Now(), totp.DefaultPeriod, totp.DefaultDigits, totp.DefaultSkew) {
			s.recordFailure(ctx, normalized)
			return nil, errors.New(errors.CodeInvalidMFA, "invalid TOTP code")
		}
	}

	// 登录成功，清除失败计数
	if s.lockout != nil {
		_ = s.lockout.Reset(ctx, normalized)
	}

	return s.finishLogin(ctx, user)
}

// finishLogin 校验通过后的公共登录收尾：签发 token + 建会话 + Outbox 事件。
func (s *AuthService) finishLogin(ctx context.Context, user *userdomain.User) (*authdomain.TokenPair, error) {
	sessionID := uuid.NewString()
	accessToken, refreshToken, refreshClaims, err := s.issueTokenPair(user.ID, sessionID)
	if err != nil {
		return nil, err
	}
	if err := s.sessions.Create(ctx, user.ID, sessionID, refreshClaims.ID, refreshTTL(refreshClaims)); err != nil {
		return nil, errors.Wrap(errors.CodeInternalError, "failed to create session", err)
	}

	// 写入 Outbox 发布登录成功事件（同用户创建一致，走统一可靠投递路径）
	if s.outbox != nil {
		payload, err := json.Marshal(contract.UserLoggedInEvent{
			UserID:   user.ID,
			Username: user.Username,
		})
		if err != nil {
			log.Printf("auth: marshal logged_in event: %v", err)
		} else if err := s.outbox.Add(ctx, nil, outbox.Event{
			AggregateID: fmt.Sprintf("user:%d", user.ID),
			EventType:   contract.EventUserLoggedIn,
			Payload:     payload,
		}); err != nil {
			log.Printf("auth: write outbox event %s: %v", contract.EventUserLoggedIn, err)
		}
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

func (s *AuthService) Register(ctx context.Context, username, password, email, phone string) (*userdomain.User, error) {
	username = normalizeUsername(username)
	existing, _ := s.userRepo.FindByUsername(ctx, username)
	if existing != nil {
		return nil, errors.New(errors.CodeUserExists, "username already exists")
	}

	// 邮箱查重（盲索引精确查询）；空邮箱跳过，DB unique 索引兜底
	if email != "" && s.cipher != nil {
		existing, _ := s.userRepo.FindByEmailHash(ctx, s.cipher.BlindIndex(email))
		if existing != nil {
			return nil, errors.New(errors.CodeUserExists, "email already exists")
		}
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, errors.Wrap(errors.CodeInternalError, "failed to hash password", err)
	}

	user := &userdomain.User{
		Username: username,
		Password: string(hashedPassword),
		Email:    email,
		Phone:    phone,
		Status:   1,
	}
	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, errors.Wrap(errors.CodeInternalError, "failed to create user", err)
	}
	return user, nil
}

// ForgotPassword 发送密码重置验证码到邮箱。用户不存在仍返回成功，避免邮箱枚举。
func (s *AuthService) ForgotPassword(ctx context.Context, email string) error {
	if s.cipher == nil || s.notifier == nil || s.resetStore == nil {
		return errors.New(errors.CodeInternalError, "password reset not configured")
	}
	hash := s.cipher.BlindIndex(email)
	user, err := s.userRepo.FindByEmailHash(ctx, hash)
	if err != nil && !stderrors.Is(err, gorm.ErrRecordNotFound) {
		return errors.Wrap(errors.CodeInternalError, "failed to find user by email", err)
	}
	if user == nil {
		return nil // 用户不存在：不暴露，静默成功
	}

	code := s.generateResetCode()
	if err := s.resetStore.Set(ctx, hash, code); err != nil {
		return errors.Wrap(errors.CodeInternalError, "failed to store reset code", err)
	}
	if err := s.notifier.Dispatch(ctx, notification.Message{
		Channel: notification.ChannelEmail,
		To:      email,
		Subject: "密码重置验证码",
		Body:    fmt.Sprintf("你的验证码：%s，%d 分钟内有效。若非本人操作请忽略。", code, int(s.resetStore.ttl.Minutes())),
	}); err != nil {
		return errors.Wrap(errors.CodeInternalError, "failed to send reset email", err)
	}
	return nil
}

// ResetPassword 用邮箱验证码设置新密码。验证码一次性（Lua 原子消费），成功后强制登出全部会话。
func (s *AuthService) ResetPassword(ctx context.Context, email, code, newPassword string) error {
	if s.cipher == nil || s.resetStore == nil {
		return errors.New(errors.CodeInternalError, "password reset not configured")
	}
	hash := s.cipher.BlindIndex(email)
	stored, err := s.resetStore.GetAndDelete(ctx, hash)
	if err != nil {
		return errors.Wrap(errors.CodeInternalError, "failed to read reset code", err)
	}
	if stored == "" || stored != code {
		return errors.New(errors.CodeInvalidResetCode, "invalid or expired reset code")
	}

	user, err := s.userRepo.FindByEmailHash(ctx, hash)
	if err != nil {
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New(errors.CodeInvalidResetCode, "invalid or expired reset code")
		}
		return errors.Wrap(errors.CodeInternalError, "failed to find user by email", err)
	}
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return errors.Wrap(errors.CodeInternalError, "failed to hash password", err)
	}
	if err := s.userRepo.UpdatePassword(ctx, user.ID, string(hashedPassword)); err != nil {
		return errors.Wrap(errors.CodeInternalError, "failed to update password", err)
	}
	if s.sessions != nil {
		_ = s.sessions.RevokeAll(ctx, user.ID)
	}
	return nil
}

// SetupTOTP 为用户生成新的 TOTP 密钥并返回 otpauth URI（未启用，需 EnableTOTP 确认）。
// 重复调用会轮换密钥（旧密钥立即失效）。
// 返回数据仅此一次全量可见，调用方应在确认启用前保存 secret。
func (s *AuthService) SetupTOTP(ctx context.Context, userID uint64, account string) (secret string, uri string, err error) {
	if account == "" {
		// 未显式提供 account 时用用户名兜底（otpauth URI 的可读标识）
		if u, loadErr := s.userRepo.FindByID(ctx, userID); loadErr == nil && u != nil {
			account = u.Username
		}
	}
	secret, err = totp.Secret()
	if err != nil {
		return "", "", errors.Wrap(errors.CodeInternalError, "failed to generate totp secret", err)
	}
	// 保存密钥但暂不启用（enabled=false），等待 EnableTOTP 用首次验证码确认
	if err := s.userRepo.UpdateTOTP(ctx, userID, secret, false); err != nil {
		return "", "", errors.Wrap(errors.CodeInternalError, "failed to save totp secret", err)
	}
	issuer := "jimu"
	if s.issuer != "" {
		issuer = s.issuer
	}
	return secret, totp.ProvisioningURI(secret, account, issuer), nil
}

// EnableTOTP 用首次生成的验证码确认启用 TOTP。码验证通过后方可开启，防误绑。
func (s *AuthService) EnableTOTP(ctx context.Context, userID uint64, code string) error {
	if code == "" {
		return errors.New(errors.CodeMFARequired, "TOTP code required")
	}
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return errors.Wrap(errors.CodeInternalError, "failed to load user", err)
	}
	if user.TOTPSecret == "" {
		return errors.New(errors.CodeInvalidMFA, "TOTP not set up, call setup first")
	}
	if !totp.Validate(user.TOTPSecret, code, time.Now(), totp.DefaultPeriod, totp.DefaultDigits, totp.DefaultSkew) {
		return errors.New(errors.CodeInvalidMFA, "invalid TOTP code")
	}
	if err := s.userRepo.UpdateTOTP(ctx, userID, user.TOTPSecret, true); err != nil {
		return errors.Wrap(errors.CodeInternalError, "failed to enable totp", err)
	}
	return nil
}

// DisableTOTP 校验当前验证码后关闭 TOTP 并清除密钥。
func (s *AuthService) DisableTOTP(ctx context.Context, userID uint64, code string) error {
	if code == "" {
		return errors.New(errors.CodeMFARequired, "TOTP code required")
	}
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return errors.Wrap(errors.CodeInternalError, "failed to load user", err)
	}
	if !user.TOTPEnabled || user.TOTPSecret == "" {
		return errors.New(errors.CodeInvalidMFA, "TOTP not enabled")
	}
	if !totp.Validate(user.TOTPSecret, code, time.Now(), totp.DefaultPeriod, totp.DefaultDigits, totp.DefaultSkew) {
		return errors.New(errors.CodeInvalidMFA, "invalid TOTP code")
	}
	if err := s.userRepo.UpdateTOTP(ctx, userID, "", false); err != nil {
		return errors.Wrap(errors.CodeInternalError, "failed to disable totp", err)
	}
	return nil
}

// generateResetCode 生成 6 位数字验证码（crypto/rand；测试可注入 resetGen）
func (s *AuthService) generateResetCode() string {
	if s.resetGen != nil {
		return s.resetGen()
	}
	var b [6]byte
	if _, err := rand.Read(b[:]); err != nil {
		return fmt.Sprintf("%06d", time.Now().UnixNano()%1000000)
	}
	code := 0
	for _, v := range b {
		code = code*10 + int(v%10)
	}
	return fmt.Sprintf("%06d", code)
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
