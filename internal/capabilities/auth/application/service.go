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

	authdomain "jimu/internal/capabilities/auth/domain"
	"jimu/internal/capabilities/encryption"
	"jimu/internal/capabilities/notification"
	"jimu/internal/capabilities/outbox"
	userdomain "jimu/internal/capabilities/user/domain"
	"jimu/internal/contract"
	"jimu/internal/kernel/auth"
	"jimu/internal/kernel/tenant"
	"jimu/internal/shared/errors"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type AuthService struct {
	userRepo             userdomain.UserRepository
	jwtUtil              *auth.JWT
	sessions             auth.SessionStore
	lockout              *auth.LoginFailureTracker
	accessMin            int
	outbox               *outbox.Outbox
	cipher               *encryption.Cipher
	notifier             notification.Dispatcher
	resetStore           *ResetStore
	loginHistory         authdomain.LoginHistoryRepository    // 登录历史（nil=不记录）
	passwordHistory      authdomain.PasswordHistoryRepository // 密码历史（nil=不做防复用检查）
	passwordHistoryCount int
	resetGen             func() string              // 验证码生成器（测试注入）
	provisioner          contract.TenantProvisioner // 开通式注册（nil = 未启用，注册仅建普通用户）
	breachChecker        contract.BreachChecker     // 泄露口令检查（nil = 未启用）
	quota                TenantQuota                // 租户配额（nil = 未启用）
	mfa                  contract.MFAVerifier       // 二次验证 + 可信设备（nil = 未启用）
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
		case contract.TenantProvisioner:
			s.provisioner = d
		case contract.BreachChecker:
			s.breachChecker = d
		case TenantQuota:
			s.quota = d
		case contract.MFAVerifier:
			s.mfa = d
		case authdomain.LoginHistoryRepository:
			s.loginHistory = d
		case authdomain.PasswordHistoryRepository:
			s.passwordHistory = d
		case passwordHistoryCount:
			s.passwordHistoryCount = int(d)
		}
	}
	return s
}

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
			s.recordLoginHistory(ctx, 0, 0, normalized, authdomain.LoginStatusLocked, "account locked")
			return nil, errors.AccountLocked(remaining)
		}
	}

	user, err := s.userRepo.FindByUsername(ctx, normalized)
	if err != nil {
		s.recordFailure(ctx, normalized)
		s.recordLoginHistory(ctx, 0, 0, normalized, authdomain.LoginStatusFailed, "user not found")
		return nil, invalidCredentials()
	}
	if user.Status != 1 {
		s.recordFailure(ctx, normalized)
		s.recordLoginHistory(ctx, user.ID, user.TenantID, normalized, authdomain.LoginStatusFailed, "user disabled")
		return nil, invalidCredentials()
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		s.recordFailure(ctx, normalized)
		s.recordLoginHistory(ctx, user.ID, user.TenantID, normalized, authdomain.LoginStatusFailed, "invalid password")
		return nil, invalidCredentials()
	}

	// TOTP 校验：用户启用后必须提供有效验证码；携带该用户的可信设备令牌时可跳过。
	// 判定与校验全部委托 mfa 能力（contract.MFAVerifier），auth 不感知密钥存储。
	if s.mfa != nil {
		mfaEnabled, mfaErr := s.mfa.Enabled(ctx, user.ID)
		if mfaErr != nil {
			return nil, errors.Wrap(errors.CodeInternalError, "mfa state check failed", mfaErr)
		}
		if mfaEnabled {
			if err := s.mfa.VerifyTOTP(ctx, user.ID, user.TenantID, totpCode); err != nil {
				if appErr, ok := err.(*errors.AppError); ok && appErr.Code == errors.CodeInvalidMFA {
					s.recordFailure(ctx, normalized)
					s.recordLoginHistory(ctx, user.ID, user.TenantID, normalized, authdomain.LoginStatusFailed, "invalid totp code")
				} else {
					s.recordLoginHistory(ctx, user.ID, user.TenantID, normalized, authdomain.LoginStatusFailed, "totp code required")
				}
				return nil, err
			}
		}
	}

	// 登录成功，清除失败计数
	if s.lockout != nil {
		_ = s.lockout.Reset(ctx, normalized)
	}

	pair, err := s.finishLogin(ctx, user)
	if err != nil {
		return nil, err
	}
	// 「记住此设备」：仅在启用 TOTP 的账号上签发；签发失败不影响登录
	if s.mfa != nil {
		if token := s.mfa.MaybeIssueDevice(ctx, user.ID, user.TenantID); token != "" {
			pair.DeviceToken = token
		}
	}
	return pair, nil
}

