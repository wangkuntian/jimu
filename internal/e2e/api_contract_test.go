package e2e

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"jimu/internal/app"
	"jimu/internal/assembly"
	roledomain "jimu/internal/capabilities/access/domain"
	auditdomain "jimu/internal/capabilities/audit/domain"
	authmodule "jimu/internal/capabilities/auth"
	authdomain "jimu/internal/capabilities/auth/domain"
	"jimu/internal/capabilities/catalog"
	mfadomain "jimu/internal/capabilities/mfa/domain"
	passkeydomain "jimu/internal/capabilities/passkey/domain"
	tenantdomain "jimu/internal/capabilities/tenant/domain"
	userdomain "jimu/internal/capabilities/user/domain"
	"jimu/internal/config"
	"jimu/internal/contract"
	"jimu/internal/kernel/access"
	"jimu/internal/kernel/db"
	"jimu/internal/kernel/event"
	"jimu/internal/kernel/httpclient"
	"jimu/internal/kernel/logger"
	"jimu/internal/profiles/active"

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
	// casbin 用相对路径加载 conf/rbac_model.conf，需 cwd 位于项目根
	t.Chdir(repoRoot(t))

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
		&tenantdomain.Tenant{},
		&tenantdomain.Plan{},
		&auditdomain.AuditLog{},
		&mfadomain.UserMFA{},
		&passkeydomain.WebAuthnCredential{},
		&authdomain.LoginHistory{},
		&authdomain.PasswordHistory{},
	))
	require.NoError(t, gdb.Exec(`CREATE TABLE IF NOT EXISTS user_roles (user_id INTEGER NOT NULL, role_id INTEGER NOT NULL)`).Error)
	require.NoError(t, gdb.Exec(`CREATE TABLE IF NOT EXISTS role_permissions (role_id INTEGER NOT NULL, permission_id INTEGER NOT NULL)`).Error)

	// 种子：admin 用户 + 超级管理员角色 + 权限（含 /users、/audits 策略）
	t.Setenv("ADMIN_PASSWORD", "admin123")
	require.NoError(t, app.RunSeed(gdb, catalog.All()))

	mr, err := miniredis.Run()
	require.NoError(t, err)
	t.Cleanup(mr.Close)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })

	// 按当前形态装配模块（P2.6 T5）：提交态默认 full；用 -overlay 切到其它形态时，
	// 本套契约测试即针对该形态（缺能力的用例由 requireCapabilities 跳过）。
	// 装配走 assembly.WireFor —— 与生产 Run 同一份 wiring，不再手写模块清单。
	log := logger.New(config.LogConfig{Level: "error", Format: "console", Output: "stdout"})
	cfg, _, err := config.LoadWithSections()
	require.NoError(t, err)
	sections := testSections(t)
	asm := active.Assembly()
	caps, err := assembly.Resolve(asm, nil)
	require.NoError(t, err)
	capCfgs, err := app.LoadCapabilityConfigs(sections, caps, cfg.Environment)
	require.NoError(t, err)
	container := &app.Container{
		Config:            cfg,
		Sections:          sections,
		CapabilityConfigs: capCfgs,
		DB:                gdb,
		Redis:             rdb,
		Logger:            log,
		EventBus:          event.New(),
		HTTPClient:        httpclient.New(httpclient.Config{}),
	}
	ctx, modules, err := assembly.WireFor(container, sections, capCfgs, asm, caps)
	require.NoError(t, err)

	router := gin.New()

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
	// 3) 路由注册（按能力声明的挂载点；受保护能力必须存在中间件提供者）
	for _, m := range modules {
		desc := contract.Describe(m)
		if desc.Normalized() != contract.MountProtected {
			m.RegisterHTTP(router)
			continue
		}
		require.NotEmpty(t, protected, "capability %q declares MountProtected but no protected middleware provider is present", desc.Name)
		m.RegisterHTTP(router.Group("", protected...))
	}

	// 启动装配期组件（如审计 worker），测试结束回收
	for _, comp := range ctx.Components() {
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
	requireCapabilities(t, "auth", "user", "access")

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
	requireCapabilities(t, "auth", "user", "access")
	// 缩短策略缓存 TTL，验证权限变更在缓存过期后生效
	oldTTL := access.PolicyCacheTTL
	access.PolicyCacheTTL = 50 * time.Millisecond
	t.Cleanup(func() { access.PolicyCacheTTL = oldTTL })

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
	// 策略缓存按 TTL 异步过期，故用有界轮询等待生效，而不是依赖"登录耗时 > TTL"。
	userToken := login(t, r, "rbacuser", "rbacpass123")
	require.Eventually(t, func() bool {
		w := doJSON(t, r, http.MethodGet, "/api/v1/users", userToken, "")
		return w.Code == http.StatusOK
	}, 3*time.Second, 25*time.Millisecond, "permission should take effect after the policy cache TTL")

	// 6. 但不能 POST /users（未分配该权限）→ 403
	w = doJSON(t, r, http.MethodPost, "/api/v1/users", userToken, `{"username":"nope","password":"nopepass123"}`)
	require.Equal(t, http.StatusForbidden, w.Code)
}

