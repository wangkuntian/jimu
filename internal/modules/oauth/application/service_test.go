// internal/modules/oauth/application/service_test.go
package application

import (
	"context"
	stderrors "errors"
	"strings"
	"testing"
	"time"

	authdomain "jimu/internal/modules/auth/domain"
	oauthdomain "jimu/internal/modules/oauth/domain"
	userdomain "jimu/internal/modules/user/domain"
	"jimu/internal/platform/auth"
	oauthplatform "jimu/internal/platform/oauth"
	apperrors "jimu/internal/shared/errors"

	"github.com/alicebob/miniredis/v2"
	"github.com/glebarez/sqlite"
	"github.com/golang-jwt/jwt/v5"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

const testJWTSecret = "01234567890123456789012345678901"

// fakeProvider 实现 platform oauth.Provider
type fakeProvider struct {
	name    string
	authURL string
	info    *oauthplatform.UserInfo
	err     error
}

func (p *fakeProvider) Name() string { return p.name }
func (p *fakeProvider) AuthURL(state string) string {
	return p.authURL + "?state=" + state
}
func (p *fakeProvider) Exchange(context.Context, string) (*oauthplatform.UserInfo, error) {
	return p.info, p.err
}

// fakeBindingRepo 顺序消费预置结果的绑定仓储
type fakeBindingRepo struct {
	results []struct {
		binding *oauthdomain.OAuthBinding
		err     error
	}
	created []*oauthdomain.OAuthBinding
}

func newRepoResult(binding *oauthdomain.OAuthBinding, err error) struct {
	binding *oauthdomain.OAuthBinding
	err     error
} {
	return struct {
		binding *oauthdomain.OAuthBinding
		err     error
	}{binding: binding, err: err}
}

func (r *fakeBindingRepo) FindByProviderSubject(context.Context, string, string) (*oauthdomain.OAuthBinding, error) {
	if len(r.results) == 0 {
		return nil, nil
	}
	res := r.results[0]
	r.results = r.results[1:]
	return res.binding, res.err
}

func (r *fakeBindingRepo) Create(_ context.Context, b *oauthdomain.OAuthBinding) error {
	r.created = append(r.created, b)
	return nil
}

// fakeSessionStore 记录会话创建的 SessionStore
type fakeSessionStore struct {
	created []sessionRecord
	err     error
}

type sessionRecord struct {
	userID    uint64
	sessionID string
	tokenID   string
	ttl       time.Duration
}

func (s *fakeSessionStore) Create(_ context.Context, userID uint64, sessionID, tokenID string, ttl time.Duration) error {
	s.created = append(s.created, sessionRecord{userID: userID, sessionID: sessionID, tokenID: tokenID, ttl: ttl})
	return s.err
}
func (s *fakeSessionStore) Rotate(context.Context, uint64, string, string, string, time.Duration) error {
	return nil
}
func (s *fakeSessionStore) Revoke(context.Context, uint64, string) error { return nil }
func (s *fakeSessionStore) RevokeAll(context.Context, uint64) error      { return nil }

func newRedisClient(t *testing.T) (*miniredis.Miniredis, *redis.Client) {
	t.Helper()
	mrs, err := miniredis.Run()
	require.NoError(t, err)
	t.Cleanup(mrs.Close)
	client := redis.NewClient(&redis.Options{Addr: mrs.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	return mrs, client
}

func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: gormlogger.Default.LogMode(gormlogger.Silent)})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&userdomain.User{}, &oauthdomain.OAuthBinding{}))
	return db
}

func newService(t *testing.T, repo oauthdomain.BindingRepository, providers map[string]oauthplatform.Provider, rdb *redis.Client, db *gorm.DB, sessions auth.SessionStore) *OAuthService {
	t.Helper()
	return NewOAuthService(repo, auth.New(testJWTSecret, "jimu", 30, 7), sessions, providers, rdb, db, 30)
}

