package application

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"strconv"
	"strings"
	"time"

	passkeydomain "jimu/internal/capabilities/passkey/domain"
	"jimu/internal/contract"
	redistore "jimu/internal/kernel/redis"
	"jimu/internal/kernel/tenant"
	"jimu/internal/shared/errors"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	// webAuthnSessionPrefix 挑战会话的 Redis key 前缀（一次性，用完即删）
	webAuthnSessionPrefix = "jimu:webauthn:session:"
	// defaultWebAuthnSessionTTL 挑战有效期默认值
	defaultWebAuthnSessionTTL = 5 * time.Minute
	// webAuthnCeremonyRegister / webAuthnCeremonyLogin 会话绑定的仪式类型，避免注册会话被用于登录
	webAuthnCeremonyRegister = "register"
	webAuthnCeremonyLogin    = "login"
	// deviceLabelMaxLen 凭证名称截断上限
	deviceLabelMaxLen = 64
)

// webAuthnSession 一次 WebAuthn 仪式的服务端状态：库的 SessionData + 归属校验信息。
// 只存服务端（Redis），客户端仅拿到随机 session_id，防止挑战被替换。
type webAuthnSession struct {
	Ceremony string               `json:"ceremony"`
	UserID   uint64               `json:"user_id"`
	Username string               `json:"username"`
	Name     string               `json:"name,omitempty"` // 注册时用户填写的凭证名称
	Data     webauthn.SessionData `json:"data"`
}

