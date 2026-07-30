package interfaces

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"jimu/internal/config"
	"jimu/internal/platform/auth"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

func TestRegisterRouteDependsOnPublicRegistration(t *testing.T) {
	tests := []struct {
		name               string
		publicRegistration bool
		want               int
	}{
		{"disabled", false, http.StatusNotFound},
		{"enabled", true, http.StatusBadRequest},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := testRouter(false)
			cfg := testAuthConfig()
			cfg.PublicRegistration = tt.publicRegistration
			RegisterAuthRoutes(r.Group("/api/v1"), nil, auth.New(strings.Repeat("s", 32), "jimu", 30, 7), cfg, nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", strings.NewReader("{}")))
			if w.Code != tt.want {
				t.Fatalf("status = %d, want %d", w.Code, tt.want)
			}
		})
	}
}

func TestLogoutRouteRequiresAccessToken(t *testing.T) {
	r := testRouter(false)
	RegisterAuthRoutes(r.Group("/api/v1"), nil, auth.New(strings.Repeat("s", 32), "jimu", 30, 7), testAuthConfig(), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil))
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestRefreshRouteStaysPublic(t *testing.T) {
	r := testRouter(false)
	RegisterAuthRoutes(r.Group("/api/v1"), nil, auth.New(strings.Repeat("s", 32), "jimu", 30, 7), testAuthConfig(), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", strings.NewReader("{}")))
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestLoginRateLimitUsesIPScope(t *testing.T) {
	limiter, closeServer := newRouterLimiter(t)
	defer closeServer()
	cfg := testAuthConfig()
	if ok, err := limiter.Allow(context.Background(), "login", "ip:127.0.0.1", cfg.LoginRateLimit, time.Duration(cfg.LoginRateWindowSec)*time.Second); err != nil || !ok {
		t.Fatalf("preconsume: ok=%v err=%v", ok, err)
	}

	r := testRouter(false)
	RegisterAuthRoutes(r.Group("/api/v1"), nil, auth.New(strings.Repeat("s", 32), "jimu", 30, 7), cfg, limiter)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(`{"username":"alice","password":"secret"}`))
	req.RemoteAddr = "127.0.0.1:1234"
	r.ServeHTTP(w, req)
	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusTooManyRequests)
	}
}

func TestLoginRateLimitUsesNormalizedUsernameScope(t *testing.T) {
	limiter, closeServer := newRouterLimiter(t)
	defer closeServer()
	cfg := testAuthConfig()
	if ok, err := limiter.Allow(context.Background(), "login", "username:alice", cfg.LoginRateLimit, time.Duration(cfg.LoginRateWindowSec)*time.Second); err != nil || !ok {
		t.Fatalf("preconsume: ok=%v err=%v", ok, err)
	}

	r := testRouter(false)
	RegisterAuthRoutes(r.Group("/api/v1"), nil, auth.New(strings.Repeat("s", 32), "jimu", 30, 7), cfg, limiter)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(`{"username":" Alice ","password":"secret"}`))
	req.RemoteAddr = "127.0.0.1:1234"
	r.ServeHTTP(w, req)
	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusTooManyRequests)
	}
}

func TestRegisterRateLimitUsesIPScope(t *testing.T) {
	limiter, closeServer := newRouterLimiter(t)
	defer closeServer()
	cfg := testAuthConfig()
	cfg.PublicRegistration = true
	if ok, err := limiter.Allow(context.Background(), "register", "ip:127.0.0.1", cfg.RegisterRateLimit, time.Duration(cfg.RegisterRateWindowSec)*time.Second); err != nil || !ok {
		t.Fatalf("preconsume: ok=%v err=%v", ok, err)
	}

	r := testRouter(false)
	RegisterAuthRoutes(r.Group("/api/v1"), nil, auth.New(strings.Repeat("s", 32), "jimu", 30, 7), cfg, limiter)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", strings.NewReader(`{"username":"alice","password":"secret"}`))
	req.RemoteAddr = "127.0.0.1:1234"
	r.ServeHTTP(w, req)
	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusTooManyRequests)
	}
}

func testRouter(debug bool) *gin.Engine {
	gin.SetMode(gin.TestMode)
	return gin.New()
}

func testAuthConfig() config.AuthConfig {
	return config.AuthConfig{
		PublicRegistration:    false,
		LoginRateLimit:        1,
		LoginRateWindowSec:    60,
		RegisterRateLimit:     1,
		RegisterRateWindowSec: 60,
	}
}

func newRouterLimiter(t *testing.T) (*auth.Limiter, func()) {
	t.Helper()
	return auth.NewLimiter(&routerLimiterRedis{counts: make(map[string]int)}, true), func() {}
}

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

func routerLimiterInt(value interface{}) (int, error) {
	switch v := value.(type) {
	case int:
		return v, nil
	case int64:
		return int(v), nil
	case string:
		return strconv.Atoi(v)
	default:
		return strconv.Atoi(fmt.Sprint(value))
	}
}
