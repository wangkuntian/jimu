package application

import (
	"context"
	stderrors "errors"
	"testing"
	"time"

	authdomain "jimu/internal/capabilities/auth/domain"
	"jimu/internal/capabilities/encryption"
	userdomain "jimu/internal/capabilities/user/domain"
	"jimu/internal/contract"
	"jimu/internal/kernel/auth"
	apperrors "jimu/internal/shared/errors"
	"jimu/internal/shared/pagination"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

func TestLoginHidesCredentialFailures(t *testing.T) {
	ctx := context.Background()
	jwtUtil := auth.New("01234567890123456789012345678901", "jimu", 30, 7)
	store := newFakeSessionStore()
	repo := &fakeUserRepo{users: map[string]*userdomain.User{
		"alice": userWithPassword(t, 42, "alice", "correct", 1),
	}}
	service := NewAuthService(repo, jwtUtil, store, nil, 30)

	_, missingErr := service.Login(ctx, "missing", "correct")
	_, wrongPasswordErr := service.Login(ctx, "alice", "wrong")
	if appCode(missingErr) != apperrors.CodeInvalidCredentials {
		t.Fatalf("missing user code = %d", appCode(missingErr))
	}
	if appCode(wrongPasswordErr) != apperrors.CodeInvalidCredentials {
		t.Fatalf("wrong password code = %d", appCode(wrongPasswordErr))
	}
	if missingErr.Error() != wrongPasswordErr.Error() {
		t.Fatalf("credential errors differ: %q vs %q", missingErr.Error(), wrongPasswordErr.Error())
	}
}

func TestLoginRejectsDisabledUser(t *testing.T) {
	service := newTestService(t, map[string]*userdomain.User{
		"alice": userWithPassword(t, 42, "alice", "correct", 0),
	}, newFakeSessionStore())
	_, err := service.Login(context.Background(), "alice", "correct")
	if appCode(err) != apperrors.CodeInvalidCredentials {
		t.Fatalf("code = %d, want %d", appCode(err), apperrors.CodeInvalidCredentials)
	}
}

func TestLoginCreatesRefreshSession(t *testing.T) {
	store := newFakeSessionStore()
	service := newTestService(t, map[string]*userdomain.User{
		"alice": userWithPassword(t, 42, "alice", "correct", 1),
	}, store)
	pair, err := service.Login(context.Background(), " Alice ", "correct")
	if err != nil {
		t.Fatal(err)
	}
	if len(store.created) != 1 {
		t.Fatalf("created sessions = %d, want 1", len(store.created))
	}
	accessClaims, err := service.jwtUtil.Parse(pair.AccessToken, auth.TokenTypeAccess)
	if err != nil {
		t.Fatal(err)
	}
	refreshClaims, err := service.jwtUtil.Parse(pair.RefreshToken, auth.TokenTypeRefresh)
	if err != nil {
		t.Fatal(err)
	}
	if accessClaims.SessionID != refreshClaims.SessionID || accessClaims.SessionID == "" {
		t.Fatalf("session mismatch: access=%q refresh=%q", accessClaims.SessionID, refreshClaims.SessionID)
	}
	if store.created[0].userID != 42 || store.created[0].sessionID != refreshClaims.SessionID || store.created[0].tokenID != refreshClaims.ID {
		t.Fatalf("created session = %#v refresh = %#v", store.created[0], refreshClaims)
	}
	if got := service.userRepo.(*fakeUserRepo).lookups[0]; got != "alice" {
		t.Fatalf("lookup username = %q, want alice", got)
	}
}

func TestRegisterRejectsDuplicateUsername(t *testing.T) {
	repo := &fakeUserRepo{users: map[string]*userdomain.User{
		"alice": userWithPassword(t, 42, "alice", "correct", 1),
	}}
	service := NewAuthService(repo, auth.New("01234567890123456789012345678901", "jimu", 30, 7), newFakeSessionStore(), nil, 30)

	_, err := service.Register(context.Background(), " Alice ", "secret123", "", "")
	if appCode(err) != apperrors.CodeUserExists {
		t.Fatalf("code = %d, want %d", appCode(err), apperrors.CodeUserExists)
	}
}

func TestRegisterRejectsDuplicateEmail(t *testing.T) {
	repo := &fakeUserRepo{users: map[string]*userdomain.User{
		"alice": userWithPassword(t, 42, "alice", "correct", 1),
	}}
	repo.findByEmailHash = func(_ context.Context, hash string) (*userdomain.User, error) {
		return repo.users["alice"], nil
	}
	service := NewAuthService(repo, auth.New("01234567890123456789012345678901", "jimu", 30, 7), newFakeSessionStore(), nil, 30, encryption.New("01234567890123456789012345678901"))
	_, err := service.Register(context.Background(), "bob", "secret123", "alice@example.com", "")
	if appCode(err) != apperrors.CodeUserExists {
		t.Fatalf("code = %d, want %d", appCode(err), apperrors.CodeUserExists)
	}
}

func TestRefreshRotatesOnlyRefreshTokens(t *testing.T) {
	store := newFakeSessionStore()
	service := newTestService(t, map[string]*userdomain.User{
		"alice": userWithPassword(t, 42, "alice", "correct", 1),
	}, store)
	pair, err := service.Login(context.Background(), "alice", "correct")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.Refresh(context.Background(), pair.AccessToken); appCode(err) != apperrors.CodeUnauthorized {
		t.Fatalf("access token refresh code = %d", appCode(err))
	}
	next, err := service.Refresh(context.Background(), pair.RefreshToken)
	if err != nil {
		t.Fatal(err)
	}
	if next.RefreshToken == pair.RefreshToken {
		t.Fatal("refresh token was not rotated")
	}
	if len(store.rotated) != 1 {
		t.Fatalf("rotations = %d, want 1", len(store.rotated))
	}
	if _, err := service.Refresh(context.Background(), pair.RefreshToken); appCode(err) != apperrors.CodeUnauthorized {
		t.Fatalf("reuse code = %d, want %d", appCode(err), apperrors.CodeUnauthorized)
	}
}

func TestLogoutRevokesSessions(t *testing.T) {
	store := newFakeSessionStore()
	service := newTestService(t, map[string]*userdomain.User{
		"alice": userWithPassword(t, 42, "alice", "correct", 1),
	}, store)
	pair, err := service.Login(context.Background(), "alice", "correct")
	if err != nil {
		t.Fatal(err)
	}
	claims, err := service.jwtUtil.Parse(pair.AccessToken, auth.TokenTypeAccess)
	if err != nil {
		t.Fatal(err)
	}
	if err := service.Logout(context.Background(), claims.UserID, claims.SessionID); err != nil {
		t.Fatal(err)
	}
	if len(store.revoked) != 1 || store.revoked[0] != claims.SessionID {
		t.Fatalf("revoked = %#v", store.revoked)
	}
	if err := service.LogoutAll(context.Background(), claims.UserID); err != nil {
		t.Fatal(err)
	}
	if len(store.revokedAll) != 1 || store.revokedAll[0] != claims.UserID {
		t.Fatalf("revokedAll = %#v", store.revokedAll)
	}
}

func TestRegisterProvisionedRequiresProvisioner(t *testing.T) {
	repo := &fakeUserRepo{users: map[string]*userdomain.User{}}
	service := NewAuthService(repo, auth.New("01234567890123456789012345678901", "jimu", 30, 7), newFakeSessionStore(), nil, 30)

	_, err := service.RegisterProvisioned(context.Background(), RegisterTenantRequest{
		Username: "alice", Password: "secret123", TenantName: "Acme",
	})
	if appCode(err) != apperrors.CodeInvalidParam {
		t.Fatalf("code = %d, want %d", appCode(err), apperrors.CodeInvalidParam)
	}
}

func TestRegisterProvisionedDelegatesToProvisioner(t *testing.T) {
	repo := &fakeUserRepo{users: map[string]*userdomain.User{}}
	fake := &fakeTenantProvisioner{result: &contract.ProvisionResult{}}
	service := NewAuthService(repo, auth.New("01234567890123456789012345678901", "jimu", 30, 7), newFakeSessionStore(), nil, 30, fake)

	res, err := service.RegisterProvisioned(context.Background(), RegisterTenantRequest{
		Username:   " Alice ",
		Password:   "secret123",
		Email:      "alice@example.com",
		TenantName: "Acme Inc",
		TenantCode: "acme",
	})
	if err != nil {
		t.Fatal(err)
	}
	if res != fake.result {
		t.Fatalf("result = %#v, want provisioner result", res)
	}
	if fake.params.Username != "alice" || fake.params.TenantName != "Acme Inc" || fake.params.TenantCode != "acme" {
		t.Fatalf("params = %#v", fake.params)
	}
	if fake.params.PasswordHash == "secret123" || fake.params.PasswordHash == "" {
		t.Fatalf("password must be hashed before provisioning: %q", fake.params.PasswordHash)
	}
}

func TestRegisterProvisionedRejectsMissingTenantName(t *testing.T) {
	repo := &fakeUserRepo{users: map[string]*userdomain.User{}}
	fake := &fakeTenantProvisioner{result: &contract.ProvisionResult{}}
	service := NewAuthService(repo, auth.New("01234567890123456789012345678901", "jimu", 30, 7), newFakeSessionStore(), nil, 30, fake)

	_, err := service.RegisterProvisioned(context.Background(), RegisterTenantRequest{
		Username: "alice", Password: "secret123",
	})
	if appCode(err) != apperrors.CodeInvalidParam {
		t.Fatalf("code = %d, want %d", appCode(err), apperrors.CodeInvalidParam)
	}
}

// TestLoginDegradesWithoutMFA 缺 MFAVerifier 端口（minimal profile）时登录成功、
// 不要求 TOTP、不 panic：mfa 是 auth 的软依赖，缺失只降级为「不做二次验证」。
func TestLoginDegradesWithoutMFA(t *testing.T) {
	service := newTestService(t, map[string]*userdomain.User{
		"alice": userWithPassword(t, 42, "alice", "secret1234", 1),
	}, newFakeSessionStore(), contract.MFAVerifier(nil), contract.TenantProvisioner(nil))

	pair, err := service.LoginWithTOTP(context.Background(), "alice", "secret1234", "")
	require.NoError(t, err)
	require.NotEmpty(t, pair.AccessToken)
	require.Empty(t, pair.DeviceToken, "缺 MFA 端口时不签发可信设备")
}

// TestProvisionedRegisterWithoutTenantFails 缺 TenantProvisioner 端口（minimal profile）
// 时开通式注册返回明确的 shared/errors 错误码，而不是 nil 解引用 panic。
func TestProvisionedRegisterWithoutTenantFails(t *testing.T) {
	service := newTestService(t, map[string]*userdomain.User{}, newFakeSessionStore(),
		contract.MFAVerifier(nil), contract.TenantProvisioner(nil))

	_, err := service.RegisterProvisioned(context.Background(), RegisterTenantRequest{
		Username: "owner", Password: "secret1234", TenantName: "acme",
	})
	require.Error(t, err)
	var appErr *apperrors.AppError
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, apperrors.CodeInvalidParam, appErr.Code)
}

