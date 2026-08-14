package e2e

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"jimu/internal/config"
	"jimu/internal/contract"
	auditmodule "jimu/internal/modules/audit"
	auditdomain "jimu/internal/modules/audit/domain"
	authmodule "jimu/internal/modules/auth"
	"jimu/internal/modules/permission"
	"jimu/internal/modules/role"
	roledomain "jimu/internal/modules/role/domain"
	usermodule "jimu/internal/modules/user"
	userdomain "jimu/internal/modules/user/domain"
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

	router := gin.New()

	modules := []contract.Module{authMod, userMod, roleMod, permMod, auditMod}
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

	return router
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