func githubProviders() map[string]oauthplatform.Provider {
	return map[string]oauthplatform.Provider{
		"github": &fakeProvider{name: "github", authURL: "https://github.com/login/oauth/authorize", info: &oauthplatform.UserInfo{Subject: "sub123", Email: "a@b.c", Name: "Alice"}},
	}
}

// dupSubjectProviders 返回 Subject 为 "dup" 的提供商，用于用户名冲突场景
func dupSubjectProviders() map[string]oauthplatform.Provider {
	return map[string]oauthplatform.Provider{
		"github": &fakeProvider{name: "github", authURL: "https://github.com/login/oauth/authorize", info: &oauthplatform.UserInfo{Subject: "dup", Email: "a@b.c"}},
	}
}

func TestAuthURLUnknownProvider(t *testing.T) {
	_, client := newRedisClient(t)
	svc := newService(t, &fakeBindingRepo{}, githubProviders(), client, nil, &fakeSessionStore{})

	_, err := svc.AuthURL(context.Background(), "wechat", "state-1")
	require.Error(t, err)
	assert.True(t, apperrors.IsCode(err, apperrors.CodeOAuthProviderNotFound))
}

func TestAuthURLOK(t *testing.T) {
	mrs, client := newRedisClient(t)
	svc := newService(t, &fakeBindingRepo{}, githubProviders(), client, nil, &fakeSessionStore{})

	url, err := svc.AuthURL(context.Background(), "github", "state-1")
	require.NoError(t, err)
	assert.Equal(t, "https://github.com/login/oauth/authorize?state=state-1", url)

	val, err := mrs.Get(oauthStateKey("state-1"))
	require.NoError(t, err)
	assert.Equal(t, "github", val)
}

func TestAuthURLRedisError(t *testing.T) {
	mrs, client := newRedisClient(t)
	mrs.SetError("boom")
	svc := newService(t, &fakeBindingRepo{}, githubProviders(), client, nil, &fakeSessionStore{})

	_, err := svc.AuthURL(context.Background(), "github", "state-1")
	require.Error(t, err)
	assert.True(t, apperrors.IsCode(err, apperrors.CodeInternalError))
}

func TestBeginLoginReturnsStateAndURL(t *testing.T) {
	mrs, client := newRedisClient(t)
	svc := newService(t, &fakeBindingRepo{}, githubProviders(), client, nil, &fakeSessionStore{})

	url, state, err := svc.BeginLogin(context.Background(), "github")
	require.NoError(t, err)
	assert.NotEmpty(t, url)
	assert.NotEmpty(t, state)
	assert.True(t, mrs.Exists(oauthStateKey(state)))
}

func TestBeginLoginPropagatesError(t *testing.T) {
	_, client := newRedisClient(t)
	svc := newService(t, &fakeBindingRepo{}, githubProviders(), client, nil, &fakeSessionStore{})

	_, _, err := svc.BeginLogin(context.Background(), "wechat")
	require.Error(t, err)
	assert.True(t, apperrors.IsCode(err, apperrors.CodeOAuthProviderNotFound))
}

func TestConsumeStateEmpty(t *testing.T) {
	_, client := newRedisClient(t)
	svc := newService(t, &fakeBindingRepo{}, githubProviders(), client, nil, &fakeSessionStore{})

	err := svc.consumeState(context.Background(), "", "github")
	require.Error(t, err)
	assert.True(t, apperrors.IsCode(err, apperrors.CodeInvalidParam))
}

func TestConsumeStateNotStored(t *testing.T) {
	_, client := newRedisClient(t)
	svc := newService(t, &fakeBindingRepo{}, githubProviders(), client, nil, &fakeSessionStore{})

	err := svc.consumeState(context.Background(), "missing", "github")
	require.Error(t, err)
	assert.True(t, apperrors.IsCode(err, apperrors.CodeInvalidParam))
}

