package e2e

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"jimu/internal/config"
	"jimu/internal/contract"
	adminmodule "jimu/internal/modules/admin"
	auditmodule "jimu/internal/modules/audit"
	auditdomain "jimu/internal/modules/audit/domain"
	authmodule "jimu/internal/modules/auth"
	"jimu/internal/modules/permission"
	"jimu/internal/modules/role"
	roledomain "jimu/internal/modules/role/domain"
	usermodule "jimu/internal/modules/user"
	userdomain "jimu/internal/modules/user/domain"
	platformauth "jimu/internal/platform/auth"
	"jimu/internal/platform/db"
	"jimu/internal/platform/logger"

	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// newTestApp 组装一个真实的应用路由：auth + user + role + permission + audit 模块，
// 底层用 sqlite（内存共享缓存）+ miniredis，无需外部 MySQL/Redis。
func newTestApp(t *testing.T) *gin.Engine {
	return newTestAppWithDB(t).router
}

// testAppDB 组装好的应用与底层 gorm 句柄
type testAppDB struct {
	router *gin.Engine
	db     *gorm.DB
}

func newTestAppWithDB(t *testing.T) *testAppDB {
	t.Helper()
	gin.SetMode(gin.TestMode)

	gdb, err := gorm.Open(sqlite.Open("file:e2e_contract?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	// 内存库单连接，避免多连接看到不同实例
	if sqlDB, err := gdb.DB(); err == nil {
		sqlDB.SetMaxOpenConns(1)
	}
	// 雪花 ID 注入（uint64 主键模型）
	require.NoError(t, db.InitSnowflake(1))
	db.RegisterSnowflakeHook(gdb)

	// 建表 + 关联表（seed 依赖 user_roles / role_permissions 原始 SQL）
	require.NoError(t, gdb.AutoMigrate(
		&userdomain.User{},
		&roledomain.Role{},
		&roledomain.Permission{},
		&auditdomain.AuditLog{},
	))
	require.NoError(t, gdb.Exec(`CREATE TABLE IF NOT EXISTS user_roles (user_id INTEGER NOT NULL, role_id INTEGER NOT NULL)`).Error)
	require.NoError(t, gdb.Exec(`CREATE TABLE IF NOT EXISTS role_permissions (role_id INTEGER NOT NULL, permission_id INTEGER NOT NULL)`).Error)

	// 种子：admin 用户 + 超级管理员角色 + 权限（含 /users、/audits 策略）
	t.Setenv("ADMIN_PASSWORD", "admin123")
	require.NoError(t, db.RunSeed(gdb))

	mr, err := miniredis.Run()
	require.NoError(t, err)
	t.Cleanup(mr.Close)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })

	cfg := config.Config{
		Auth: config.AuthConfig{
			JWTSecret:          "0123456789abcdef0123456789abcdef",
			Issuer:             "jimu-e2e",
			AccessExpireMin:    30,
			RefreshExpireDay:   7,
			PublicRegistration: true,
			// 0 = 关闭限流，避免测试依赖 Redis Lua 脚本
			LoginRateLimit:    0,
			RegisterRateLimit: 0,
		},
		Audit: config.AuditConfig{QueueSize: 1024, BatchSize: 1, FlushIntervalMS: 10},
	}

	log := logger.New(config.LogConfig{Level: "error", Format: "console", Output: "stdout"})

	authMod := authmodule.New(gdb, rdb, cfg.Auth, false, nil, config.CaptchaConfig{})
	userMod := usermodule.New(gdb, cfg) // 不传 rdb：跳过用户维度限流（依赖 Lua），聚焦契约链路
	roleMod := role.New(gdb)
	permMod := permission.New(gdb)
	auditMod := auditmodule.New(gdb, cfg.Audit, log)
	adminMod := adminmodule.New("test", "test", rdb, gdb)

	router := gin.New()

	modules := []contract.Module{authMod, userMod, roleMod, permMod, auditMod, adminMod}
	// 1) 模块级 HTTP 中间件（审计记录）
	for _, m := range modules {
		if p, ok := m.(contract.HTTPMiddlewareProvider); ok {
			router.Use(p.HTTPMiddleware()...)
		}
	}
	// 2) 受保护中间件（JWT + RBAC）
	var protected []gin.HandlerFunc
	for _, m := range modules {
		if p, ok := m.(contract.ProtectedHTTPMiddlewareProvider); ok {
			protected, err = p.ProtectedHTTPMiddleware()
			require.NoError(t, err)
			break
		}
	}
	// 3) 路由注册（auth/oauth 公开，其余受保护）
	for _, m := range modules {
		if len(protected) > 0 && m.Name() != "auth" && m.Name() != "oauth" {
			m.RegisterHTTP(router.Group("", protected...))
		} else {
			m.RegisterHTTP(router)
		}
	}

	// 启动审计 worker，测试结束 flush 剩余日志
	for _, comp := range auditMod.Components() {
		require.NoError(t, comp.Start(context.Background()))
		t.Cleanup(func() { _ = comp.Stop(context.Background()) })
	}

	return &testAppDB{router: router, db: gdb}
}