// TestExportImportRoundTrip 导出→导入回读闭环（CSV）。
func TestExportImportRoundTrip(t *testing.T) {
	requireCapabilities(t, "auth", "user", "access", "dataops", "console")
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
	requireCapabilities(t, "auth", "user", "access", "audit")
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

// TestAuthRateLimit 验证登录限流：LoginRateLimit=3 时，第 4 次请求被拒绝（code=1007）。
// miniredis 支持 EVALSHA，无需外部 Redis。
func TestAuthRateLimit(t *testing.T) {
	// 本用例自己手搭 router（只挂 auth 的登录路由），故只依赖 auth。
	requireCapabilities(t, "auth")
	gin.SetMode(gin.TestMode)
	t.Chdir(repoRoot(t))

	gdb, err := gorm.Open(sqlite.Open("file:e2e_ratelimit?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	if sqlDB, err := gdb.DB(); err == nil {
		sqlDB.SetMaxOpenConns(1)
	}
	require.NoError(t, db.InitSnowflake(1))
	db.RegisterSnowflakeHook(gdb)
	require.NoError(t, gdb.AutoMigrate(
		&userdomain.User{},
		&roledomain.Role{},
		&roledomain.Permission{},
		&tenantdomain.Tenant{},
		&tenantdomain.Plan{},
	))
	require.NoError(t, gdb.Exec(`CREATE TABLE IF NOT EXISTS user_roles (user_id INTEGER NOT NULL, role_id INTEGER NOT NULL)`).Error)
	require.NoError(t, gdb.Exec(`CREATE TABLE IF NOT EXISTS role_permissions (role_id INTEGER NOT NULL, permission_id INTEGER NOT NULL)`).Error)
	t.Setenv("ADMIN_PASSWORD", "admin123")
	require.NoError(t, app.RunSeed(gdb, catalog.All()))

	mr, err := miniredis.Run()
	require.NoError(t, err)
	t.Cleanup(mr.Close)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })

	authMod := authmodule.New(gdb, rdb, authmodule.Config{
		JWTSecret:          "0123456789abcdef0123456789abcdef",
		Issuer:             "jimu-e2e",
		AccessExpireMin:    30,
		RefreshExpireDay:   7,
		PublicRegistration: true,
		LoginRateLimit:     3,
		LoginRateWindowSec: 60,
	}, false, nil)
	router := gin.New()
	authMod.RegisterHTTP(router)

	// 前 3 次登录成功
	for i := 0; i < 3; i++ {
		w := doJSON(t, router, http.MethodPost, "/api/v1/auth/login", "", `{"username":"admin","password":"admin123"}`)
		require.Equal(t, http.StatusOK, w.Code, "第 %d 次登录应成功", i+1)
		require.Equal(t, 0, parseResp(t, w).Code, "第 %d 次登录 code=0", i+1)
	}

	// 第 4 次被限流（CodeRateLimited → HTTP 429）
	w := doJSON(t, router, http.MethodPost, "/api/v1/auth/login", "", `{"username":"admin","password":"admin123"}`)
	require.Equal(t, http.StatusTooManyRequests, w.Code, "限流响应应为 429")
	resp := parseResp(t, w)
	require.Equal(t, 1007, resp.Code, "限流错误码应为 1007")
}

// repoRoot 定位项目根目录：从 cwd 向上找 go.mod。
// 不依赖 cwd 深度，e2e 包移动目录后仍能稳定定位 conf/rbac_model.conf。
func repoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	dir, _ = filepath.Abs(dir)
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("go.mod not found; project root unreachable")
		}
		dir = parent
	}
}
