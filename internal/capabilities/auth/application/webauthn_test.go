package application

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"testing"
	"time"

	authdomain "jimu/internal/capabilities/auth/domain"
	userdomain "jimu/internal/capabilities/user/domain"
	"jimu/internal/kernel/auth"
	"jimu/internal/kernel/tenant"
	apperrors "jimu/internal/shared/errors"

	"github.com/alicebob/miniredis/v2"
	"github.com/fxamacker/cbor/v2"
	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

const (
	testRPID     = "localhost"
	testRPOrigin = "http://localhost:8080"
)

// --- 内存凭证仓储 ---

type fakeWebAuthnRepo struct {
	items  []*authdomain.WebAuthnCredential
	nextID uint64
}

func newFakeWebAuthnRepo() *fakeWebAuthnRepo { return &fakeWebAuthnRepo{nextID: 1} }

func (r *fakeWebAuthnRepo) Create(_ context.Context, credential *authdomain.WebAuthnCredential) error {
	credential.ID = r.nextID
	r.nextID++
	r.items = append(r.items, credential)
	return nil
}

func (r *fakeWebAuthnRepo) FindByCredentialID(_ context.Context, credentialID string) (*authdomain.WebAuthnCredential, error) {
	for _, item := range r.items {
		if item.CredentialID == credentialID {
			return item, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

// ListByUser 与真实仓储一致：按 id 倒序（最新注册的在前）
func (r *fakeWebAuthnRepo) ListByUser(_ context.Context, tenantID, userID uint64) ([]authdomain.WebAuthnCredential, error) {
	var out []authdomain.WebAuthnCredential
	for i := len(r.items) - 1; i >= 0; i-- {
		item := r.items[i]
		if item.UserID != userID {
			continue
		}
		if tenantID != 0 && item.TenantID != tenantID {
			continue
		}
		out = append(out, *item)
	}
	return out, nil
}

func (r *fakeWebAuthnRepo) CountByUser(_ context.Context, userID uint64) (int64, error) {
	var n int64
	for _, item := range r.items {
		if item.UserID == userID {
			n++
		}
	}
	return n, nil
}

func (r *fakeWebAuthnRepo) Touch(_ context.Context, id uint64, signCount uint32, backupState bool, usedAt time.Time) error {
	for _, item := range r.items {
		if item.ID == id {
			item.SignCount = signCount
			item.BackupState = backupState
			item.LastUsedAt = &usedAt
		}
	}
	return nil
}

func (r *fakeWebAuthnRepo) UpdateName(_ context.Context, tenantID, userID, id uint64, name string) error {
	for _, item := range r.items {
		if item.ID == id && item.UserID == userID && (tenantID == 0 || item.TenantID == tenantID) {
			item.Name = name
		}
	}
	return nil
}

func (r *fakeWebAuthnRepo) Delete(_ context.Context, tenantID, userID, id uint64) error {
	kept := r.items[:0]
	for _, item := range r.items {
		if item.ID == id && item.UserID == userID && (tenantID == 0 || item.TenantID == tenantID) {
			continue
		}
		kept = append(kept, item)
	}
	r.items = kept
	return nil
}

// --- 虚拟认证器 ---

// virtualAuthenticator 用真实 ES256 密钥生成注册证明与登录断言，端到端验证 WebAuthn 流程。
// 证明格式用 none（无需 attestation 签名），断言用私钥签 authData||sha256(clientDataJSON)。
type virtualAuthenticator struct {
	key       *ecdsa.PrivateKey
	credID    []byte
	signCount uint32
}

func newVirtualAuthenticator(t *testing.T) *virtualAuthenticator {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	credID := make([]byte, 32)
	_, err = rand.Read(credID)
	require.NoError(t, err)
	return &virtualAuthenticator{key: key, credID: credID}
}

// coseKey 编码 COSE_Key（EC2 / P-256 / ES256）
func (a *virtualAuthenticator) coseKey(t *testing.T) []byte {
	t.Helper()
	// PublicKey.Bytes() 为未压缩点：0x04 || X(32) || Y(32)
	raw, err := a.key.PublicKey.Bytes()
	require.NoError(t, err)
	require.Len(t, raw, 65)
	encoded, err := cbor.Marshal(map[int]interface{}{
		1:  2, // kty: EC2
		3:  -7,
		-1: 1, // crv: P-256
		-2: raw[1:33],
		-3: raw[33:65],
	})
	require.NoError(t, err)
	return encoded
}

// authData 组装 authenticatorData；withAttestedCredential 为真时附上凭证公钥（注册用）
func (a *virtualAuthenticator) authData(t *testing.T, withAttestedCredential bool) []byte {
	t.Helper()
	rpIDHash := sha256.Sum256([]byte(testRPID))
	// UP + UV（+ AT 表示含 attestedCredentialData）
	flags := byte(protocol.FlagUserPresent | protocol.FlagUserVerified)
	var buf []byte
	buf = append(buf, rpIDHash[:]...)
	if withAttestedCredential {
		flags |= byte(protocol.FlagAttestedCredentialData)
	}
	buf = append(buf, flags)
	count := make([]byte, 4)
	binary.BigEndian.PutUint32(count, a.signCount)
	buf = append(buf, count...)
	if !withAttestedCredential {
		return buf
	}
	buf = append(buf, make([]byte, 16)...) // AAGUID：全 0
	credIDLen := make([]byte, 2)
	binary.BigEndian.PutUint16(credIDLen, uint16(len(a.credID)))
	buf = append(buf, credIDLen...)
	buf = append(buf, a.credID...)
	buf = append(buf, a.coseKey(t)...)
	return buf
}

func clientDataJSON(t *testing.T, ceremony, challenge, origin string) []byte {
	t.Helper()
	encoded, err := json.Marshal(map[string]interface{}{
		"type":        ceremony,
		"challenge":   challenge,
		"origin":      origin,
		"crossOrigin": false,
	})
	require.NoError(t, err)
	return encoded
}

// registrationResponse 构造 navigator.credentials.create() 的返回 JSON
func (a *virtualAuthenticator) registrationResponse(t *testing.T, challenge string) []byte {
	t.Helper()
	attestationObject, err := cbor.Marshal(map[string]interface{}{
		"fmt":      "none",
		"attStmt":  map[string]interface{}{},
		"authData": a.authData(t, true),
	})
	require.NoError(t, err)
	response, err := json.Marshal(map[string]interface{}{
		"id":    base64.RawURLEncoding.EncodeToString(a.credID),
		"rawId": base64.RawURLEncoding.EncodeToString(a.credID),
		"type":  "public-key",
		"response": map[string]interface{}{
			"clientDataJSON":    base64.RawURLEncoding.EncodeToString(clientDataJSON(t, "webauthn.create", challenge, testRPOrigin)),
			"attestationObject": base64.RawURLEncoding.EncodeToString(attestationObject),
		},
		"clientExtensionResults": map[string]interface{}{},
	})
	require.NoError(t, err)
	return response
}

// assertionResponse 构造 navigator.credentials.get() 的返回 JSON
func (a *virtualAuthenticator) assertionResponse(t *testing.T, challenge string) []byte {
	t.Helper()
	a.signCount++
	authData := a.authData(t, false)
	clientData := clientDataJSON(t, "webauthn.get", challenge, testRPOrigin)

	digest := sha256.Sum256(clientData)
	signed := append(append([]byte{}, authData...), digest[:]...)
	signature, err := ecdsa.SignASN1(rand.Reader, a.key, sha256Sum(signed))
	require.NoError(t, err)

	response, err := json.Marshal(map[string]interface{}{
		"id":    base64.RawURLEncoding.EncodeToString(a.credID),
		"rawId": base64.RawURLEncoding.EncodeToString(a.credID),
		"type":  "public-key",
		"response": map[string]interface{}{
			"clientDataJSON":    base64.RawURLEncoding.EncodeToString(clientData),
			"authenticatorData": base64.RawURLEncoding.EncodeToString(authData),
			"signature":         base64.RawURLEncoding.EncodeToString(signature),
		},
		"clientExtensionResults": map[string]interface{}{},
	})
	require.NoError(t, err)
	return response
}

func sha256Sum(data []byte) []byte {
	sum := sha256.Sum256(data)
	return sum[:]
}

// --- 服务装配 ---

type webAuthnHarness struct {
	service *AuthService
	repo    *fakeUserRepo
	creds   *fakeWebAuthnRepo
	user    *userdomain.User
}

// newWebAuthnHarness 装配启用 WebAuthn 的 AuthService（miniredis 存挑战）
func newWebAuthnHarness(t *testing.T) *webAuthnHarness {
	t.Helper()
	mrs, err := miniredis.Run()
	require.NoError(t, err)
	t.Cleanup(mrs.Close)
	rdb := redis.NewClient(&redis.Options{Addr: mrs.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })

	handle, err := webauthn.New(&webauthn.Config{
		RPDisplayName: "Jimu Test",
		RPID:          testRPID,
		RPOrigins:     []string{testRPOrigin},
	})
	require.NoError(t, err)

	repo := &fakeUserRepo{users: map[string]*userdomain.User{}}
	alice := userWithPassword(t, 42, "alice", "correct", 1)
	alice.TenantID = 1
	repo.users["alice"] = alice

	creds := newFakeWebAuthnRepo()
	svc := NewAuthService(repo, auth.New("01234567890123456789012345678901", "jimu", 30, 7),
		newFakeSessionStore(), nil, 30, rdb, handle, creds, WithWebAuthnSessionTTL(2*time.Minute))
	return &webAuthnHarness{service: svc, repo: repo, creds: creds, user: alice}
}

// registerCredential 走完注册流程并返回虚拟认证器
func (h *webAuthnHarness) registerCredential(t *testing.T, name string) *virtualAuthenticator {
	t.Helper()
	ctx := tenant.WithTenant(context.Background(), 1)
	creation, sessionID, err := h.service.BeginWebAuthnRegistration(ctx, h.user.ID, name)
	require.NoError(t, err)
	require.NotEmpty(t, sessionID)
	require.NotNil(t, creation)

	authenticator := newVirtualAuthenticator(t)
	info, err := h.service.FinishWebAuthnRegistration(ctx, h.user.ID, sessionID,
		authenticator.registrationResponse(t, creation.Response.Challenge.String()))
	require.NoError(t, err)
	assert.NotZero(t, info.ID)
	return authenticator
}

func TestWebAuthnRegisterAndLogin(t *testing.T) {
	h := newWebAuthnHarness(t)
	authenticator := h.registerCredential(t, "MacBook")

	// 凭证已落库且不含明文标识
	require.Len(t, h.creds.items, 1)
	stored := h.creds.items[0]
	assert.Equal(t, uint64(42), stored.UserID)
	assert.Equal(t, uint64(1), stored.TenantID)
	assert.Equal(t, base64.RawURLEncoding.EncodeToString(authenticator.credID), stored.CredentialID)
	assert.Equal(t, "MacBook", stored.Name)
	assert.NotEmpty(t, stored.PublicKey)
	assert.True(t, stored.UserVerified, "注册时 UV=1 应被记录")

	// 无密码登录
	ctx := context.Background()
	assertion, sessionID, err := h.service.BeginWebAuthnLogin(ctx, "Alice") // 用户名大小写归一化
	require.NoError(t, err)
	require.NotEmpty(t, sessionID)
	require.NotEmpty(t, assertion.Response.Challenge.String())

	pair, err := h.service.FinishWebAuthnLogin(ctx, sessionID, authenticator.assertionResponse(t, assertion.Response.Challenge.String()))
	require.NoError(t, err)
	assert.NotEmpty(t, pair.AccessToken)
	assert.NotEmpty(t, pair.RefreshToken)

	// 签名计数器与最近使用时间已回写
	assert.Equal(t, uint32(1), h.creds.items[0].SignCount)
	assert.NotNil(t, h.creds.items[0].LastUsedAt)
}

func TestWebAuthnSessionIsSingleUse(t *testing.T) {
	h := newWebAuthnHarness(t)
	authenticator := h.registerCredential(t, "")

	ctx := context.Background()
	assertion, sessionID, err := h.service.BeginWebAuthnLogin(ctx, "alice")
	require.NoError(t, err)

	body := authenticator.assertionResponse(t, assertion.Response.Challenge.String())
	_, err = h.service.FinishWebAuthnLogin(ctx, sessionID, body)
	require.NoError(t, err)

	// 挑战一次性：同一 session_id 再次提交直接拒绝
	_, err = h.service.FinishWebAuthnLogin(ctx, sessionID, body)
	assert.Equal(t, apperrors.CodeWebAuthnVerificationFailed, appCode(err))
}

func TestWebAuthnRejectsForeignOrWrongCeremonySession(t *testing.T) {
	h := newWebAuthnHarness(t)
	authenticator := h.registerCredential(t, "")

	ctx := tenant.WithTenant(context.Background(), 1)

	// 注册会话不能用于登录
	creation, registerSession, err := h.service.BeginWebAuthnRegistration(ctx, h.user.ID, "")
	require.NoError(t, err)
	_, err = h.service.FinishWebAuthnLogin(context.Background(), registerSession,
		authenticator.assertionResponse(t, creation.Response.Challenge.String()))
	assert.Equal(t, apperrors.CodeWebAuthnVerificationFailed, appCode(err))

	// 会话归属其他用户时注册被拒
	_, loginSession, err := h.service.BeginWebAuthnLogin(context.Background(), "alice")
	require.NoError(t, err)
	_, err = h.service.FinishWebAuthnRegistration(ctx, 99, loginSession, []byte("{}"))
	assert.Equal(t, apperrors.CodeWebAuthnVerificationFailed, appCode(err))

	// 未知 / 过期 session
	_, err = h.service.FinishWebAuthnRegistration(ctx, h.user.ID, "not-exist", []byte("{}"))
	assert.Equal(t, apperrors.CodeWebAuthnVerificationFailed, appCode(err))
}

func TestWebAuthnRejectsTamperedAssertion(t *testing.T) {
	h := newWebAuthnHarness(t)
	authenticator := h.registerCredential(t, "")
	ctx := context.Background()

	assertion, sessionID, err := h.service.BeginWebAuthnLogin(ctx, "alice")
	require.NoError(t, err)

	// 用另一把密钥签名（模拟伪造/密钥不匹配）；凭证 ID 相同，但签名对应的公钥不匹配
	other := newVirtualAuthenticator(t)
	other.credID = authenticator.credID
	forged := other.assertionResponse(t, assertion.Response.Challenge.String())

	_, err = h.service.FinishWebAuthnLogin(ctx, sessionID, forged)
	assert.Equal(t, apperrors.CodeWebAuthnVerificationFailed, appCode(err))

	// 正确密钥的断言仍可用：换新会话并针对新挑战重新签名
	second, secondSession, err := h.service.BeginWebAuthnLogin(ctx, "alice")
	require.NoError(t, err)
	_, err = h.service.FinishWebAuthnLogin(ctx, secondSession,
		authenticator.assertionResponse(t, second.Response.Challenge.String()))
	assert.NoError(t, err)
}

func TestWebAuthnBeginLoginWithoutCredential(t *testing.T) {
	h := newWebAuthnHarness(t)
	_, _, err := h.service.BeginWebAuthnLogin(context.Background(), "alice")
	assert.Equal(t, apperrors.CodeWebAuthnNoCredential, appCode(err))

	// 用户不存在与用户禁用都返回认证失败（不泄漏账号状态）
	_, _, err = h.service.BeginWebAuthnLogin(context.Background(), "nobody")
	assert.Equal(t, apperrors.CodeInvalidCredentials, appCode(err))

	disabled := userWithPassword(t, 43, "bob", "correct", 0)
	h.repo.users["bob"] = disabled
	_, _, err = h.service.BeginWebAuthnLogin(context.Background(), "bob")
	assert.Equal(t, apperrors.CodeInvalidCredentials, appCode(err))
}

func TestWebAuthnCredentialManagement(t *testing.T) {
	h := newWebAuthnHarness(t)
	h.registerCredential(t, "Phone")
	h.registerCredential(t, "Laptop")
	require.Len(t, h.creds.items, 2)

	ctx := tenant.WithTenant(context.Background(), 1)
	list, err := h.service.ListWebAuthnCredentials(ctx, h.user.ID)
	require.NoError(t, err)
	assert.Len(t, list, 2)

	// 跨租户不可见
	other, err := h.service.ListWebAuthnCredentials(tenant.WithTenant(context.Background(), 2), h.user.ID)
	require.NoError(t, err)
	assert.Empty(t, other)

	require.NoError(t, h.service.RenameWebAuthnCredential(ctx, h.user.ID, list[0].ID, "会议室电脑"))
	assert.Equal(t, "会议室电脑", h.creds.items[1].Name)

	// 空名称非法
	err = h.service.RenameWebAuthnCredential(ctx, h.user.ID, list[0].ID, "   ")
	assert.Equal(t, apperrors.CodeInvalidParam, appCode(err))

	require.NoError(t, h.service.DeleteWebAuthnCredential(ctx, h.user.ID, list[0].ID))
	assert.Len(t, h.creds.items, 1)

	// 越权删除他人凭证无效
	require.NoError(t, h.service.DeleteWebAuthnCredential(ctx, 99, h.creds.items[0].ID))
	assert.Len(t, h.creds.items, 1)
}

func TestWebAuthnDisabledWhenNotConfigured(t *testing.T) {
	svc := newTestService(t, map[string]*userdomain.User{}, newFakeSessionStore())
	ctx := context.Background()

	_, _, err := svc.BeginWebAuthnRegistration(ctx, 42, "")
	assert.Equal(t, apperrors.CodeInternalError, appCode(err))

	_, _, err = svc.BeginWebAuthnLogin(ctx, "alice")
	assert.Equal(t, apperrors.CodeInternalError, appCode(err))

	_, err = svc.ListWebAuthnCredentials(ctx, 42)
	assert.Equal(t, apperrors.CodeInternalError, appCode(err))
}

func TestWebAuthnRegistrationRejectsUnparsableBody(t *testing.T) {
	h := newWebAuthnHarness(t)
	ctx := tenant.WithTenant(context.Background(), 1)

	_, sessionID, err := h.service.BeginWebAuthnRegistration(ctx, h.user.ID, "")
	require.NoError(t, err)

	_, err = h.service.FinishWebAuthnRegistration(ctx, h.user.ID, sessionID, []byte("not json"))
	assert.Equal(t, apperrors.CodeWebAuthnVerificationFailed, appCode(err))
	assert.Empty(t, h.creds.items)
}
