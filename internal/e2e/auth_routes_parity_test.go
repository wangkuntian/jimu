package e2e

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestAuthRoutesParity 钉住 P1.6 拆分后的对外 URL 集合：auth/mfa/passkey/captcha
// 四个能力拆开后，认证域路由必须与拆分前逐条一致（不缺失、不新增）。
func TestAuthRoutesParity(t *testing.T) {
	requireCapabilities(t, "auth", "mfa", "passkey", "captcha")

	app := newTestAppWithDB(t)

	registered := make(map[string]bool)
	for _, r := range app.router.Routes() {
		registered[r.Method+" "+r.Path] = true
	}

	want := []string{
		// auth：公开 + 受保护
		http.MethodPost + " /api/v1/auth/login",
		http.MethodPost + " /api/v1/auth/register",
		http.MethodPost + " /api/v1/auth/refresh",
		http.MethodPost + " /api/v1/auth/forgot-password",
		http.MethodPost + " /api/v1/auth/reset-password",
		http.MethodPost + " /api/v1/auth/logout",
		http.MethodPost + " /api/v1/auth/logout-all",
		http.MethodGet + " /api/v1/auth/login-history",
		// mfa：TOTP + 可信设备
		http.MethodPost + " /api/v1/auth/mfa/setup",
		http.MethodPost + " /api/v1/auth/mfa/enable",
		http.MethodPost + " /api/v1/auth/mfa/disable",
		http.MethodGet + " /api/v1/auth/devices",
		http.MethodDelete + " /api/v1/auth/devices",
		http.MethodDelete + " /api/v1/auth/devices/:id",
		// passkey：WebAuthn
		http.MethodPost + " /api/v1/auth/webauthn/login/begin",
		http.MethodPost + " /api/v1/auth/webauthn/login/finish",
		http.MethodPost + " /api/v1/auth/webauthn/register/begin",
		http.MethodPost + " /api/v1/auth/webauthn/register/finish",
		http.MethodGet + " /api/v1/auth/webauthn/credentials",
		http.MethodPut + " /api/v1/auth/webauthn/credentials/:id",
		http.MethodDelete + " /api/v1/auth/webauthn/credentials/:id",
		// captcha：公开
		http.MethodGet + " /api/v1/captcha",
	}
	for _, route := range want {
		require.True(t, registered[route], "缺失路由：%s", route)
	}

	// 认证域不应出现拆分前的重复/冗余路由（逐条计数）
	count := map[string]int{}
	for _, r := range app.router.Routes() {
		if len(r.Path) >= len("/api/v1/auth") && r.Path[:len("/api/v1/auth")] == "/api/v1/auth" {
			count[r.Method+" "+r.Path]++
		}
	}
	for route, n := range count {
		require.Equal(t, 1, n, "路由重复注册：%s", route)
	}
}