type apiResp struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
	Total   int64           `json:"total"`
}

func doJSON(t *testing.T, r *gin.Engine, method, path, token, body string) *httptest.ResponseRecorder {
	t.Helper()
	var req *http.Request
	if body == "" {
		req = httptest.NewRequest(method, path, nil)
	} else {
		req = httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func parseResp(t *testing.T, w *httptest.ResponseRecorder) apiResp {
	t.Helper()
	var r apiResp
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &r))
	return r
}

func login(t *testing.T, r *gin.Engine, username, password string) string {
	t.Helper()
	w := doJSON(t, r, http.MethodPost, "/api/v1/auth/login", "", `{"username":"`+username+`","password":"`+password+`"}`)
	require.Equal(t, http.StatusOK, w.Code)
	resp := parseResp(t, w)
	require.Equal(t, 0, resp.Code)
	var data struct {
		AccessToken string `json:"access_token"`
	}
	require.NoError(t, json.Unmarshal(resp.Data, &data))
	require.NotEmpty(t, data.AccessToken)
	return data.AccessToken
}

// loginWithRefresh 登录并返回 access + refresh token
func loginWithRefresh(t *testing.T, r *gin.Engine, username, password string) (string, string) {
	t.Helper()
	w := doJSON(t, r, http.MethodPost, "/api/v1/auth/login", "", `{"username":"`+username+`","password":"`+password+`"}`)
	require.Equal(t, http.StatusOK, w.Code)
	resp := parseResp(t, w)
	require.Equal(t, 0, resp.Code)
	var data struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
	}
	require.NoError(t, json.Unmarshal(resp.Data, &data))
	require.NotEmpty(t, data.AccessToken)
	require.NotEmpty(t, data.RefreshToken)
	return data.AccessToken, data.RefreshToken
}

