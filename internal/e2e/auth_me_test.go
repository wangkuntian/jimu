package e2e

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type meResponse struct {
	User struct {
		ID       uint64 `json:"id"`
		Username string `json:"username"`
		Status   int8   `json:"status"`
		TenantID uint64 `json:"tenant_id"`
	} `json:"user"`
	Roles       []string `json:"roles"`
	Permissions []struct {
		Resource string `json:"resource"`
		Action   string `json:"action"`
	} `json:"permissions"`
}

func getMe(t *testing.T, r *gin.Engine, token string) meResponse {
	t.Helper()
	w := doJSON(t, r, http.MethodGet, "/api/v1/auth/me", token, "")
	require.Equal(t, http.StatusOK, w.Code)
	resp := parseResp(t, w)
	require.Equal(t, 0, resp.Code)
	var me meResponse
	require.NoError(t, json.Unmarshal(resp.Data, &me))
	return me
}

// TestAuthMeReportsIdentityAndPermissions 管理员应拿到身份 + 角色 + 非空权限点，
// 且权限点里必须含租户列表读权限（v0.4.0 前端「平台工作区」入口的判定依据）。
func TestAuthMeReportsIdentityAndPermissions(t *testing.T) {
	requireCapabilities(t, "auth", "user", "access", "tenant")
	r := newTestApp(t)

	me := getMe(t, r, login(t, r, "admin", "admin123"))

	require.Equal(t, "admin", me.User.Username)
	require.NotZero(t, me.User.ID)
	require.EqualValues(t, 1, me.User.TenantID, "admin 归默认租户")
	require.Contains(t, me.Roles, "平台超级管理员")

	points := map[string]bool{}
	for _, p := range me.Permissions {
		points[p.Resource+" "+p.Action] = true
	}
	require.True(t, points["/api/v1/tenants GET"], "权限点必须含 /api/v1/tenants GET，实际 %v", points)
}

// TestAuthMeSucceedsForRolelessUser 公开注册出来的用户零角色：
// /auth/me 必须 200 + 空数组（前端据此渲染工作区壳），而管理端点是 403。
// 若 /auth/me 误挂在权限中间件下，这里会拿到 403，前端会把「无权限」误判成会话失效。
func TestAuthMeSucceedsForRolelessUser(t *testing.T) {
	requireCapabilities(t, "auth", "user", "access", "tenant")
	r := newTestApp(t)

	w := doJSON(t, r, http.MethodPost, "/api/v1/auth/register", "", `{"username":"roleless","password":"roleless123"}`)
	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, 0, parseResp(t, w).Code)

	token := login(t, r, "roleless", "roleless123")

	me := getMe(t, r, token)
	require.Empty(t, me.Roles)
	require.Empty(t, me.Permissions)
	require.EqualValues(t, 1, me.User.TenantID, "公开注册归默认租户")

	// 对照：同一 token 调管理端点必须是 403（不是 401、也不是 200）
	w = doJSON(t, r, http.MethodGet, "/api/v1/tenants", token, "")
	require.Equal(t, http.StatusForbidden, w.Code)
}

// TestAuthMeRequiresToken 无 token 访问 /auth/me → 401（不能因为它是自服务端点就放开）。
func TestAuthMeRequiresToken(t *testing.T) {
	requireCapabilities(t, "auth")
	r := newTestApp(t)

	w := doJSON(t, r, http.MethodGet, "/api/v1/auth/me", "", "")
	require.Equal(t, http.StatusUnauthorized, w.Code)
}