// fakeTenantProvisioner 记录 Provision 入参，返回预置结果/错误
type fakeTenantProvisioner struct {
	params contract.ProvisionRequest
	result *contract.ProvisionResult
	err    error
}

func (f *fakeTenantProvisioner) Provision(_ context.Context, params contract.ProvisionRequest) (*contract.ProvisionResult, error) {
	f.params = params
	if f.err != nil {
		return nil, f.err
	}
	return f.result, nil
}

type fakeUserRepo struct {
	users           map[string]*userdomain.User
	lookups         []string
	created         []string
	findByEmailHash func(ctx context.Context, hash string) (*userdomain.User, error)
	updatePassword  func(ctx context.Context, id uint64, hashed string) error
}

func (r *fakeUserRepo) FindByID(_ context.Context, id uint64) (*userdomain.User, error) {
	for _, user := range r.users {
		if user.ID == id {
			return user, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

func (r *fakeUserRepo) FindByUsername(_ context.Context, username string) (*userdomain.User, error) {
	r.lookups = append(r.lookups, username)
	user, ok := r.users[username]
	if !ok {
		return nil, stderrors.New("not found")
	}
	return user, nil
}

func (r *fakeUserRepo) List(context.Context, uint64, int, int, string, string) ([]userdomain.User, int64, error) {
	return nil, 0, stderrors.New("not implemented")
}

func (r *fakeUserRepo) Create(_ context.Context, user *userdomain.User) error {
	username := normalizeUsername(user.Username)
	if _, ok := r.users[username]; ok {
		return stderrors.New("duplicate username")
	}
	user.Username = username
	r.users[username] = user
	r.created = append(r.created, username)
	return nil
}

func (r *fakeUserRepo) Update(context.Context, *userdomain.User) error {
	return nil
}

func (r *fakeUserRepo) Delete(context.Context, uint64) error {
	return nil
}

func (r *fakeUserRepo) FindByEmailHash(ctx context.Context, hash string) (*userdomain.User, error) {
	if r.findByEmailHash != nil {
		return r.findByEmailHash(ctx, hash)
	}
	return nil, gorm.ErrRecordNotFound
}

func (r *fakeUserRepo) FindByPhoneHash(context.Context, string) (*userdomain.User, error) {
	return nil, stderrors.New("not found")
}

func (r *fakeUserRepo) UpdatePassword(ctx context.Context, id uint64, hashed string) error {
	if r.updatePassword != nil {
		return r.updatePassword(ctx, id, hashed)
	}
	return nil
}

type sessionRecord struct {
	userID    uint64
	sessionID string
	tokenID   string
}

type fakeSessionStore struct {
	sessions   map[string]sessionRecord
	created    []sessionRecord
	rotated    []sessionRecord
	revoked    []string
	revokedAll []uint64
}

func newFakeSessionStore() *fakeSessionStore {
	return &fakeSessionStore{sessions: make(map[string]sessionRecord)}
}

func (s *fakeSessionStore) Create(_ context.Context, userID uint64, sessionID, tokenID string, _ time.Duration) error {
	record := sessionRecord{userID: userID, sessionID: sessionID, tokenID: tokenID}
	s.sessions[sessionID] = record
	s.created = append(s.created, record)
	return nil
}

func (s *fakeSessionStore) Rotate(_ context.Context, userID uint64, sessionID, oldTokenID, newTokenID string, _ time.Duration) error {
	record, ok := s.sessions[sessionID]
	if !ok || record.userID != userID {
		return auth.ErrSessionNotFound
	}
	if record.tokenID != oldTokenID {
		return auth.ErrTokenReuse
	}
	record.tokenID = newTokenID
	s.sessions[sessionID] = record
	s.rotated = append(s.rotated, record)
	return nil
}

func (s *fakeSessionStore) Revoke(_ context.Context, _ uint64, sessionID string) error {
	delete(s.sessions, sessionID)
	s.revoked = append(s.revoked, sessionID)
	return nil
}

func (s *fakeSessionStore) RevokeAll(_ context.Context, userID uint64) error {
	for sessionID, record := range s.sessions {
		if record.userID == userID {
			delete(s.sessions, sessionID)
		}
	}
	s.revokedAll = append(s.revokedAll, userID)
	return nil
}

func newTestService(t *testing.T, users map[string]*userdomain.User, store auth.SessionStore, deps ...interface{}) *AuthService {
	t.Helper()
	return NewAuthService(&fakeUserRepo{users: users}, auth.New("01234567890123456789012345678901", "jimu", 30, 7), store, nil, 30, deps...)
}

func userWithPassword(t *testing.T, id uint64, username, password string, status int8) *userdomain.User {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		t.Fatal(err)
	}
	return &userdomain.User{ID: id, Username: username, Password: string(hash), Status: status}
}

func appCode(err error) int {
	var appErr *apperrors.AppError
	if stderrors.As(err, &appErr) {
		return appErr.Code
	}
	return 0
}

var _ authdomain.AuthServiceInterface = (*AuthService)(nil)

type fakeLoginHistoryRepo struct {
	records []authdomain.LoginHistory
	err     error
}

func (r *fakeLoginHistoryRepo) Create(_ context.Context, record *authdomain.LoginHistory) error {
	if r.err != nil {
		return r.err
	}
	r.records = append(r.records, *record)
	return nil
}

func (r *fakeLoginHistoryRepo) ListByUser(context.Context, uint64, uint64, int, int) ([]authdomain.LoginHistory, int64, error) {
	return r.records, int64(len(r.records)), nil
}

func TestLoginRecordsHistory(t *testing.T) {
	history := &fakeLoginHistoryRepo{}
	alice := userWithPassword(t, 42, "alice", "correct", 1)
	alice.TenantID = 7
	repo := &fakeUserRepo{users: map[string]*userdomain.User{"alice": alice}}
	service := NewAuthService(repo, auth.New("01234567890123456789012345678901", "jimu", 30, 7), newFakeSessionStore(), nil, 30, history)
	ctx := contract.WithClientInfo(context.Background(), "203.0.113.7", "curl/8.0")

	// 成功
	_, err := service.LoginWithTOTP(ctx, "alice", "correct", "")
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	if len(history.records) != 1 {
		t.Fatalf("records = %d, want 1", len(history.records))
	}
	success := history.records[0]
	if success.Status != authdomain.LoginStatusSuccess || success.UserID != 42 || success.TenantID != 7 {
		t.Fatalf("unexpected success record: %+v", success)
	}
	if success.IP != "203.0.113.7" || success.UserAgent != "curl/8.0" {
		t.Fatalf("client info not recorded: %+v", success)
	}

	// 密码错误
	_, _ = service.LoginWithTOTP(ctx, "alice", "wrong", "")
	if len(history.records) != 2 {
		t.Fatalf("records = %d, want 2", len(history.records))
	}
	failure := history.records[1]
	if failure.Status != authdomain.LoginStatusFailed || failure.Reason != "invalid password" {
		t.Fatalf("unexpected failure record: %+v", failure)
	}

	// 账号不存在：user_id 为 0，仍记录用户名便于排查
	_, _ = service.LoginWithTOTP(ctx, "nobody", "x", "")
	if len(history.records) != 3 {
		t.Fatalf("records = %d, want 3", len(history.records))
	}
	missing := history.records[2]
	if missing.UserID != 0 || missing.Reason != "user not found" || missing.Username != "nobody" {
		t.Fatalf("unexpected missing-user record: %+v", missing)
	}
}

func TestLoginHistoryRecordFailureDoesNotBreakLogin(t *testing.T) {
	history := &fakeLoginHistoryRepo{err: stderrors.New("db down")}
	repo := &fakeUserRepo{users: map[string]*userdomain.User{
		"alice": userWithPassword(t, 42, "alice", "correct", 1),
	}}
	service := NewAuthService(repo, auth.New("01234567890123456789012345678901", "jimu", 30, 7), newFakeSessionStore(), nil, 30, history)

	// 审计旁路失败不应影响登录
	if _, err := service.LoginWithTOTP(context.Background(), "alice", "correct", ""); err != nil {
		t.Fatalf("login should succeed even if history write fails: %v", err)
	}
}

func TestListLoginHistory(t *testing.T) {
	history := &fakeLoginHistoryRepo{records: []authdomain.LoginHistory{{ID: 1, UserID: 42, Status: authdomain.LoginStatusSuccess}}}
	service := NewAuthService(&fakeUserRepo{}, auth.New("01234567890123456789012345678901", "jimu", 30, 7), newFakeSessionStore(), nil, 30, history)

	records, total, err := service.ListLoginHistory(context.Background(), 42, pagination.Pagination{Page: 1, PageSize: 20})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if total != 1 || len(records) != 1 {
		t.Fatalf("records = %d total = %d", len(records), total)
	}
}

func TestListLoginHistoryWithoutRepository(t *testing.T) {
	service := NewAuthService(&fakeUserRepo{}, auth.New("01234567890123456789012345678901", "jimu", 30, 7), newFakeSessionStore(), nil, 30)
	if _, _, err := service.ListLoginHistory(context.Background(), 42, pagination.Pagination{Page: 1, PageSize: 20}); err == nil {
		t.Fatal("expected error when login history repo is not configured")
	}
}

type fakePasswordHistoryRepo struct {
	hashes   []string
	added    []string
	trimKeep []int
}

func (r *fakePasswordHistoryRepo) Add(_ context.Context, _, _ uint64, hash string) error {
	r.added = append(r.added, hash)
	return nil
}

func (r *fakePasswordHistoryRepo) ListRecentHashes(context.Context, uint64, int) ([]string, error) {
	return r.hashes, nil
}

func (r *fakePasswordHistoryRepo) Trim(_ context.Context, _ uint64, keep int) error {
	r.trimKeep = append(r.trimKeep, keep)
	return nil
}

// newPasswordHistoryService 构造带密码历史仓储的 AuthService（复用重置流程所需依赖）
func newPasswordHistoryService(t *testing.T, repo *fakeUserRepo, history authdomain.PasswordHistoryRepository, count int) *AuthService {
	t.Helper()
	_, rclient := newResetRedis(t)
	resetStore := NewResetStore(rclient, 15*time.Minute)
	svc := NewAuthService(repo, auth.New("01234567890123456789012345678901", "jimu", 30, 7), newFakeSessionStore(), nil, 30,
		encryption.New(resetTestKey), resetStore, &fakeDispatcher{}, history, WithPasswordHistory(count))
	svc.resetGen = func() string { return "123456" }
	return svc
}

func newResetUserRepo(t *testing.T, password string, id uint64) *fakeUserRepo {
	t.Helper()
	user := userWithPassword(t, id, "alice", password, 1)
	repo := &fakeUserRepo{users: map[string]*userdomain.User{"alice": user}}
	repo.findByEmailHash = func(context.Context, string) (*userdomain.User, error) {
		return user, nil
	}
	return repo
}

func TestResetPasswordRejectsCurrentPassword(t *testing.T) {
	ctx := context.Background()
	repo := newResetUserRepo(t, "correct", 42)
	svc := newPasswordHistoryService(t, repo, &fakePasswordHistoryRepo{}, 5)

	require.NoError(t, svc.ForgotPassword(ctx, "alice@example.com"))
	err := svc.ResetPassword(ctx, "alice@example.com", "123456", "correct")
	assert.Equal(t, apperrors.CodePasswordReused, appCode(err), "不得改回当前密码")
}

func TestResetPasswordRejectsHistoryPassword(t *testing.T) {
	ctx := context.Background()
	oldHash, err := bcrypt.GenerateFromPassword([]byte("old-secret"), bcrypt.DefaultCost)
	require.NoError(t, err)

	repo := newResetUserRepo(t, "correct", 42)
	history := &fakePasswordHistoryRepo{hashes: []string{string(oldHash)}}
	svc := newPasswordHistoryService(t, repo, history, 5)

	require.NoError(t, svc.ForgotPassword(ctx, "alice@example.com"))
	err = svc.ResetPassword(ctx, "alice@example.com", "123456", "old-secret")
	assert.Equal(t, apperrors.CodePasswordReused, appCode(err), "不得复用历史密码")
}

func TestResetPasswordRecordsPreviousHash(t *testing.T) {
	ctx := context.Background()
	repo := newResetUserRepo(t, "correct", 42)
	previousHash := repo.users["alice"].Password
	history := &fakePasswordHistoryRepo{}
	svc := newPasswordHistoryService(t, repo, history, 5)

	require.NoError(t, svc.ForgotPassword(ctx, "alice@example.com"))
	require.NoError(t, svc.ResetPassword(ctx, "alice@example.com", "123456", "brand-new-pass"))

	require.Len(t, history.added, 1)
	assert.Equal(t, previousHash, history.added[0], "应记录被替换的旧密码哈希")
	assert.Equal(t, []int{5}, history.trimKeep, "应按配置条数裁剪历史")
}

func TestResetPasswordAllowsReuseWhenDisabled(t *testing.T) {
	ctx := context.Background()
	repo := newResetUserRepo(t, "correct", 42)
	history := &fakePasswordHistoryRepo{}
	// count=0 表示未启用防复用策略
	svc := newPasswordHistoryService(t, repo, history, 0)

	require.NoError(t, svc.ForgotPassword(ctx, "alice@example.com"))
	require.NoError(t, svc.ResetPassword(ctx, "alice@example.com", "123456", "correct"))
	assert.Empty(t, history.added, "未启用时不应写入历史")
}
