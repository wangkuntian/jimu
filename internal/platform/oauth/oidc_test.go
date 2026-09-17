package oauth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeIDP 模拟一个 OIDC 提供商的 discovery/token/userinfo 三个端点
type fakeIDP struct {
	*httptest.Server
	issuer        string
	discoveryHits int
	lastForm      url.Values
	lastAuth      string
	userinfo      map[string]string
	withUserinfo  bool
}

func newFakeIDP(t *testing.T, userinfo map[string]string, withUserinfo bool) *fakeIDP {
	t.Helper()
	idp := &fakeIDP{userinfo: userinfo, withUserinfo: withUserinfo}
	mux := http.NewServeMux()
	mux.HandleFunc("/.well-known/openid-configuration", func(w http.ResponseWriter, _ *http.Request) {
		idp.discoveryHits++
		doc := map[string]string{
			"issuer":                 idp.issuer,
			"authorization_endpoint": idp.URL + "/authorize",
			"token_endpoint":         idp.URL + "/token",
		}
		if idp.withUserinfo {
			doc["userinfo_endpoint"] = idp.URL + "/userinfo"
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(doc)
	})
	mux.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		idp.lastForm = r.PostForm
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"access_token": "access-1", "token_type": "Bearer"})
	})
	mux.HandleFunc("/userinfo", func(w http.ResponseWriter, r *http.Request) {
		idp.lastAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(idp.userinfo)
	})
	idp.Server = httptest.NewServer(mux)
	idp.issuer = idp.URL
	t.Cleanup(idp.Close)
	return idp
}

func newOIDCProvider(t *testing.T, idp *fakeIDP, mutate func(*OIDCConfig)) *OIDCProvider {
	t.Helper()
	cfg := OIDCConfig{
		ClientID:     "jimu-client",
		ClientSecret: "jimu-secret",
		RedirectURL:  "http://localhost:8080/api/v1/oauth/keycloak/callback",
		IssuerURL:    idp.issuer,
	}
	if mutate != nil {
		mutate(&cfg)
	}
	return NewOIDCProvider("keycloak", cfg, mockClient())
}

func TestOIDCAuthURLUsesDiscoveryAndCaches(t *testing.T) {
	idp := newFakeIDP(t, nil, true)
	p := newOIDCProvider(t, idp, nil)

	authURL, err := p.AuthURL(context.Background(), "state-1")
	require.NoError(t, err)
	u, err := url.Parse(authURL)
	require.NoError(t, err)
	assert.Equal(t, idp.URL+"/authorize", u.Scheme+"://"+u.Host+u.Path)

	q := u.Query()
	assert.Equal(t, "code", q.Get("response_type"))
	assert.Equal(t, "jimu-client", q.Get("client_id"))
	assert.Equal(t, "http://localhost:8080/api/v1/oauth/keycloak/callback", q.Get("redirect_uri"))
	assert.Equal(t, "state-1", q.Get("state"))
	assert.Equal(t, "openid profile email", q.Get("scope"))

	// 第二次调用命中缓存，不再重复 discovery
	_, err = p.AuthURL(context.Background(), "state-2")
	require.NoError(t, err)
	assert.Equal(t, 1, idp.discoveryHits)
}

func TestOIDCAuthURLHonoursConfiguredScopes(t *testing.T) {
	idp := newFakeIDP(t, nil, true)
	p := newOIDCProvider(t, idp, func(cfg *OIDCConfig) { cfg.Scopes = []string{"openid", "groups"} })

	authURL, err := p.AuthURL(context.Background(), "s")
	require.NoError(t, err)
	assert.Contains(t, authURL, "scope=openid+groups")
}