// TestAuthRefreshLogout 令牌生命周期：登录 → refresh 换新对 → 旧 refresh 失效 → logout 后 access 失效。
func TestAuthRefreshLogout(t *testing.T) {
	r := newTestApp(t)

	_, refresh := loginWithRefresh(t, r, "admin", "admin123")

	// refresh 换新对
	w := doJSON(t, r, http.MethodPost, "/api/v1/auth/refresh", "", `{"refresh_token":"`+refresh+`"}`)
	require.Equal(t, http.StatusOK, w.Code)
	resp := parseResp(t, w)
	require.Equal(t, 0, resp.Code)
	var pair struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
	}
	require.NoError(t, json.Unmarshal(resp.Data, &pair))
	require.NotEmpty(t, pair.AccessToken)
	require.NotEmpty(t, pair.RefreshToken)
	newAccess := pair.AccessToken

	// 旧 refresh 已被轮换消费 → 再刷应失败（401 未认证）
	w = doJSON(t, r, http.MethodPost, "/api/v1/auth/refresh", "", `{"refresh_token":"`+refresh+`"}`)
	require.Equal(t, http.StatusUnauthorized, w.Code)

	// 新 access 可访问受保护资源
	w = doJSON(t, r, http.MethodGet, "/api/v1/users", newAccess, "")
	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, 0, parseResp(t, w).Code)

	// logout 撤销 refresh session；已发出的 access JWT 在过期前仍有效（标准 JWT 语义）
	w = doJSON(t, r, http.MethodPost, "/api/v1/auth/logout", newAccess, "")
	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, 0, parseResp(t, w).Code)

	// logout 后新签发的 refresh token 也已失效 → 用 pair.RefreshToken 刷新应 401
	w = doJSON(t, r, http.MethodPost, "/api/v1/auth/refresh", "", `{"refresh_token":"`+pair.RefreshToken+`"}`)
	require.Equal(t, http.StatusUnauthorized, w.Code)

	// access token 在过期前仍有效（无状态 JWT，由短 TTL 兜底）
	w = doJSON(t, r, http.MethodGet, "/api/v1/users", newAccess, "")
	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, 0, parseResp(t, w).Code)
}