func TestConsumeStateMismatchProvider(t *testing.T) {
	mrs, client := newRedisClient(t)
	require.NoError(t, mrs.Set(oauthStateKey("state-1"), "github"))
	svc := newService(t, &fakeBindingRepo{}, githubProviders(), client, nil, &fakeSessionStore{})

	err := svc.consumeState(context.Background(), "state-1", "google")
	require.Error(t, err)
	assert.True(t, apperrors.IsCode(err, apperrors.CodeInvalidParam))
}

func TestConsumeStateOKDeletesState(t *testing.T) {
	mrs, client := newRedisClient(t)
	require.NoError(t, mrs.Set(oauthStateKey("state-1"), "github"))
	svc := newService(t, &fakeBindingRepo{}, githubProviders(), client, nil, &fakeSessionStore{})

	err := svc.consumeState(context.Background(), "state-1", "github")
	require.NoError(t, err)
	assert.False(t, mrs.Exists(oauthStateKey("state-1")), "state 应被一次性消费删除")
}

func TestConsumeStateReadError(t *testing.T) {
	mrs, client := newRedisClient(t)
	mrs.SetError("boom")
	svc := newService(t, &fakeBindingRepo{}, githubProviders(), client, nil, &fakeSessionStore{})

	// providerName 为空使 val(空串)==providerName 绕过不匹配分支，进入内部错误分支
	err := svc.consumeState(context.Background(), "state-1", "")
	require.Error(t, err)
	assert.True(t, apperrors.IsCode(err, apperrors.CodeInternalError))
}

func TestLoginRejectsInvalidState(t *testing.T) {
	_, client := newRedisClient(t)
	svc := newService(t, &fakeBindingRepo{}, githubProviders(), client, nil, &fakeSessionStore{})

	_, err := svc.Login(context.Background(), "github", "code", "wrong-state")
	require.Error(t, err)
	assert.True(t, apperrors.IsCode(err, apperrors.CodeInvalidParam))
}

func TestLoginUnknownProvider(t *testing.T) {
	mrs, client := newRedisClient(t)
	require.NoError(t, mrs.Set(oauthStateKey("state-1"), "wechat"))
	svc := newService(t, &fakeBindingRepo{}, githubProviders(), client, nil, &fakeSessionStore{})

	_, err := svc.Login(context.Background(), "wechat", "code", "state-1")
	require.Error(t, err)
	assert.True(t, apperrors.IsCode(err, apperrors.CodeOAuthProviderNotFound))
}

func TestLoginExchangeFailure(t *testing.T) {
	mrs, client := newRedisClient(t)
	require.NoError(t, mrs.Set(oauthStateKey("state-1"), "github"))
	providers := map[string]oauthplatform.Provider{
		"github": &fakeProvider{name: "github", err: stderrors.New("upstream 500")},
	}
	svc := newService(t, &fakeBindingRepo{}, providers, client, nil, &fakeSessionStore{})

	_, err := svc.Login(context.Background(), "github", "code", "state-1")
	require.Error(t, err)
	assert.True(t, apperrors.IsCode(err, apperrors.CodeInternalError))
}

func TestLoginExistingBinding(t *testing.T) {
	mrs, client := newRedisClient(t)
	require.NoError(t, mrs.Set(oauthStateKey("state-1"), "github"))
	sessions := &fakeSessionStore{}
	repo := &fakeBindingRepo{results: []struct {
		binding *oauthdomain.OAuthBinding
		err     error
	}{
		newRepoResult(&oauthdomain.OAuthBinding{UserID: 42, Provider: "github", Subject: "sub123"}, nil),
	}}
	svc := newService(t, repo, githubProviders(), client, nil, sessions)

	pair, err := svc.Login(context.Background(), "github", "code", "state-1")
	require.NoError(t, err)
	assert.Equal(t, 30*60, pair.ExpiresIn)
	require.Len(t, sessions.created, 1)
	assert.Equal(t, uint64(42), sessions.created[0].userID)

	refreshClaims, err := svc.jwtUtil.Parse(pair.RefreshToken, auth.TokenTypeRefresh)
	require.NoError(t, err)
	assert.Equal(t, uint64(42), refreshClaims.UserID)
	assert.Equal(t, sessions.created[0].sessionID, refreshClaims.SessionID)
	assert.Equal(t, sessions.created[0].tokenID, refreshClaims.ID)
}

