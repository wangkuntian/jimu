package interfaces

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"jimu/internal/capabilities/passkey/application"
	passkeydomain "jimu/internal/capabilities/passkey/domain"
	"jimu/internal/config"
	"jimu/internal/contract"
	"jimu/internal/kernel/auth"
	apperrors "jimu/internal/shared/errors"

	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// handlerUserRepo 固定用户只读端口
type handlerUserRepo struct {
	user *contract.Userinfo
}

func (r *handlerUserRepo) GetByID(context.Context, uint64) (*contract.Userinfo, error) {
	if r.user == nil {
		return nil, contract.ErrNotFound
	}
	return r.user, nil
}

func (r *handlerUserRepo) FindByUsername(context.Context, string) (*contract.Userinfo, error) {
	if r.user == nil {
		return nil, contract.ErrNotFound
	}
	return r.user, nil
}

func (r *handlerUserRepo) List(context.Context, int, int) ([]contract.Userinfo, int64, error) {
	return nil, 0, nil
}

// handlerFinalizer 记录登录收尾调用（passkey 手势测试用）
type handlerFinalizer struct{}

func (handlerFinalizer) FinalizeLogin(context.Context, uint64) (*contract.TokenPair, error) {
	return &contract.TokenPair{AccessToken: "access", RefreshToken: "refresh", ExpiresIn: 1800}, nil
}
func (handlerFinalizer) CheckLocked(context.Context, string) (bool, time.Duration, error) {
	return false, 0, nil
}
func (handlerFinalizer) RecordFailure(context.Context, string) {}
func (handlerFinalizer) ResetFailure(context.Context, string)  {}
func (handlerFinalizer) RecordLoginHistory(context.Context, uint64, uint64, string, string, string) {
}

// webauthnHandlerCredRepo 内存凭证仓储
type webauthnHandlerCredRepo struct {
	items  []*passkeydomain.WebAuthnCredential
	nextID uint64
}

func newWebAuthnHandlerCredRepo() *webauthnHandlerCredRepo {
	return &webauthnHandlerCredRepo{nextID: 1}
}

func (r *webauthnHandlerCredRepo) Create(_ context.Context, credential *passkeydomain.WebAuthnCredential) error {
	credential.ID = r.nextID
	r.nextID++
	r.items = append(r.items, credential)
	return nil
}