// WebAuthnCredentialInfo 凭证对外信息（不含公钥等敏感字段）
type WebAuthnCredentialInfo struct {
	ID             uint64     `json:"id"`
	Name           string     `json:"name,omitempty"`
	AAGUID         string     `json:"aaguid,omitempty"`
	BackupEligible bool       `json:"backup_eligible"`
	BackupState    bool       `json:"backup_state"`
	UserVerified   bool       `json:"user_verified"`
	LastUsedAt     *time.Time `json:"last_used_at,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
}

// PasskeyService 通行密钥（WebAuthn）用例：无密码登录与凭证管理。
// 登录收尾（令牌/会话/登录历史/锁定）经 contract.LoginFinalizer 委托 auth 能力。
type PasskeyService struct {
	users      contract.UserinfoSource
	creds      passkeydomain.WebAuthnCredentialRepository
	webauthn   *webauthn.WebAuthn
	rdb        redistore.Client
	sessionTTL time.Duration
	finalizer  contract.LoginFinalizer
}

// Deps PasskeyService 的显式依赖。
type Deps struct {
	Users       contract.UserinfoSource
	Credentials passkeydomain.WebAuthnCredentialRepository
	WebAuthn    *webauthn.WebAuthn
	Redis       redistore.Client
	SessionTTL  time.Duration
	Finalizer   contract.LoginFinalizer
}

// NewPasskeyService 创建通行密钥服务。
func NewPasskeyService(deps Deps) *PasskeyService {
	return &PasskeyService{
		users:      deps.Users,
		creds:      deps.Credentials,
		webauthn:   deps.WebAuthn,
		rdb:        deps.Redis,
		sessionTTL: deps.SessionTTL,
		finalizer:  deps.Finalizer,
	}
}

// webAuthnUser 适配库的 webauthn.User 接口
type webAuthnUser struct {
	id          uint64
	tenantID    uint64
	username    string
	credentials []webauthn.Credential
}

func (u *webAuthnUser) WebAuthnID() []byte { return webauthnUserHandle(u.id) }
func (u *webAuthnUser) WebAuthnName() string {
	return u.username
}
func (u *webAuthnUser) WebAuthnDisplayName() string                { return u.username }
func (u *webAuthnUser) WebAuthnCredentials() []webauthn.Credential { return u.credentials }

// webauthnUserHandle 由用户 ID 生成稳定的 user handle（≤64 字节，登录时用于反查用户）
func webauthnUserHandle(userID uint64) []byte {
	return []byte("jimu-user-" + strconv.FormatUint(userID, 10))
}

// BeginWebAuthnRegistration 开始注册通行密钥（需已登录）：返回 options 与 session_id。
func (s *PasskeyService) BeginWebAuthnRegistration(ctx context.Context, userID uint64, name string) (*protocol.CredentialCreation, string, error) {
	if err := s.webAuthnReady(); err != nil {
		return nil, "", err
	}
	user, err := s.loadWebAuthnUser(ctx, userID)
	if err != nil {
		return nil, "", err
	}

	creation, session, err := s.webauthn.BeginRegistration(user,
		// 排除已注册的认证器，避免同一设备重复注册
		webauthn.WithExclusions(webauthn.Credentials(user.WebAuthnCredentials()).CredentialDescriptors()),
		// 优先使用可发现凭证（通行密钥），认证器不支持时回退为普通凭证
		webauthn.WithResidentKeyRequirement(protocol.ResidentKeyRequirementPreferred),
	)
	if err != nil {
		return nil, "", errors.Wrap(errors.CodeInternalError, "failed to begin webauthn registration", err)
	}

	sessionID, err := s.storeWebAuthnSession(ctx, webAuthnSession{
		Ceremony: webAuthnCeremonyRegister,
		UserID:   userID,
		Username: user.username,
		Name:     truncateString(strings.TrimSpace(name), deviceLabelMaxLen),
		Data:     *session,
	})
	if err != nil {
		return nil, "", err
	}
	return creation, sessionID, nil
}

// FinishWebAuthnRegistration 完成注册：校验证明并落库凭证。body 为浏览器返回的凭证 JSON。
func (s *PasskeyService) FinishWebAuthnRegistration(ctx context.Context, userID uint64, sessionID string, body []byte) (*WebAuthnCredentialInfo, error) {
	if err := s.webAuthnReady(); err != nil {
		return nil, err
	}
	session, err := s.consumeWebAuthnSession(ctx, sessionID, webAuthnCeremonyRegister, userID)
	if err != nil {
		return nil, err
	}
	user, err := s.loadWebAuthnUser(ctx, userID)
	if err != nil {
		return nil, err
	}

	parsed, err := protocol.ParseCredentialCreationResponseBody(bytes.NewReader(body))
	if err != nil {
		return nil, errors.Wrap(errors.CodeWebAuthnVerificationFailed, "invalid webauthn registration response", err)
	}
	credential, err := s.webauthn.CreateCredential(user, session.Data, parsed)
	if err != nil {
		return nil, errors.Wrap(errors.CodeWebAuthnVerificationFailed, "webauthn registration verification failed", err)
	}

	stored, err := s.saveWebAuthnCredential(ctx, userID, user.tenantID, session.Name, credential)
	if err != nil {
		return nil, err
	}
	info := toWebAuthnCredentialInfo(*stored)
	return &info, nil
}

// BeginWebAuthnLogin 开始无密码登录：按用户名返回 allowCredentials 断言与 session_id。
func (s *PasskeyService) BeginWebAuthnLogin(ctx context.Context, username string) (*protocol.CredentialAssertion, string, error) {
	if err := s.webAuthnReady(); err != nil {
		return nil, "", err
	}
	normalized := normalizeUsername(username)
	user, err := s.users.FindByUsername(ctx, normalized)
	if err != nil || user.Status != 1 {
		// 不区分「用户不存在」与「用户禁用」，避免用户名枚举
		return nil, "", invalidCredentials()
	}

	waUser, err := s.loadWebAuthnUser(ctx, user.ID)
	if err != nil {
		return nil, "", err
	}
	if len(waUser.credentials) == 0 {
		return nil, "", errors.New(errors.CodeWebAuthnNoCredential, "no passkey registered for this user")
	}

	assertion, session, err := s.webauthn.BeginLogin(waUser)
	if err != nil {
		return nil, "", errors.Wrap(errors.CodeInternalError, "failed to begin webauthn login", err)
	}

	sessionID, err := s.storeWebAuthnSession(ctx, webAuthnSession{
		Ceremony: webAuthnCeremonyLogin,
		UserID:   user.ID,
		Username: normalized,
		Data:     *session,
	})
	if err != nil {
		return nil, "", err
	}
	return assertion, sessionID, nil
}

// FinishWebAuthnLogin 完成无密码登录：验签通过后经 auth 的 LoginFinalizer 签发 token。
// 通行密钥是抗钓鱼的强因子，因此不再要求密码；TOTP 亦不叠加（其强度低于通行密钥）。
func (s *PasskeyService) FinishWebAuthnLogin(ctx context.Context, sessionID string, body []byte) (*contract.TokenPair, error) {
	if err := s.webAuthnReady(); err != nil {
		return nil, err
	}
	session, err := s.consumeWebAuthnSession(ctx, sessionID, webAuthnCeremonyLogin, 0)
	if err != nil {
		return nil, err
	}

	user, err := s.users.GetByID(ctx, session.UserID)
	if err != nil || user.Status != 1 {
		return nil, invalidCredentials()
	}
	if s.finalizer != nil {
		locked, remaining, err := s.finalizer.CheckLocked(ctx, user.Username)
		if err != nil {
			return nil, errors.Wrap(errors.CodeInternalError, "lockout check failed", err)
		}
		if locked {
			s.finalizer.RecordLoginHistory(ctx, user.ID, user.TenantID, user.Username, contract.LoginStatusLocked, "account locked")
			return nil, errors.AccountLocked(remaining)
		}
	}

	waUser, err := s.loadWebAuthnUser(ctx, user.ID)
	if err != nil {
		return nil, err
	}
	parsed, err := protocol.ParseCredentialRequestResponseBody(bytes.NewReader(body))
	if err != nil {
		s.recordFailure(ctx, user.ID, user.TenantID, user.Username)
		return nil, errors.Wrap(errors.CodeWebAuthnVerificationFailed, "invalid webauthn login response", err)
	}
	updated, err := s.webauthn.ValidateLogin(waUser, session.Data, parsed)
	if err != nil {
		s.recordFailure(ctx, user.ID, user.TenantID, user.Username)
		return nil, errors.Wrap(errors.CodeWebAuthnVerificationFailed, "webauthn login verification failed", err)
	}

	// 回写签名计数器与备份状态（克隆检测依赖计数器单调递增）
	if err := s.touchWebAuthnCredential(ctx, updated); err != nil {
		return nil, err
	}
	return s.finalizer.FinalizeLogin(ctx, user.ID)
}

// recordFailure 记录一次失败登录（锁定计数 + 登录历史）；finalizer 未配置时跳过。
func (s *PasskeyService) recordFailure(ctx context.Context, userID, tenantID uint64, username string) {
	if s.finalizer == nil {
		return
	}
	s.finalizer.RecordFailure(ctx, username)
	s.finalizer.RecordLoginHistory(ctx, userID, tenantID, username, contract.LoginStatusFailed, "webauthn verification failed")
}

// ListWebAuthnCredentials 列出当前用户的通行密钥（按上下文租户隔离）
func (s *PasskeyService) ListWebAuthnCredentials(ctx context.Context, userID uint64) ([]WebAuthnCredentialInfo, error) {
	if err := s.webAuthnReady(); err != nil {
		return nil, err
	}
	credentials, err := s.creds.ListByUser(ctx, tenant.FromContext(ctx), userID)
	if err != nil {
		return nil, errors.Wrap(errors.CodeInternalError, "failed to list webauthn credentials", err)
	}
	out := make([]WebAuthnCredentialInfo, 0, len(credentials))
	for _, credential := range credentials {
		out = append(out, toWebAuthnCredentialInfo(credential))
	}
	return out, nil
}

// RenameWebAuthnCredential 重命名通行密钥
func (s *PasskeyService) RenameWebAuthnCredential(ctx context.Context, userID, id uint64, name string) error {
	if err := s.webAuthnReady(); err != nil {
		return err
	}
	name = truncateString(strings.TrimSpace(name), deviceLabelMaxLen)
	if name == "" {
		return errors.New(errors.CodeInvalidParam, "name is required")
	}
	if err := s.creds.UpdateName(ctx, tenant.FromContext(ctx), userID, id, name); err != nil {
		return errors.Wrap(errors.CodeInternalError, "failed to rename webauthn credential", err)
	}
	return nil
}

// DeleteWebAuthnCredential 删除通行密钥
func (s *PasskeyService) DeleteWebAuthnCredential(ctx context.Context, userID, id uint64) error {
	if err := s.webAuthnReady(); err != nil {
		return err
	}
	if err := s.creds.Delete(ctx, tenant.FromContext(ctx), userID, id); err != nil {
		return errors.Wrap(errors.CodeInternalError, "failed to delete webauthn credential", err)
	}
	return nil
}

// webAuthnReady 检查 WebAuthn 是否已配置
func (s *PasskeyService) webAuthnReady() error {
	if s.webauthn == nil || s.creds == nil {
		return errors.New(errors.CodeInternalError, "webauthn is not configured")
	}
	return nil
}

// loadWebAuthnUser 装配库所需的用户与凭证集合
func (s *PasskeyService) loadWebAuthnUser(ctx context.Context, userID uint64) (*webAuthnUser, error) {
	user, err := s.users.GetByID(ctx, userID)
	if err != nil {
		if isNotFound(err) {
			return nil, errors.Wrap(errors.CodeUserNotFound, "user not found", err)
		}
		return nil, errors.Wrap(errors.CodeInternalError, "failed to load user", err)
	}
	stored, err := s.creds.ListByUser(ctx, 0, userID)
	if err != nil {
		return nil, errors.Wrap(errors.CodeInternalError, "failed to load webauthn credentials", err)
	}
	credentials := make([]webauthn.Credential, 0, len(stored))
	for _, item := range stored {
		credentials = append(credentials, toLibraryCredential(item))
	}
	return &webAuthnUser{id: user.ID, tenantID: user.TenantID, username: user.Username, credentials: credentials}, nil
}

// saveWebAuthnCredential 把库返回的凭证转换为领域对象并落库
func (s *PasskeyService) saveWebAuthnCredential(ctx context.Context, userID, tenantID uint64, name string, credential *webauthn.Credential) (*passkeydomain.WebAuthnCredential, error) {
	transports, err := json.Marshal(credential.Transport)
	if err != nil {
		return nil, errors.Wrap(errors.CodeInternalError, "failed to marshal credential transports", err)
	}
	stored := &passkeydomain.WebAuthnCredential{
		TenantID:          tenantID,
		UserID:            userID,
		CredentialID:      base64.RawURLEncoding.EncodeToString(credential.ID),
		PublicKey:         credential.PublicKey,
		AttestationType:   credential.AttestationType,
		AttestationFormat: credential.AttestationFormat,
		AAGUID:            hex.EncodeToString(credential.Authenticator.AAGUID),
		SignCount:         credential.Authenticator.SignCount,
		Transports:        string(transports),
		BackupEligible:    credential.Flags.BackupEligible,
		BackupState:       credential.Flags.BackupState,
		UserPresent:       credential.Flags.UserPresent,
		UserVerified:      credential.Flags.UserVerified,
		Name:              name,
	}
	if err := s.creds.Create(ctx, stored); err != nil {
		return nil, errors.Wrap(errors.CodeInternalError, "failed to store webauthn credential", err)
	}
	return stored, nil
}

// touchWebAuthnCredential 回写登录后的签名计数器与备份状态
func (s *PasskeyService) touchWebAuthnCredential(ctx context.Context, credential *webauthn.Credential) error {
	stored, err := s.creds.FindByCredentialID(ctx, base64.RawURLEncoding.EncodeToString(credential.ID))
	if err != nil {
		if isNotFound(err) {
			return errors.New(errors.CodeWebAuthnVerificationFailed, "webauthn credential not found")
		}
		return errors.Wrap(errors.CodeInternalError, "failed to load webauthn credential", err)
	}
	if err := s.creds.Touch(ctx, stored.ID, credential.Authenticator.SignCount, credential.Flags.BackupState, time.Now()); err != nil {
		return errors.Wrap(errors.CodeInternalError, "failed to update webauthn credential", err)
	}
	return nil
}

// storeWebAuthnSession 写入一次性挑战会话，返回客户端可见的 session_id
func (s *PasskeyService) storeWebAuthnSession(ctx context.Context, session webAuthnSession) (string, error) {
	payload, err := json.Marshal(session)
	if err != nil {
		return "", errors.Wrap(errors.CodeInternalError, "failed to marshal webauthn session", err)
	}
	sessionID := uuid.NewString()
	if err := s.rdb.Set(ctx, webAuthnSessionPrefix+sessionID, payload, s.webAuthnTTL()).Err(); err != nil {
		return "", errors.Wrap(errors.CodeInternalError, "failed to store webauthn session", err)
	}
	return sessionID, nil
}

// consumeWebAuthnSession 读取并删除挑战会话；校验仪式类型与（注册时的）用户归属。
func (s *PasskeyService) consumeWebAuthnSession(ctx context.Context, sessionID, ceremony string, userID uint64) (*webAuthnSession, error) {
	if sessionID == "" {
		return nil, errors.New(errors.CodeInvalidParam, "session_id is required")
	}
	if s.rdb == nil {
		return nil, errors.New(errors.CodeInternalError, "webauthn session store is not configured")
	}
	key := webAuthnSessionPrefix + sessionID
	raw, err := s.rdb.Get(ctx, key).Result()
	if err != nil {
		// 不存在或已过期一律按无效挑战处理，避免区分内部状态
		return nil, errors.New(errors.CodeWebAuthnVerificationFailed, "webauthn session not found or expired")
	}
	// 一次性：读取后立即删除，防止挑战重放
	if err := s.rdb.Del(ctx, key).Err(); err != nil {
		return nil, errors.Wrap(errors.CodeInternalError, "failed to consume webauthn session", err)
	}

	var session webAuthnSession
	if err := json.Unmarshal([]byte(raw), &session); err != nil {
		return nil, errors.Wrap(errors.CodeWebAuthnVerificationFailed, "invalid webauthn session", err)
	}
	if session.Ceremony != ceremony {
		return nil, errors.New(errors.CodeWebAuthnVerificationFailed, "webauthn session ceremony mismatch")
	}
	if userID != 0 && session.UserID != userID {
		return nil, errors.New(errors.CodeWebAuthnVerificationFailed, "webauthn session does not belong to the current user")
	}
	return &session, nil
}

// webAuthnTTL 挑战有效期（配置为空时用默认 5 分钟）
func (s *PasskeyService) webAuthnTTL() time.Duration {
	if s.sessionTTL > 0 {
		return s.sessionTTL
	}
	return defaultWebAuthnSessionTTL
}

// toLibraryCredential 领域对象 → 库凭证（登录验签需要公钥与计数器）
func toLibraryCredential(stored passkeydomain.WebAuthnCredential) webauthn.Credential {
	id, err := base64.RawURLEncoding.DecodeString(stored.CredentialID)
	if err != nil {
		// 存储的凭证 ID 一定由本服务写入；解码失败说明数据被篡改，用空 ID 让库直接拒绝
		id = nil
	}
	var transports []protocol.AuthenticatorTransport
	if stored.Transports != "" {
		_ = json.Unmarshal([]byte(stored.Transports), &transports)
	}
	aaguid, err := hex.DecodeString(stored.AAGUID)
	if err != nil {
		aaguid = nil
	}
	return webauthn.Credential{
		ID:                id,
		PublicKey:         stored.PublicKey,
		AttestationType:   stored.AttestationType,
		AttestationFormat: stored.AttestationFormat,
		Transport:         transports,
		Flags: webauthn.CredentialFlags{
			UserPresent:    stored.UserPresent,
			UserVerified:   stored.UserVerified,
			BackupEligible: stored.BackupEligible,
			BackupState:    stored.BackupState,
		},
		Authenticator: webauthn.Authenticator{
			AAGUID:    aaguid,
			SignCount: stored.SignCount,
		},
	}
}

// toWebAuthnCredentialInfo 领域对象 → 对外信息
func toWebAuthnCredentialInfo(stored passkeydomain.WebAuthnCredential) WebAuthnCredentialInfo {
	return WebAuthnCredentialInfo{
		ID:             stored.ID,
		Name:           stored.Name,
		AAGUID:         stored.AAGUID,
		BackupEligible: stored.BackupEligible,
		BackupState:    stored.BackupState,
		UserVerified:   stored.UserVerified,
		LastUsedAt:     stored.LastUsedAt,
		CreatedAt:      stored.CreatedAt,
	}
}

// isNotFound 判断 gorm 未找到记录或端口未找到。
func isNotFound(err error) bool {
	return err == gorm.ErrRecordNotFound || err == contract.ErrNotFound
}

func normalizeUsername(username string) string {
	return strings.ToLower(strings.TrimSpace(username))
}

func invalidCredentials() error {
	return errors.New(errors.CodeInvalidCredentials, "invalid credentials")
}

// truncateString 截断字符串到最大长度。
func truncateString(s string, max int) string {
	if max <= 0 || len(s) <= max {
		return s
	}
	return s[:max]
}