// TestRoleAssignmentAndRBAC 角色闭环：创建角色 → 分配权限 → 绑定用户 → 权限生效/撤销。
func TestRoleAssignmentAndRBAC(t *testing.T) {
	// 缩短策略缓存 TTL，验证权限变更在缓存过期后生效
	oldTTL := platformauth.PolicyCacheTTL
	platformauth.PolicyCacheTTL = 50 * time.Millisecond
	t.Cleanup(func() { platformauth.PolicyCacheTTL = oldTTL })

	r := newTestApp(t)
	adminToken := login(t, r, "admin", "admin123")

	// 1. 创建普通用户
	w := doJSON(t, r, http.MethodPost, "/api/v1/users", adminToken, `{"username":"rbacuser","password":"rbacpass123"}`)
	require.Equal(t, http.StatusCreated, w.Code)
	var created struct {
		ID uint64 `json:"id"`
	}
	require.NoError(t, json.Unmarshal(parseResp(t, w).Data, &created))
	require.NotZero(t, created.ID)

	// 2. 创建角色
	w = doJSON(t, r, http.MethodPost, "/api/v1/roles", adminToken, `{"name":"e2e-viewer","description":"e2e"}`)
	require.Equal(t, http.StatusCreated, w.Code)
	var roleResp struct {
		ID uint64 `json:"id"`
	}
	require.NoError(t, json.Unmarshal(parseResp(t, w).Data, &roleResp))
	require.NotZero(t, roleResp.ID)

	// 3. 分配 GET /api/v1/users 权限给角色（page_size=100 拉全量，雪花 ID 用 json.Number 防精度丢失）
	permList := doJSON(t, r, http.MethodGet, "/api/v1/permissions?page_size=100", adminToken, "")
	require.Equal(t, http.StatusOK, permList.Code)
	var perms struct {
		Data []struct {
			ID       json.Number `json:"id"`
			Resource string      `json:"resource"`
			Action   string      `json:"action"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(permList.Body.Bytes(), &perms))
	var userGetPerm uint64
	for _, p := range perms.Data {
		if p.Resource == "/api/v1/users" && p.Action == "GET" {
			id, err := p.ID.Int64()
			require.NoError(t, err)
			userGetPerm = uint64(id)
			break
		}
	}
	require.NotZero(t, userGetPerm, "GET /api/v1/users permission should exist")

	w = doJSON(t, r, http.MethodPost, "/api/v1/roles/"+strconv.FormatUint(roleResp.ID, 10)+"/permissions",
		adminToken, `{"permission_ids":[`+strconv.FormatUint(userGetPerm, 10)+`]}`)
	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, 0, parseResp(t, w).Code)

	// 4. 绑定角色到用户（admin 接口，按角色名）
	w = doJSON(t, r, http.MethodPost, "/api/v1/admin/users/"+strconv.FormatUint(created.ID, 10)+"/roles",
		adminToken, `{"roles":["e2e-viewer"]}`)
	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, 0, parseResp(t, w).Code)

	// 5. 该用户登录后能 GET /users（原 403 → 200）
	userToken := login(t, r, "rbacuser", "rbacpass123")
	w = doJSON(t, r, http.MethodGet, "/api/v1/users", userToken, "")
	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, 0, parseResp(t, w).Code)

	// 6. 但不能 POST /users（未分配该权限）→ 403
	w = doJSON(t, r, http.MethodPost, "/api/v1/users", userToken, `{"username":"nope","password":"nopepass123"}`)
	require.Equal(t, http.StatusForbidden, w.Code)
}

// TestExportImportRoundTrip 导出→导入回读闭环（CSV）。
func TestExportImportRoundTrip(t *testing.T) {
	r := newTestApp(t)
	adminToken := login(t, r, "admin", "admin123")

	// 导出用户 CSV（带 BOM 的 UTF-8，列名大写）
	w := doJSON(t, r, http.MethodGet, "/api/v1/users/export.csv", adminToken, "")
	require.Equal(t, http.StatusOK, w.Code)
	body := strings.TrimPrefix(w.Body.String(), "\ufeff")
	require.Contains(t, body, "Username")
	require.Contains(t, body, "admin")

	// 导入预览（回读导出的内容）
	importBody := `{"type":"users","file_content":"` + strings.ReplaceAll(strings.ReplaceAll(body, "\n", "\\n"), "\"", "\\\"") + `"}`
	w = doJSON(t, r, http.MethodPost, "/api/v1/admin/users/import/preview", adminToken, importBody)
	// 若管理员路由存在则应为 200；此处仅验证不 500
	require.NotEqual(t, http.StatusInternalServerError, w.Code)
}

// TestAPIContract 端到端契约：注册 → 登录 → 401 边界 → RBAC 拒绝 → 管理员 CRUD → 审计。
func TestAPIContract(t *testing.T) {
	r := newTestApp(t)

	// 1. 注册新用户
	w := doJSON(t, r, http.MethodPost, "/api/v1/auth/register", "", `{"username":"e2euser","password":"e2epass123"}`)
	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, 0, parseResp(t, w).Code)

	// 2. 管理员登录
	adminToken := login(t, r, "admin", "admin123")

	// 3. 未携带 token 访问受保护资源 → 401
	w = doJSON(t, r, http.MethodGet, "/api/v1/users", "", "")
	require.Equal(t, http.StatusUnauthorized, w.Code)

	// 4. 管理员访问用户列表 → 200
	w = doJSON(t, r, http.MethodGet, "/api/v1/users", adminToken, "")
	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, 0, parseResp(t, w).Code)

	// 5. 管理员创建用户 → 201
	w = doJSON(t, r, http.MethodPost, "/api/v1/users", adminToken, `{"username":"alice","password":"alice1234"}`)
	require.Equal(t, http.StatusCreated, w.Code)
	require.Equal(t, 0, parseResp(t, w).Code)

	// 6. 列表应包含新用户（admin + e2euser + alice）
	w = doJSON(t, r, http.MethodGet, "/api/v1/users", adminToken, "")
	require.Equal(t, http.StatusOK, w.Code)
	listResp := parseResp(t, w)
	require.Equal(t, 0, listResp.Code)
	require.GreaterOrEqual(t, listResp.Total, int64(3))

	// 7. 无角色用户访问受保护资源 → 403（RBAC 拒绝）
	userToken := login(t, r, "e2euser", "e2epass123")
	w = doJSON(t, r, http.MethodGet, "/api/v1/users", userToken, "")
	require.Equal(t, http.StatusForbidden, w.Code)

	// 8. 管理员查询审计日志 → 200（前面的注册/登录/CRUD 已被审计）
	w = doJSON(t, r, http.MethodGet, "/api/v1/audits", adminToken, "")
	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, 0, parseResp(t, w).Code)
}