func (r *webauthnHandlerCredRepo) FindByCredentialID(_ context.Context, credentialID string) (*passkeydomain.WebAuthnCredential, error) {
	for _, item := range r.items {
		if item.CredentialID == credentialID {
			return item, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

func (r *webauthnHandlerCredRepo) ListByUser(_ context.Context, tenantID, userID uint64) ([]passkeydomain.WebAuthnCredential, error) {
	var out []passkeydomain.WebAuthnCredential
	for _, item := range r.items {
		if item.UserID == userID && (tenantID == 0 || item.TenantID == tenantID) {
			out = append(out, *item)
		}
	}
	return out, nil
}

func (r *webauthnHandlerCredRepo) CountByUser(_ context.Context, userID uint64) (int64, error) {
	var n int64
	for _, item := range r.items {
		if item.UserID == userID {
			n++
		}
	}
	return n, nil
}

func (r *webauthnHandlerCredRepo) Touch(_ context.Context, id uint64, signCount uint32, backupState bool, usedAt time.Time) error {
	for _, item := range r.items {
		if item.ID == id {
			item.SignCount = signCount
			item.BackupState = backupState
			item.LastUsedAt = &usedAt
		}
	}
	return nil
}

func (r *webauthnHandlerCredRepo) UpdateName(_ context.Context, tenantID, userID, id uint64, name string) error {
	for _, item := range r.items {
		if item.ID == id && item.UserID == userID && (tenantID == 0 || item.TenantID == tenantID) {
			item.Name = name
		}
	}
	return nil
}

func (r *webauthnHandlerCredRepo) Delete(_ context.Context, tenantID, userID, id uint64) error {
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

// newWebAuthnHandler 构造启用 WebAuthn 的处理器与内存仓储
func newWebAuthnHandler(t *testing.T) (*PasskeyHandler, *handlerUserRepo, *webauthnHandlerCredRepo) {
	t.Helper()
	mrs, err := miniredis.Run()
	require.NoError(t, err)
	t.Cleanup(mrs.Close)
	rdb := redis.NewClient(&redis.Options{Addr: mrs.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })

	handle, err := webauthn.New(&webauthn.Config{
		RPDisplayName: "Jimu Test",
		RPID:          "localhost",
		RPOrigins:     []string{"http://localhost:8080"},
	})
	require.NoError(t, err)

	user := &contract.Userinfo{ID: 42, Username: "alice", Status: 1, TenantID: 1}
	repo := &handlerUserRepo{user: user}
	creds := newWebAuthnHandlerCredRepo()
	svc := application.NewPasskeyService(application.Deps{
		Users:       repo,
		Credentials: creds,
		WebAuthn:    handle,
		Redis:       rdb,
		SessionTTL:  time.Minute,
		Finalizer:   handlerFinalizer{},
	})
	return NewPasskeyHandler(svc, config.AuthConfig{}, nil), repo, creds
}

// invokeHandler 以直接调用处理器的方式发请求
func invokeHandler(t *testing.T, method, target, body string, userID uint64, fn func(*gin.Context)) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Handle(method, "/x", func(c *gin.Context) {
		if userID != 0 {
			c.Set("user_id", userID)
		}
		fn(c)
	})
	req := httptest.NewRequest(method, target, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func decodeBody(t *testing.T, w *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	return body
}

func TestWebAuthnHandlersRequireUserContext(t *testing.T) {
	handler, _, _ := newWebAuthnHandler(t)

	cases := []struct {
		name string
		fn   func(*gin.Context)
	}{
		{"begin register", handler.BeginWebAuthnRegistration},
		{"finish register", handler.FinishWebAuthnRegistration},
		{"list credentials", handler.ListWebAuthnCredentials},
		{"rename credential", handler.RenameWebAuthnCredential},
		{"delete credential", handler.DeleteWebAuthnCredential},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := invokeHandler(t, http.MethodPost, "/x", "", 0, tc.fn)
			assert.Equal(t, http.StatusUnauthorized, w.Code)
			assert.Equal(t, float64(apperrors.CodeUnauthorized), decodeBody(t, w)["code"])
		})
	}
}

func TestWebAuthnRegisterBeginAndFinish(t *testing.T) {
	handler, _, creds := newWebAuthnHandler(t)

	// begin：返回 session_id 与注册选项
	w := invokeHandler(t, http.MethodPost, "/x", `{"name":"MacBook"}`, 42, handler.BeginWebAuthnRegistration)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	data := decodeBody(t, w)["data"].(map[string]any)
	sessionID, ok := data["session_id"].(string)
	require.True(t, ok, "缺少 session_id: %s", w.Body.String())
	options, ok := data["options"].(map[string]any)
	require.True(t, ok)
	require.Contains(t, options, "publicKey")

	// finish：请求体不是合法凭证 → 通行密钥校验失败
	w = invokeHandler(t, http.MethodPost, "/x?session_id="+sessionID, `{"not":"a credential"}`, 42, handler.FinishWebAuthnRegistration)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, float64(apperrors.CodeWebAuthnVerificationFailed), decodeBody(t, w)["code"])
	assert.Empty(t, creds.items)

	// session 已消费：再次提交同样失败
	w = invokeHandler(t, http.MethodPost, "/x?session_id="+sessionID, `{}`, 42, handler.FinishWebAuthnRegistration)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestWebAuthnBeginRegisterRejectsMalformedBody(t *testing.T) {
	handler, _, _ := newWebAuthnHandler(t)
	w := invokeHandler(t, http.MethodPost, "/x", `{`, 42, handler.BeginWebAuthnRegistration)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, float64(apperrors.CodeInvalidParam), decodeBody(t, w)["code"])
}

func TestWebAuthnLoginBeginWithoutCredential(t *testing.T) {
	handler, _, _ := newWebAuthnHandler(t)
	gin.SetMode(gin.TestMode)

	w := invokeHandler(t, http.MethodPost, "/x", "", 0, func(c *gin.Context) {
		c.Set("validated_req", &webAuthnLoginBeginRequest{Username: "alice"})
		handler.BeginWebAuthnLogin(c)
	})
	// 用户存在但没有凭证 → 2010
	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Equal(t, float64(apperrors.CodeWebAuthnNoCredential), decodeBody(t, w)["code"])
}

func TestWebAuthnLoginBeginRateLimited(t *testing.T) {
	handler, _, _ := newWebAuthnHandler(t)
	handler.limiter = auth.NewLimiter(&routerLimiterRedis{counts: map[string]int{}}, true)
	handler.cfg = config.AuthConfig{LoginRateLimit: 1, LoginRateWindowSec: 60}
	gin.SetMode(gin.TestMode)

	call := func() *httptest.ResponseRecorder {
		return invokeHandler(t, http.MethodPost, "/x", "", 0, func(c *gin.Context) {
			c.Set("validated_req", &webAuthnLoginBeginRequest{Username: "alice"})
			handler.BeginWebAuthnLogin(c)
		})
	}
	assert.NotEqual(t, http.StatusTooManyRequests, call().Code, "首次请求应放行")
	assert.Equal(t, http.StatusTooManyRequests, call().Code, "同 IP 超限应返回 429")
}

func TestWebAuthnLoginFinishInvalidSession(t *testing.T) {
	handler, _, _ := newWebAuthnHandler(t)
	w := invokeHandler(t, http.MethodPost, "/x?session_id=not-exist", `{}`, 0, handler.FinishWebAuthnLogin)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, float64(apperrors.CodeWebAuthnVerificationFailed), decodeBody(t, w)["code"])
}

func TestWebAuthnCredentialManagementHandlers(t *testing.T) {
	handler, _, creds := newWebAuthnHandler(t)

	// 预置两条凭证（分属两个租户，用于验证按上下文租户隔离）
	require.NoError(t, creds.Create(context.Background(), &passkeydomain.WebAuthnCredential{
		TenantID: 1, UserID: 42, CredentialID: "cred-1", PublicKey: []byte{1}, Name: "Phone",
	}))
	require.NoError(t, creds.Create(context.Background(), &passkeydomain.WebAuthnCredential{
		TenantID: 2, UserID: 42, CredentialID: "cred-2", PublicKey: []byte{2}, Name: "Other tenant",
	}))

	// 列表（无租户上下文 → 平台视角，返回全部）
	w := invokeHandler(t, http.MethodGet, "/x", "", 42, handler.ListWebAuthnCredentials)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	list := decodeBody(t, w)["data"].([]any)
	assert.Len(t, list, 2)

	// 重命名非法 ID
	w = invokeHandler(t, http.MethodPut, "/x", `{"name":"x"}`, 42, func(c *gin.Context) {
		c.Params = gin.Params{{Key: "id", Value: "abc"}}
		handler.RenameWebAuthnCredential(c)
	})
	assert.Equal(t, http.StatusBadRequest, w.Code)

	// 重命名空名称
	w = invokeHandler(t, http.MethodPut, "/x", `{"name":"  "}`, 42, func(c *gin.Context) {
		c.Params = gin.Params{{Key: "id", Value: "1"}}
		handler.RenameWebAuthnCredential(c)
	})
	assert.Equal(t, http.StatusBadRequest, w.Code)

	// 正常重命名
	w = invokeHandler(t, http.MethodPut, "/x", `{"name":"会议室电脑"}`, 42, func(c *gin.Context) {
		c.Params = gin.Params{{Key: "id", Value: "1"}}
		handler.RenameWebAuthnCredential(c)
	})
	assert.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Equal(t, "会议室电脑", creds.items[0].Name)

	// 非法 ID 删除
	w = invokeHandler(t, http.MethodDelete, "/x", "", 42, func(c *gin.Context) {
		c.Params = gin.Params{{Key: "id", Value: "0"}}
		handler.DeleteWebAuthnCredential(c)
	})
	assert.Equal(t, http.StatusBadRequest, w.Code)

	// 正常删除
	w = invokeHandler(t, http.MethodDelete, "/x", "", 42, func(c *gin.Context) {
		c.Params = gin.Params{{Key: "id", Value: "1"}}
		handler.DeleteWebAuthnCredential(c)
	})
	assert.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Len(t, creds.items, 1)
}

func TestWebAuthnHandlersWithoutConfiguration(t *testing.T) {
	svc := application.NewPasskeyService(application.Deps{Users: &handlerUserRepo{}})
	handler := NewPasskeyHandler(svc, config.AuthConfig{}, nil)

	cases := []struct {
		name string
		fn   func(*gin.Context)
	}{
		{"begin register", handler.BeginWebAuthnRegistration},
		{"list credentials", handler.ListWebAuthnCredentials},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := invokeHandler(t, http.MethodPost, "/x", "", 42, tc.fn)
			assert.Equal(t, http.StatusInternalServerError, w.Code)
			assert.Equal(t, float64(apperrors.CodeInternalError), decodeBody(t, w)["code"])
		})
	}
}