func TestLoginCreatesUserAndBinding(t *testing.T) {
	mrs, client := newRedisClient(t)
	require.NoError(t, mrs.Set(oauthStateKey("state-1"), "github"))
	db := newTestDB(t)
	repo := &fakeBindingRepo{results: []struct {
		binding *oauthdomain.OAuthBinding
		err     error
	}{newRepoResult(nil, gorm.ErrRecordNotFound)}}
	svc := newService(t, repo, githubProviders(), client, db, &fakeSessionStore{})

	pair, err := svc.Login(context.Background(), "github", "code", "state-1")
	require.NoError(t, err)
	require.NotEmpty(t, pair.AccessToken)

	var user userdomain.User
	require.NoError(t, db.Where("username = ?", "github_sub123").First(&user).Error)
	assert.Equal(t, int8(1), user.Status)
	assert.True(t, user.Password != "" && user.Password != user.Username)

	var binding oauthdomain.OAuthBinding
	require.NoError(t, db.Where("provider = ? AND subject = ?", "github", "sub123").First(&binding).Error)
	assert.Equal(t, user.ID, binding.UserID)
}

func TestLoginCreateConflictFallsBackToExistingBinding(t *testing.T) {
	mrs, client := newRedisClient(t)
	require.NoError(t, mrs.Set(oauthStateKey("state-1"), "github"))
	db := newTestDB(t)
	// 预置同名用户，使 createUserWithBinding 的 INSERT 触发唯一约束失败
	require.NoError(t, db.Create(&userdomain.User{Username: "github_dup", Password: "x", Status: 1}).Error)

	repo := &fakeBindingRepo{results: []struct {
		binding *oauthdomain.OAuthBinding
		err     error
	}{
		newRepoResult(nil, gorm.ErrRecordNotFound),
		newRepoResult(&oauthdomain.OAuthBinding{UserID: 42, Provider: "github", Subject: "dup"}, nil),
	}}
	svc := newService(t, repo, dupSubjectProviders(), client, db, &fakeSessionStore{})

	pair, err := svc.Login(context.Background(), "github", "code", "state-1")
	require.NoError(t, err)
	require.NotEmpty(t, pair.AccessToken)

	refreshClaims, err := svc.jwtUtil.Parse(pair.RefreshToken, auth.TokenTypeRefresh)
	require.NoError(t, err)
	assert.Equal(t, uint64(42), refreshClaims.UserID, "创建冲突后应回退重查绑定")
}

func TestLoginCreateConflictFallbackFails(t *testing.T) {
	mrs, client := newRedisClient(t)
	require.NoError(t, mrs.Set(oauthStateKey("state-1"), "github"))
	db := newTestDB(t)
	require.NoError(t, db.Create(&userdomain.User{Username: "github_dup", Password: "x", Status: 1}).Error)

	repo := &fakeBindingRepo{results: []struct {
		binding *oauthdomain.OAuthBinding
		err     error
	}{
		newRepoResult(nil, gorm.ErrRecordNotFound),
		newRepoResult(nil, stderrors.New("still gone")),
	}}
	svc := newService(t, repo, dupSubjectProviders(), client, db, &fakeSessionStore{})

	_, err := svc.Login(context.Background(), "github", "code", "state-1")
	require.Error(t, err)
	assert.True(t, apperrors.IsCode(err, apperrors.CodeInternalError))
}

func TestLoginRepoErrorDoesNotCreateUser(t *testing.T) {
	mrs, client := newRedisClient(t)
	require.NoError(t, mrs.Set(oauthStateKey("state-1"), "github"))
	repo := &fakeBindingRepo{results: []struct {
		binding *oauthdomain.OAuthBinding
		err     error
	}{newRepoResult(nil, stderrors.New("db down"))}}
	svc := newService(t, repo, githubProviders(), client, nil, &fakeSessionStore{})

	_, err := svc.Login(context.Background(), "github", "code", "state-1")
	require.Error(t, err)
	assert.True(t, apperrors.IsCode(err, apperrors.CodeInternalError))
}