func TestOIDCExchange(t *testing.T) {
	idp := newFakeIDP(t, map[string]string{"sub": "user-1", "email": "alice@example.com", "name": "Alice"}, true)
	p := newOIDCProvider(t, idp, nil)

	info, err := p.Exchange(context.Background(), "code-1")
	require.NoError(t, err)
	assert.Equal(t, "user-1", info.Subject)
	assert.Equal(t, "alice@example.com", info.Email)
	assert.Equal(t, "Alice", info.Name)

	// token 端点按 client_secret_post 收到授权码与客户端凭证
	assert.Equal(t, "authorization_code", idp.lastForm.Get("grant_type"))
	assert.Equal(t, "code-1", idp.lastForm.Get("code"))
	assert.Equal(t, "jimu-client", idp.lastForm.Get("client_id"))
	assert.Equal(t, "jimu-secret", idp.lastForm.Get("client_secret"))
	assert.Equal(t, "Bearer access-1", idp.lastAuth)
}

func TestOIDCExchangeFallsBackToPreferredUsername(t *testing.T) {
	idp := newFakeIDP(t, map[string]string{"sub": "user-2", "preferred_username": "bob"}, true)
	p := newOIDCProvider(t, idp, nil)

	info, err := p.Exchange(context.Background(), "code-2")
	require.NoError(t, err)
	assert.Equal(t, "bob", info.Name, "缺少 name 时用 preferred_username 兜底")
	assert.Empty(t, info.Email)
}

func TestOIDCDiscoveryUpstreamError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	p := NewOIDCProvider("keycloak", OIDCConfig{
		ClientID: "id", ClientSecret: "s", RedirectURL: "http://localhost/cb", IssuerURL: srv.URL,
	}, mockClient())
	_, err := p.AuthURL(context.Background(), "state")
	assert.ErrorContains(t, err, "discovery")
}

func TestOIDCDiscoveryMissingEndpoints(t *testing.T) {
	var issuer string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"issuer": issuer})
	}))
	defer srv.Close()
	issuer = srv.URL

	p := NewOIDCProvider("keycloak", OIDCConfig{
		ClientID: "id", ClientSecret: "s", RedirectURL: "http://localhost/cb", IssuerURL: srv.URL,
	}, mockClient())
	_, err := p.AuthURL(context.Background(), "state")
	assert.ErrorContains(t, err, "authorization/token")
}

func TestOIDCDiscoveryIssuerMismatch(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"issuer":"https://other.example.com","authorization_endpoint":"https://other/a","token_endpoint":"https://other/t"}`))
	}))
	defer srv.Close()

	p := NewOIDCProvider("keycloak", OIDCConfig{
		ClientID: "id", ClientSecret: "s", RedirectURL: "http://localhost/cb", IssuerURL: srv.URL,
	}, mockClient())
	_, err := p.AuthURL(context.Background(), "state")
	assert.ErrorContains(t, err, "does not match")
}

func TestOIDCExchangeRequiresUserinfoEndpoint(t *testing.T) {
	idp := newFakeIDP(t, nil, false)
	p := newOIDCProvider(t, idp, nil)

	_, err := p.Exchange(context.Background(), "code")
	assert.ErrorContains(t, err, "userinfo_endpoint")
}

func TestOIDCExchangeRejectsMissingSubject(t *testing.T) {
	idp := newFakeIDP(t, map[string]string{"email": "a@x.com"}, true)
	p := newOIDCProvider(t, idp, nil)

	_, err := p.Exchange(context.Background(), "code")
	assert.ErrorContains(t, err, "sub")
}

func TestOIDCDiscoveryRetriesAfterFailure(t *testing.T) {
	var failures int
	mux := http.NewServeMux()
	var issuer string
	mux.HandleFunc("/.well-known/openid-configuration", func(w http.ResponseWriter, _ *http.Request) {
		// 出站 client 默认重试 2 次，这里让首次调用（含重试）全部失败
		if failures < 3 {
			failures++
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{
			"issuer": issuer, "authorization_endpoint": issuer + "/authorize", "token_endpoint": issuer + "/token",
		})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()
	issuer = srv.URL

	p := NewOIDCProvider("keycloak", OIDCConfig{
		ClientID: "id", ClientSecret: "s", RedirectURL: "http://localhost/cb", IssuerURL: srv.URL,
	}, mockClient())

	_, err := p.AuthURL(context.Background(), "state")
	assert.Error(t, err, "首次 discovery 失败（含出站重试）")
	_, err = p.AuthURL(context.Background(), "state")
	assert.NoError(t, err, "失败不缓存，下次调用重试")
}
