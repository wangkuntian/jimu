// internal/modules/oauth/interfaces/handler_test.go
package interfaces

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"jimu/internal/modules/oauth/application"
	oauthdomain "jimu/internal/modules/oauth/domain"
	"jimu/internal/platform/auth"
	oauthplatform "jimu/internal/platform/oauth"
	apperrors "jimu/internal/shared/errors"

	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

type fakeProvider struct {
	info *oauthplatform.UserInfo
	err  error
}

func (p *fakeProvider) Name() string { return "github" }
func (p *fakeProvider) AuthURL(state string) string {
	return "https://github.com/login/oauth/authorize?state=" + state
}
func (p *fakeProvider) Exchange(context.Context, string) (*oauthplatform.UserInfo, error) {
	return p.info, p.err
}

type fakeBindingRepo struct {
	binding *oauthdomain.OAuthBinding
	err     error
}

func (r *fakeBindingRepo) FindByProviderSubject(context.Context, string, string) (*oauthdomain.OAuthBinding, error) {
	return r.binding, r.err
}
func (r *fakeBindingRepo) Create(context.Context, *oauthdomain.OAuthBinding) error { return nil }

type fakeSessionStore struct {
	err error
}

func (s *fakeSessionStore) Create(context.Context, uint64, string, string, time.Duration) error {
	return s.err
}
func (s *fakeSessionStore) Rotate(context.Context, uint64, string, string, string, time.Duration) error {
	return nil
}
func (s *fakeSessionStore) Revoke(context.Context, uint64, string) error { return nil }
func (s *fakeSessionStore) RevokeAll(context.Context, uint64) error      { return nil }

func newTestHandler(t *testing.T, providers map[string]oauthplatform.Provider) *OAuthHandler {
	t.Helper()
	mrs, err := miniredis.Run()
	require.NoError(t, err)
	t.Cleanup(mrs.Close)
	client := redis.NewClient(&redis.Options{Addr: mrs.Addr()})
	t.Cleanup(func() { _ = client.Close() })

	jwtUtil := auth.New("01234567890123456789012345678901", "jimu", 30, 7)
	svc := application.NewOAuthService(
		&fakeBindingRepo{binding: &oauthdomain.OAuthBinding{UserID: 42, Provider: "github", Subject: "sub123"}},
		jwtUtil,
		&fakeSessionStore{},
		providers,
		client,
		nil,
		30,
	)
	return NewOAuthHandler(svc)
}

func githubProvider() oauthplatform.Provider {
	return &fakeProvider{info: &oauthplatform.UserInfo{Subject: "sub123", Email: "a@b.c"}}
}

func newTestRouter(t *testing.T, handler *OAuthHandler) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	RegisterOAuthRoutes(r.Group("/api/v1"), handler.service)
	return r
}

func doRequest(t *testing.T, r *gin.Engine, method, path string) *httptest.ResponseRecorder {
	t.Helper()
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(method, path, nil))
	return w
}

func assertCode(t *testing.T, w *httptest.ResponseRecorder, code int) {
	t.Helper()
	var body struct {
		Code int `json:"code"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, code, body.Code)
}

func TestOAuthRoutesRegistered(t *testing.T) {
	handler := newTestHandler(t, map[string]oauthplatform.Provider{"github": githubProvider()})
	r := newTestRouter(t, handler)

	routes := make(map[string]bool)
	for _, route := range r.Routes() {
		routes[route.Method+" "+route.Path] = true
	}
	assert.True(t, routes["GET /api/v1/oauth/:provider/login"], "缺少 login 路由")
	assert.True(t, routes["GET /api/v1/oauth/:provider/callback"], "缺少 callback 路由")
}

func TestLoginHandlerRedirects(t *testing.T) {
	handler := newTestHandler(t, map[string]oauthplatform.Provider{"github": githubProvider()})
	r := newTestRouter(t, handler)

	w := doRequest(t, r, http.MethodGet, "/api/v1/oauth/github/login")
	assert.Equal(t, http.StatusFound, w.Code)
	loc := w.Header().Get("Location")
	assert.True(t, strings.HasPrefix(loc, "https://github.com/login/oauth/authorize?state="))
	assert.NotEmpty(t, strings.TrimPrefix(loc, "https://github.com/login/oauth/authorize?state="))
}

func TestLoginHandlerUnknownProvider(t *testing.T) {
	handler := newTestHandler(t, map[string]oauthplatform.Provider{"github": githubProvider()})
	r := newTestRouter(t, handler)

	w := doRequest(t, r, http.MethodGet, "/api/v1/oauth/wechat/login")
	assert.Equal(t, http.StatusNotFound, w.Code)
	assertCode(t, w, apperrors.CodeOAuthProviderNotFound)
}

func TestCallbackHandlerSuccess(t *testing.T) {
	handler := newTestHandler(t, map[string]oauthplatform.Provider{"github": githubProvider()})
	// 预置 state（模拟 BeginLogin 阶段写入）
	_, err := handler.service.AuthURL(context.Background(), "github", "state-1")
	require.NoError(t, err)
	r := newTestRouter(t, handler)

	w := doRequest(t, r, http.MethodGet, "/api/v1/oauth/github/callback?code=auth-code&state=state-1")
	assert.Equal(t, http.StatusOK, w.Code)
	assertCode(t, w, apperrors.CodeOK)

	var body struct {
		Data struct {
			AccessToken  string `json:"access_token"`
			RefreshToken string `json:"refresh_token"`
			ExpiresIn    int    `json:"expires_in"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.NotEmpty(t, body.Data.AccessToken)
	assert.NotEmpty(t, body.Data.RefreshToken)
	assert.Equal(t, 30*60, body.Data.ExpiresIn)
}

func TestCallbackHandlerInvalidState(t *testing.T) {
	handler := newTestHandler(t, map[string]oauthplatform.Provider{"github": githubProvider()})
	r := newTestRouter(t, handler)

	w := doRequest(t, r, http.MethodGet, "/api/v1/oauth/github/callback?code=auth-code&state=nope")
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assertCode(t, w, apperrors.CodeInvalidParam)
}

func TestCallbackHandlerExchangeFailure(t *testing.T) {
	handler := newTestHandler(t, map[string]oauthplatform.Provider{
		"github": &fakeProvider{err: gorm.ErrInvalidDB},
	})
	_, err := handler.service.AuthURL(context.Background(), "github", "state-1")
	require.NoError(t, err)
	r := newTestRouter(t, handler)

	w := doRequest(t, r, http.MethodGet, "/api/v1/oauth/github/callback?code=auth-code&state=state-1")
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assertCode(t, w, apperrors.CodeInternalError)
}