func TestLoginSessionCreateFailure(t *testing.T) {
	mrs, client := newRedisClient(t)
	require.NoError(t, mrs.Set(oauthStateKey("state-1"), "github"))
	repo := &fakeBindingRepo{results: []struct {
		binding *oauthdomain.OAuthBinding
		err     error
	}{
		newRepoResult(&oauthdomain.OAuthBinding{UserID: 42, Provider: "github", Subject: "sub123"}, nil),
	}}
	svc := newService(t, repo, githubProviders(), client, nil, &fakeSessionStore{err: stderrors.New("redis full")})

	_, err := svc.Login(context.Background(), "github", "code", "state-1")
	require.Error(t, err)
	assert.True(t, apperrors.IsCode(err, apperrors.CodeInternalError))
}

func TestCreateUserWithBindingTruncatesLongUsername(t *testing.T) {
	db := newTestDB(t)
	svc := newService(t, &fakeBindingRepo{}, nil, nil, db, &fakeSessionStore{})

	longSubject := strings.Repeat("s", 100)
	userID, err := svc.createUserWithBinding(context.Background(), "github", &oauthplatform.UserInfo{Subject: longSubject})
	require.NoError(t, err)
	require.NotZero(t, userID)

	var user userdomain.User
	require.NoError(t, db.First(&user, userID).Error)
	assert.Len(t, user.Username, 64)
	assert.Equal(t, "github_"+strings.Repeat("s", 57), user.Username)

	// 随机密码应被 bcrypt 哈希存储
	assert.True(t, strings.HasPrefix(user.Password, "$2a$"), "密码应为 bcrypt 哈希")
}

func TestCreateUserWithBindingTransactionFailure(t *testing.T) {
	db := newTestDB(t)
	require.NoError(t, db.Create(&userdomain.User{Username: "github_dup", Password: "x", Status: 1}).Error)
	svc := newService(t, &fakeBindingRepo{}, nil, nil, db, &fakeSessionStore{})

	_, err := svc.createUserWithBinding(context.Background(), "github", &oauthplatform.UserInfo{Subject: "dup"})
	require.Error(t, err)
}

func TestRefreshTTLNilExpiry(t *testing.T) {
	assert.Equal(t, time.Duration(0), refreshTTL(auth.Claims{}))
}

func TestRefreshTTLPositive(t *testing.T) {
	exp := time.Now().Add(time.Hour)
	d := refreshTTL(auth.Claims{RegisteredClaims: jwt.RegisteredClaims{ExpiresAt: jwt.NewNumericDate(exp)}})
	assert.Greater(t, d, time.Duration(0))
	assert.Less(t, d, time.Hour)
}

func TestLoginReturnsCompleteTokenPair(t *testing.T) {
	mrs, client := newRedisClient(t)
	require.NoError(t, mrs.Set(oauthStateKey("state-1"), "github"))
	repo := &fakeBindingRepo{results: []struct {
		binding *oauthdomain.OAuthBinding
		err     error
	}{
		newRepoResult(&oauthdomain.OAuthBinding{UserID: 7, Provider: "github", Subject: "sub123"}, nil),
	}}
	svc := newService(t, repo, githubProviders(), client, nil, &fakeSessionStore{})

	pair, err := svc.Login(context.Background(), "github", "code", "state-1")
	require.NoError(t, err)
	assert.Equal(t, &authdomain.TokenPair{
		AccessToken:  pair.AccessToken,
		RefreshToken: pair.RefreshToken,
		ExpiresIn:    30 * 60,
	}, pair)
	require.NotEmpty(t, pair.AccessToken)
	require.NotEmpty(t, pair.RefreshToken)
	assert.Equal(t, 30*60, pair.ExpiresIn)
}