// finishLogin 校验通过后的公共登录收尾：签发 token + 建会话 + Outbox 事件。
func (s *AuthService) finishLogin(ctx context.Context, user *userdomain.User) (*authdomain.TokenPair, error) {
	sessionID := uuid.NewString()
	accessToken, refreshToken, refreshClaims, err := s.issueTokenPair(user.ID, effectiveTenantID(user.TenantID), sessionID)
	if err != nil {
		return nil, err
	}
	if err := s.sessions.Create(ctx, user.ID, sessionID, refreshClaims.ID, refreshTTL(refreshClaims)); err != nil {
		return nil, errors.Wrap(errors.CodeInternalError, "failed to create session", err)
	}
	s.recordLoginHistory(ctx, user.ID, user.TenantID, user.Username, authdomain.LoginStatusSuccess, "")

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
	if err := s.checkRegistrationAvailable(ctx, username, email); err != nil {
		return nil, err
	}

	if err := s.checkBreachedPassword(ctx, password); err != nil {
		return nil, err
	}
	// 公开注册的用户归默认租户：默认租户被分配套餐时同样受配额约束
	if s.quota != nil {
		if err := s.quota.CheckUserQuota(ctx, tenant.DefaultTenantID); err != nil {
			return nil, err
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
		// 公开注册入口无租户上下文，新用户归默认租户
		TenantID: tenant.DefaultTenantID,
	}
	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, errors.Wrap(errors.CodeInternalError, "failed to create user", err)
	}
	return user, nil
}

// RegisterTenantRequest 开通式注册请求：创建新租户，注册者成为该租户 owner
type RegisterTenantRequest struct {
	Username   string
	Password   string
	Email      string
	Phone      string
	TenantName string // 租户名称（必填）
	TenantCode string // 租户编码（可选；空则自动生成）
}

// RegisterProvisioned 开通式注册（auth.provisioning.enabled）：单事务创建
// 新租户 + owner 用户 + 模板角色与权限绑定。未启用时返回参数错误。
// 实际开通事务由 tenant 能力经 contract.TenantProvisioner 端口执行。
func (s *AuthService) RegisterProvisioned(ctx context.Context, req RegisterTenantRequest) (*contract.ProvisionResult, error) {
	if s.provisioner == nil {
		return nil, errors.New(errors.CodeInvalidParam, "tenant provisioning is disabled")
	}
	if req.TenantName == "" {
		return nil, errors.New(errors.CodeInvalidParam, "tenant_name is required for provisioned registration")
	}
	username := normalizeUsername(req.Username)
	if err := s.checkRegistrationAvailable(ctx, username, req.Email); err != nil {
		return nil, err
	}
	if err := s.checkBreachedPassword(ctx, req.Password); err != nil {
		return nil, err
	}
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, errors.Wrap(errors.CodeInternalError, "failed to hash password", err)
	}
	return s.provisioner.Provision(ctx, contract.ProvisionRequest{
		Username:     username,
		PasswordHash: string(hashedPassword),
		Email:        req.Email,
		Phone:        req.Phone,
		TenantName:   req.TenantName,
		TenantCode:   req.TenantCode,
	})
}

// checkRegistrationAvailable 注册前置查重：用户名 + 邮箱盲索引（DB unique 兜底并发）
func (s *AuthService) checkRegistrationAvailable(ctx context.Context, username, email string) error {
	existing, _ := s.userRepo.FindByUsername(ctx, username)
	if existing != nil {
		return errors.New(errors.CodeUserExists, "username already exists")
	}

	// 邮箱查重（盲索引精确查询）；空邮箱跳过，DB unique 索引兜底
	if email != "" && s.cipher != nil {
		existing, _ := s.userRepo.FindByEmailHash(ctx, s.cipher.BlindIndex(email))
		if existing != nil {
			return errors.New(errors.CodeUserExists, "email already exists")
		}
	}
	return nil
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
	if err := s.checkPasswordReuse(ctx, user, newPassword); err != nil {
		return err
	}
	if err := s.checkBreachedPassword(ctx, newPassword); err != nil {
		return err
	}
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return errors.Wrap(errors.CodeInternalError, "failed to hash password", err)
	}
	if err := s.userRepo.UpdatePassword(ctx, user.ID, string(hashedPassword)); err != nil {
		return errors.Wrap(errors.CodeInternalError, "failed to update password", err)
	}
	// 记录被替换掉的旧密码，供后续防复用检查
	s.recordPasswordHistory(ctx, user)
	// 改密后吊销全部可信设备：旧设备不得继续跳过 TOTP
	if s.mfa != nil {
		s.mfa.RevokeDevices(ctx, user.ID)
	}
	if s.sessions != nil {
		_ = s.sessions.RevokeAll(ctx, user.ID)
	}
	return nil
}

// checkBreachedPassword 检查口令是否出现在已知泄露集合中（HIBP）。
// 未启用检查器时直接放行；检查服务不可用时也放行（只记日志），
// 避免外部依赖故障阻断注册与改密。
func (s *AuthService) checkBreachedPassword(ctx context.Context, password string) error {
	if s.breachChecker == nil {
		return nil
	}
	breached, err := s.breachChecker.IsBreached(ctx, password)
	if err != nil {
		log.Printf("auth: breach check skipped: %v", err)
		return nil
	}
	if breached {
		return errors.New(errors.CodePasswordBreached, "password has appeared in a known data breach")
	}
	return nil
}

// passwordHistoryCount 注入防复用检查的历史条数（0=关闭）的 dep。
type passwordHistoryCount int

