package oauth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"jimu/internal/platform/httpclient"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockTokenServer 返回 access_token 的 token 端点；status=500 时模拟上游失败
func mockTokenServer(t *testing.T, status int, extra map[string]interface{}) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if status != http.StatusOK {
			w.WriteHeader(status)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		resp := map[string]interface{}{"access_token": "test-token", "token_type": "Bearer"}
		for k, v := range extra {
			resp[k] = v
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
}

func mockClient() *httpclient.Client {
	return httpclient.New(httpclient.Config{RetryIntervalMS: 1})
}

func TestGoogleProviderExchange(t *testing.T) {
	userSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"id": "g123", "email": "a@x.com", "name": "Alice"})
	}))
	defer userSrv.Close()
	tokenSrv := mockTokenServer(t, http.StatusOK, nil)
	defer tokenSrv.Close()

	p := NewGoogleProvider(GoogleConfig{ClientID: "id", ClientSecret: "secret", RedirectURL: "http://localhost/cb"}, mockClient())
	p.config.Endpoint.TokenURL = tokenSrv.URL
	p.userInfoURL = userSrv.URL

	assert.Equal(t, "google", p.Name())

	info, err := p.Exchange(context.Background(), "code")
	require.NoError(t, err)
	assert.Equal(t, "g123", info.Subject)
	assert.Equal(t, "a@x.com", info.Email)
	assert.Equal(t, "Alice", info.Name)
}

func TestGoogleProviderExchangeTokenError(t *testing.T) {
	tokenSrv := mockTokenServer(t, http.StatusInternalServerError, nil)
	defer tokenSrv.Close()

	p := NewGoogleProvider(GoogleConfig{ClientID: "id", ClientSecret: "secret", RedirectURL: "http://localhost/cb"}, mockClient())
	p.config.Endpoint.TokenURL = tokenSrv.URL

	_, err := p.Exchange(context.Background(), "code")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "google exchange")
}

func TestGoogleProviderExchangeBadUserInfo(t *testing.T) {
	userSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{invalid json`))
	}))
	defer userSrv.Close()
	tokenSrv := mockTokenServer(t, http.StatusOK, nil)
	defer tokenSrv.Close()

	p := NewGoogleProvider(GoogleConfig{ClientID: "id", ClientSecret: "secret", RedirectURL: "http://localhost/cb"}, mockClient())
	p.config.Endpoint.TokenURL = tokenSrv.URL
	p.userInfoURL = userSrv.URL

	_, err := p.Exchange(context.Background(), "code")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unmarshal google userinfo")
}

func TestGitHubProviderExchange(t *testing.T) {
	userSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"id": 42, "email": "git@x.com", "login": "octo"})
	}))
	defer userSrv.Close()
	tokenSrv := mockTokenServer(t, http.StatusOK, nil)
	defer tokenSrv.Close()

	p := NewGitHubProvider(GitHubConfig{ClientID: "id", ClientSecret: "secret", RedirectURL: "http://localhost/cb"}, mockClient())
	p.config.Endpoint.TokenURL = tokenSrv.URL
	p.userInfoURL = userSrv.URL

	assert.Equal(t, "github", p.Name())

	info, err := p.Exchange(context.Background(), "code")
	require.NoError(t, err)
	assert.Equal(t, "42", info.Subject)
	assert.Equal(t, "git@x.com", info.Email)
	assert.Equal(t, "octo", info.Name)
}

func TestWeChatProviderExchange(t *testing.T) {
	var gotQuery string
	userSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"openid": "wx-1", "nickname": "小王"})
	}))
	defer userSrv.Close()
	tokenSrv := mockTokenServer(t, http.StatusOK, map[string]interface{}{"openid": "wx-openid-token"})
	defer tokenSrv.Close()

	p := NewWeChatProvider(WeChatConfig{ClientID: "id", ClientSecret: "secret", RedirectURL: "http://localhost/cb"}, mockClient())
	p.config.Endpoint.TokenURL = tokenSrv.URL
	p.userInfoURL = userSrv.URL

	assert.Equal(t, "wechat", p.Name())

	info, err := p.Exchange(context.Background(), "code")
	require.NoError(t, err)
	assert.Equal(t, "wx-1", info.Subject)
	assert.Equal(t, "小王", info.Name)
	assert.Contains(t, gotQuery, "openid=wx-openid-token")
	assert.Contains(t, gotQuery, "access_token=test-token")
}

func TestWeChatProviderExchangeMissingOpenID(t *testing.T) {
	tokenSrv := mockTokenServer(t, http.StatusOK, nil) // 无 openid
	defer tokenSrv.Close()

	p := NewWeChatProvider(WeChatConfig{ClientID: "id", ClientSecret: "secret", RedirectURL: "http://localhost/cb"}, mockClient())
	p.config.Endpoint.TokenURL = tokenSrv.URL

	_, err := p.Exchange(context.Background(), "code")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "wechat openid missing")
}