// ---- 限流 Redis 替身（redis.Scripter） ----

type routerLimiterRedis struct {
	counts map[string]int
}

func (r *routerLimiterRedis) Eval(ctx context.Context, script string, keys []string, args ...interface{}) *redis.Cmd {
	return r.eval(keys, args...)
}

func (r *routerLimiterRedis) EvalSha(ctx context.Context, sha1 string, keys []string, args ...interface{}) *redis.Cmd {
	return r.eval(keys, args...)
}

func (r *routerLimiterRedis) EvalRO(ctx context.Context, script string, keys []string, args ...interface{}) *redis.Cmd {
	return r.eval(keys, args...)
}

func (r *routerLimiterRedis) EvalShaRO(ctx context.Context, sha1 string, keys []string, args ...interface{}) *redis.Cmd {
	return r.eval(keys, args...)
}

func (r *routerLimiterRedis) ScriptExists(ctx context.Context, hashes ...string) *redis.BoolSliceCmd {
	exists := make([]bool, len(hashes))
	for i := range exists {
		exists[i] = true
	}
	return redis.NewBoolSliceResult(exists, nil)
}

func (r *routerLimiterRedis) ScriptLoad(ctx context.Context, script string) *redis.StringCmd {
	return redis.NewStringResult("fake-script", nil)
}

func (r *routerLimiterRedis) eval(keys []string, args ...interface{}) *redis.Cmd {
	if len(keys) != 1 || len(args) != 2 {
		return redis.NewCmdResult(nil, fmt.Errorf("unexpected limiter call"))
	}
	limit, err := routerLimiterInt(args[1])
	if err != nil {
		return redis.NewCmdResult(nil, err)
	}
	r.counts[keys[0]]++
	if r.counts[keys[0]] > limit {
		return redis.NewCmdResult(int64(0), nil)
	}
	return redis.NewCmdResult(int64(1), nil)
}

func routerLimiterInt(v interface{}) (int, error) {
	switch n := v.(type) {
	case int:
		return n, nil
	case int64:
		return int(n), nil
	default:
		return 0, fmt.Errorf("unexpected arg type %T", v)
	}
}