// WithPasswordHistory 返回注入防复用历史条数的 dep，供 NewAuthService 使用。
func WithPasswordHistory(count int) interface{} { return passwordHistoryCount(count) }

// checkPasswordReuse 阻止把密码改回当前值或最近使用过的历史密码。
// 未配置仓储或条数为 0 时跳过检查（视为未启用该策略）。
func (s *AuthService) checkPasswordReuse(ctx context.Context, user *userdomain.User, newPassword string) error {
	if s.passwordHistory == nil || s.passwordHistoryCount <= 0 {
		return nil
	}
	if bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(newPassword)) == nil {
		return errors.New(errors.CodePasswordReused, "new password must differ from the current one")
	}
	hashes, err := s.passwordHistory.ListRecentHashes(ctx, user.ID, s.passwordHistoryCount)
	if err != nil {
		return errors.Wrap(errors.CodeInternalError, "failed to load password history", err)
	}
	for _, hash := range hashes {
		if bcrypt.CompareHashAndPassword([]byte(hash), []byte(newPassword)) == nil {
			return errors.New(errors.CodePasswordReused, "password was used recently")
		}
	}
	return nil
}

// recordPasswordHistory 记录旧密码哈希并裁剪到配置条数；失败只记日志，不阻断改密结果。
func (s *AuthService) recordPasswordHistory(ctx context.Context, user *userdomain.User) {
	if s.passwordHistory == nil || s.passwordHistoryCount <= 0 || user.Password == "" {
		return
	}
	if err := s.passwordHistory.Add(ctx, user.TenantID, user.ID, user.Password); err != nil {
		log.Printf("auth: record password history for user %d: %v", user.ID, err)
		return
	}
	if err := s.passwordHistory.Trim(ctx, user.ID, s.passwordHistoryCount); err != nil {
		log.Printf("auth: trim password history for user %d: %v", user.ID, err)
	}
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

	accessToken, newRefreshToken, newRefreshClaims, err := s.issueTokenPair(claims.UserID, claims.TenantID, claims.SessionID)
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
	// 登出全部设备同时吊销可信设备，避免遗留可跳过 TOTP 的凭证
	if s.mfa != nil {
		s.mfa.RevokeDevices(ctx, userID)
	}
	return nil
}

func (s *AuthService) issueTokenPair(userID, tenantID uint64, sessionID string) (string, string, auth.Claims, error) {
	accessToken, err := s.jwtUtil.GenerateAccess(userID, tenantID, sessionID)
	if err != nil {
		return "", "", auth.Claims{}, errors.Wrap(errors.CodeInternalError, "failed to generate access token", err)
	}
	refreshToken, refreshClaims, err := s.jwtUtil.GenerateRefresh(userID, tenantID, sessionID)
	if err != nil {
		return "", "", auth.Claims{}, errors.Wrap(errors.CodeInternalError, "failed to generate refresh token", err)
	}
	return accessToken, refreshToken, refreshClaims, nil
}

// effectiveTenantID 返回用于签发的租户 ID；未归属（0）时归默认租户。
func effectiveTenantID(userTenant uint64) uint64 {
	if userTenant == 0 {
		return tenant.DefaultTenantID
	}
	return userTenant
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

// ---- contract.LoginFinalizer 实现：供 passkey 无密码登录复用登录收尾 ----

// FinalizeLogin 为已验证用户签发令牌、建会话、记登录历史、发事件。
// 返回视图不含 device_token（passkey 不签发可信设备）。
func (s *AuthService) FinalizeLogin(ctx context.Context, userID uint64) (*contract.TokenPair, error) {
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil || user.Status != 1 {
		return nil, invalidCredentials()
	}
	pair, err := s.finishLogin(ctx, user)
	if err != nil {
		return nil, err
	}
	return &contract.TokenPair{
		AccessToken:  pair.AccessToken,
		RefreshToken: pair.RefreshToken,
		ExpiresIn:    pair.ExpiresIn,
	}, nil
}

// CheckLocked 登录前的失败锁定检查（未锁定返回 false）。
func (s *AuthService) CheckLocked(ctx context.Context, username string) (bool, time.Duration, error) {
	if s.lockout == nil {
		return false, 0, nil
	}
	return s.lockout.CheckLocked(ctx, normalizeUsername(username))
}

// RecordFailure 实现 contract.LoginFinalizer：记录一次失败尝试（不影响主流程）。
func (s *AuthService) RecordFailure(ctx context.Context, username string) {
	s.recordFailure(ctx, normalizeUsername(username))
}

// ResetFailure 实现 contract.LoginFinalizer：登录成功后清除失败计数。
func (s *AuthService) ResetFailure(ctx context.Context, username string) {
	if s.lockout != nil {
		_ = s.lockout.Reset(ctx, normalizeUsername(username))
	}
}

// RecordLoginHistory 实现 contract.LoginFinalizer：记录一次登录尝试。
func (s *AuthService) RecordLoginHistory(ctx context.Context, userID, tenantID uint64, username, status, reason string) {
	s.recordLoginHistory(ctx, userID, tenantID, username, status, reason)
}
