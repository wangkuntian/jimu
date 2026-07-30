# Jimu Runtime and Security Baseline Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [x]`) syntax for tracking.

**Goal:** 建立可验证的生产配置、独立管理端点、安全认证会话、可靠审计和统一应用生命周期。

**Architecture:** 保留现有 Gin/Gorm/Redis 模块结构，新增显式 `Application` 生命周期与 management server。认证的 access token 保持无状态，refresh session 使用 Redis 持久化；审计使用有界 worker 队列批量写入 MariaDB。

**Tech Stack:** Go 1.26、Gin、Gorm、MariaDB、go-redis、JWT v5、Zap、Prometheus client_golang。

## Global Constraints

- 部署目标为 Docker Compose / 单机，同时支持内网中后台与可配置公开注册。
- 除 MariaDB 和 Redis 外不增加必需服务。
- 保留 `/api/v1` 与 `{code,message,data}` 响应外形，其他接口允许不兼容调整。
- 所有行为修改遵循 TDD：先看到聚焦测试正确失败，再写最小实现。
- 不覆盖现有 `Dockerfile`、`docker-compose.yml` 用户修改；合并这些文件时逐行保留已有差异。
- 未经用户明确要求，不执行 `git commit`。

---

## File Structure

- `internal/config/config.go`：运行、管理、HTTP 安全、审计与认证配置结构。
- `internal/config/validate.go`：环境感知、无敏感值泄漏的配置校验。
- `internal/platform/observability/health.go`：liveness/readiness handler 与依赖探测接口。
- `internal/platform/http/server.go`：单个 HTTP server 的构造、启动和关闭。
- `internal/platform/http/management.go`：独立 management server 和 pprof 开关。
- `internal/app/application.go`：资源所有权、启动顺序与反向关闭顺序。
- `internal/modules/audit/application/worker.go`：有界审计队列、批写和 drain。
- `internal/platform/auth/jwt.go`：带 token type、issuer、subject 与 JTI 的 JWT。
- `internal/platform/auth/session.go`：Redis refresh session 的保存、轮换和撤销。
- `internal/platform/http/middleware/security.go`：请求体上限、CORS 与可信代理配置。
- `internal/shared/response/response.go`：HTTP 状态码与稳定业务错误映射。

## Task 1: Production-aware configuration validation

**Files:**
- Modify: `internal/config/config.go`
- Create: `internal/config/validate.go`
- Modify: `internal/config/config_test.go`
- Modify: `configs/app.yaml`
- Modify: `configs/app.test.yaml`
- Modify: `configs/app.prod.yaml`

**Interfaces:**
- Produces: `func (c *Config) Validate(env string) error`
- Produces: `ManagementConfig`, expanded `HTTPConfig`, `AuditConfig`, and expanded `AuthConfig`
- Consumes: environment name from `JIMU_ENV`

- [x] **Step 1: Write failing validation tests**

Add table tests that construct `Config` directly, so production validation does not depend on process environment or YAML lookup:

```go
func validProdConfig() Config {
	return Config{
		HTTP: HTTPConfig{Host: "0.0.0.0", Port: 8080, Mode: HTTPModeRelease,
			ReadHeaderTimeoutSec: 5, ReadTimeoutSec: 15, WriteTimeoutSec: 30,
			IdleTimeoutSec: 60, ShutdownTimeoutSec: 30, MaxBodyBytes: 1 << 20,
			TrustedProxies: []string{"127.0.0.1"}, AllowedOrigins: []string{"https://admin.example.com"}},
		Management: ManagementConfig{Host: "127.0.0.1", Port: 9090, ProbeTimeoutSec: 2},
		DB: DBConfig{Host: "mariadb", Port: 3306, User: "jimu", Password: "strong-db-password", Database: "jimu", MaxOpen: 20, MaxIdle: 5},
		Redis: RedisConfig{Addr: "redis:6379"},
		Log: LogConfig{Level: LogLevelInfo, Format: LogFormatJSON, Output: "stdout"},
		Auth: AuthConfig{JWTSecret: strings.Repeat("x", 32), Issuer: "jimu", AccessExpireMin: 30, RefreshExpireDay: 7},
		Server: ServerConfig{TimeoutSec: 30, RateLimitRate: 100, RateLimitBurst: 200},
		Audit: AuditConfig{QueueSize: 256, BatchSize: 50, FlushIntervalMS: 500},
	}
}

func TestValidateProdRejectsInsecureValues(t *testing.T) {
	tests := []struct {
		name string
		mutate func(*Config)
		key string
	}{
		{"default JWT secret", func(c *Config) { c.Auth.JWTSecret = "change-me-in-production" }, "auth.jwt_secret"},
		{"short JWT secret", func(c *Config) { c.Auth.JWTSecret = "short" }, "auth.jwt_secret"},
		{"default DB password", func(c *Config) { c.DB.Password = "root" }, "db.password"},
		{"invalid management port", func(c *Config) { c.Management.Port = 0 }, "management.port"},
		{"wildcard CORS", func(c *Config) { c.HTTP.AllowedOrigins = []string{"*"} }, "http.allowed_origins"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := validProdConfig()
			tt.mutate(&cfg)
			err := cfg.Validate("prod")
			if err == nil || !strings.Contains(err.Error(), tt.key) {
				t.Fatalf("Validate() error = %v, want key %q", err, tt.key)
			}
			if strings.Contains(err.Error(), cfg.Auth.JWTSecret) || strings.Contains(err.Error(), cfg.DB.Password) {
				t.Fatalf("validation error leaked a secret: %v", err)
			}
		})
	}
}

func TestLoadOverridesNewNestedFields(t *testing.T) {
	t.Setenv("JIMU__MANAGEMENT__PORT", "9191")
	t.Setenv("JIMU__AUTH__PUBLIC_REGISTRATION", "true")
	t.Setenv("JIMU__HTTP__MAX_BODY_BYTES", "2048")
	cfg, err := Load()
	if err != nil { t.Fatal(err) }
	if cfg.Management.Port != 9191 || !cfg.Auth.PublicRegistration || cfg.HTTP.MaxBodyBytes != 2048 {
		t.Fatalf("nested overrides not applied: %#v", cfg)
	}
}
```

- [x] **Step 2: Run the test and verify RED**

Run: `go test ./internal/config -run 'TestValidateProdRejectsInsecureValues' -v`

Expected: compile failure because the new configuration fields and `Validate` do not exist.

- [x] **Step 3: Add exact configuration types and validation**

Add these fields without introducing a generic map-based configuration layer:

```go
type ManagementConfig struct {
	Host            string `mapstructure:"host"`
	Port            int    `mapstructure:"port"`
	EnablePprof     bool   `mapstructure:"enable_pprof"`
	ProbeTimeoutSec int    `mapstructure:"probe_timeout_sec"`
}

type AuditConfig struct {
	QueueSize       int `mapstructure:"queue_size"`
	BatchSize       int `mapstructure:"batch_size"`
	FlushIntervalMS int `mapstructure:"flush_interval_ms"`
}
```

Extend `HTTPConfig` with the exact timeout, body-size, proxy and origin fields used by the test. Extend `AuthConfig` with `Issuer string`, `PublicRegistration bool`, and login/registration rate-limit settings. Add `Management ManagementConfig` and `Audit AuditConfig` to `Config`.

Implement `Validate` with field-specific errors:

```go
func (c *Config) Validate(env string) error {
	if err := c.validateCommon(); err != nil { return err }
	if env != "prod" { return nil }
	if len(c.Auth.JWTSecret) < 32 || c.Auth.JWTSecret == "change-me-in-production" || strings.Contains(c.Auth.JWTSecret, "${") {
		return errors.New("invalid auth.jwt_secret")
	}
	if c.DB.Password == "" || c.DB.Password == "root" || strings.Contains(c.DB.Password, "${") {
		return errors.New("invalid db.password")
	}
	if c.Management.Port < 1 || c.Management.Port > 65535 {
		return errors.New("invalid management.port")
	}
	for _, origin := range c.HTTP.AllowedOrigins {
		if origin == "*" { return errors.New("invalid http.allowed_origins") }
	}
	}
	return nil
}
```

Make `Load` call `cfg.Validate(env)` after environment overrides. Extend `applyEnvOverrides` for every new field, parsing booleans with `strconv.ParseBool`, integers with `strconv.Atoi`, and comma-separated origin/proxy lists with trimming and empty-item removal. Invalid environment values return a configuration error instead of being silently ignored. Update YAML files to contain literal non-secret defaults; production secrets must be supplied through `JIMU__DB__PASSWORD` and `JIMU__AUTH__JWT_SECRET`, not `${VAR}` YAML interpolation.

- [x] **Step 4: Verify GREEN and configuration compatibility**

Run:

```bash
go test ./internal/config -v
JIMU_ENV=test go test ./internal/config -run TestLoad -v
```

Expected: all configuration tests pass and no error includes a configured secret.

## Task 2: Correct liveness/readiness and isolate management endpoints

**Files:**
- Modify: `go.mod`
- Modify: `go.sum`
- Rewrite: `internal/platform/observability/health.go`
- Create: `internal/platform/observability/health_test.go`
- Create: `internal/platform/http/management.go`
- Create: `internal/platform/http/management_test.go`
- Modify: `internal/app/bootstrap.go`

**Interfaces:**
- Produces: `type Checker interface { Ping(context.Context) error }`
- Produces: `func HealthRouter(readiness *observability.Readiness, enablePprof bool) http.Handler`
- Produces: `func NewManagementServer(cfg config.ManagementConfig, handler http.Handler) *Server`

- [x] **Step 1: Write failing handler tests**

Use controllable checker functions:

```go
type checkerFunc func(context.Context) error
func (f checkerFunc) Ping(ctx context.Context) error { return f(ctx) }

func TestReadinessStatus(t *testing.T) {
	tests := []struct { name string; dbErr, redisErr error; want int }{
		{"ready", nil, nil, http.StatusOK},
		{"database down", errors.New("down"), nil, http.StatusServiceUnavailable},
		{"redis down", nil, errors.New("down"), http.StatusServiceUnavailable},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := gin.New()
			RegisterHealth(r, NewReadiness(50*time.Millisecond,
				checkerFunc(func(context.Context) error { return tt.dbErr }),
				checkerFunc(func(context.Context) error { return tt.redisErr })))
			w := httptest.NewRecorder()
			r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/readyz", nil))
			if w.Code != tt.want { t.Fatalf("status = %d, want %d", w.Code, tt.want) }
		})
	}
}
```

Add a management-router test asserting `/metrics` exists, `/livez` and `/readyz` exist, and `/debug/pprof/` returns 404 when `enablePprof` is false.

- [x] **Step 2: Run tests and verify RED**

Run: `go test ./internal/platform/observability ./internal/platform/http -run 'Test(ReadinessStatus|Management)' -v`

Expected: compile failure because `Readiness`, `RegisterHealth`, and management router do not exist.

- [x] **Step 3: Implement health contracts and management router**

Add the pinned Prometheus dependency before writing the handler:

```bash
go get github.com/prometheus/client_golang@v1.24.1
```

Implement readiness without returning dependency error details:

```go
type Readiness struct { timeout time.Duration; checkers []Checker }

func (r *Readiness) Ready(ctx context.Context) bool {
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()
	for _, checker := range r.checkers {
		if err := checker.Ping(ctx); err != nil { return false }
	}
	return true
}
```

`RegisterHealth` returns plain JSON `{"status":"up"}` for `/livez`, `{"status":"ready"}` with 200 for ready, and `{"status":"unavailable"}` with 503 otherwise. Adapt `*sql.DB` and `*redis.Client` behind small checker adapters.

Build the management router with `promhttp.Handler()` and conditionally register standard-library pprof handlers only when enabled. Remove `/health`, `/debug/metrics`, `/debug/pprof`, and Swagger diagnostics from the public router; Swagger remains a development-only public API concern and is not served by management.

- [x] **Step 4: Verify GREEN**

Run: `go test ./internal/platform/observability ./internal/platform/http -v`

Expected: health and endpoint-exposure tests pass.

## Task 3: Explicit application lifecycle and resource ownership

**Files:**
- Create: `internal/app/application.go`
- Create: `internal/app/application_test.go`
- Modify: `internal/contract/module.go`
- Modify: `internal/app/container.go`
- Modify: `internal/app/bootstrap.go`
- Modify: `internal/platform/http/server.go`
- Modify: `internal/platform/logger/zap.go`
- Modify: `cmd/server/main.go`

**Interfaces:**
- Produces: `contract.Component` and `contract.ComponentProvider`
- Produces: `func (a *Application) Run(ctx context.Context) error`
- Produces: `func Bootstrap(ctx context.Context, modules ...contract.Module) (*Application, error)`

- [x] **Step 1: Write lifecycle-order tests**

```go
type fakeComponent struct { name string; calls *[]string; startErr error }
func (f *fakeComponent) Start(context.Context) error {
	*f.calls = append(*f.calls, "start:"+f.name); return f.startErr
}
func (f *fakeComponent) Stop(context.Context) error {
	*f.calls = append(*f.calls, "stop:"+f.name); return nil
}

func TestApplicationStopsComponentsInReverseOrder(t *testing.T) {
	var calls []string
	a := NewApplication(time.Second,
		&fakeComponent{name:"worker", calls:&calls},
		&fakeComponent{name:"management", calls:&calls},
		&fakeComponent{name:"public", calls:&calls})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := a.Run(ctx); err != nil { t.Fatal(err) }
	want := []string{"start:worker", "start:management", "start:public", "stop:public", "stop:management", "stop:worker"}
	if !reflect.DeepEqual(calls, want) { t.Fatalf("calls = %v, want %v", calls, want) }
}
```

Also test that a start failure stops only components that successfully started.

- [x] **Step 2: Run and verify RED**

Run: `go test ./internal/app -run TestApplication -v`

Expected: compile failure because `Application` and lifecycle interfaces do not exist.

- [x] **Step 3: Implement lifecycle and remove nested initialization**

Define lifecycle contracts outside `app` so modules do not create an import cycle:

```go
type Component interface {
	Start(context.Context) error
	Stop(context.Context) error
}

type ComponentProvider interface {
	Components() []Component
}
```

`Application.Run` starts components in declared order, blocks on context cancellation or component error, and stops successfully started components in reverse order with one shared shutdown deadline. Register workers before HTTP servers so reverse shutdown stops public traffic before draining workers. Combine errors with `errors.Join`.

Make `Server.Start` call `net.Listen` followed by `Serve` in a goroutine and report non-`http.ErrServerClosed` errors. Make `Server.Stop` call `Shutdown(ctx)`. Do not call `os.Exit`, sleep for a fixed second, or maintain a second active-request counter because `http.Server.Shutdown` already drains active handlers.

Make `Bootstrap` accept an already constructed `Container`; remove the second config/DB/Redis creation currently performed inside `Bootstrap`. Add `Container.Close(ctx)` to close Redis, close the underlying `sql.DB`, and call logger `Sync` without treating stdout `EINVAL` as fatal.

`cmd/server/main.go` owns signal handling through `signal.NotifyContext`, constructs the container once, builds both servers and workers, calls `Application.Run`, logs the final error, and exits nonzero only after cleanup.

- [x] **Step 4: Verify GREEN and process build**

Run:

```bash
go test ./internal/app ./internal/platform/http -v
go build ./cmd/server ./cmd/cli
```

Expected: lifecycle tests pass and both entry points build.

## Task 4: Safe bounded audit worker

**Files:**
- Modify: `internal/modules/audit/domain/audit.go`
- Modify: `internal/modules/audit/infrastructure/mysql_repository.go`
- Create: `internal/modules/audit/application/worker.go`
- Create: `internal/modules/audit/application/worker_test.go`
- Modify: `internal/modules/audit/interfaces/middleware.go`
- Create: `internal/modules/audit/interfaces/middleware_test.go`
- Modify: `internal/modules/audit/module.go`
- Modify: `internal/contract/module.go`
- Modify: `internal/app/bootstrap.go`

**Interfaces:**
- Changes: `AuditRepository` adds `CreateBatch(context.Context, []AuditLog) error`
- Produces: `func NewWorker(repo domain.AuditRepository, cfg config.AuditConfig, log *logger.Logger) *Worker`
- Produces: `func (w *Worker) Enqueue(domain.AuditLog) bool`
- `Worker` implements `contract.Component`; audit Module implements `contract.ComponentProvider` and `contract.HTTPMiddlewareProvider`

- [x] **Step 1: Write regression tests for anonymous requests and cancellation**

```go
func TestAuditMiddlewareAllowsAnonymousRequest(t *testing.T) {
	queue := &fakeQueue{}
	r := gin.New()
	r.Use(AuditMiddleware(queue))
	r.GET("/public", func(c *gin.Context) { c.Status(http.StatusNoContent) })
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/public", nil))
	if w.Code != http.StatusNoContent { t.Fatalf("status = %d", w.Code) }
	if len(queue.logs) != 1 || queue.logs[0].UserID != 0 || queue.logs[0].Username != "" {
		t.Fatalf("anonymous audit = %#v", queue.logs)
	}
}
```

Write worker tests using a fake batch repository to prove: a full queue returns false without blocking; a flush writes the expected batch; `Stop` drains accepted records using its shutdown context; request-context cancellation does not cancel the batch write.

- [x] **Step 2: Run and verify RED**

Run: `go test ./internal/modules/audit/... -run 'Test(AuditMiddlewareAllowsAnonymousRequest|Worker)' -v`

Expected: anonymous test panics with the current middleware or fails to compile against the queue interface; worker tests fail because `Worker` does not exist.

- [x] **Step 3: Implement safe extraction and worker queue**

Define the middleware dependency narrowly:

```go
type Queue interface { Enqueue(domain.AuditLog) bool }

func optionalUint64(c *gin.Context, key string) uint64 {
	v, ok := c.Get(key); if !ok { return 0 }
	n, _ := v.(uint64); return n
}

func optionalString(c *gin.Context, key string) string {
	v, ok := c.Get(key); if !ok { return "" }
	s, _ := v.(string); return s
}
```

Do not read or persist request bodies. Create the audit record after `c.Next()` and call `Enqueue`; record a structured warning when the queue rejects it. The worker owns `chan domain.AuditLog`, ticker-based batch flushing, and a private lifecycle context rather than request contexts. `CreateBatch` uses one Gorm `Create(&logs)` call.

Add the cross-cutting middleware contract:

```go
type HTTPMiddlewareProvider interface {
	HTTPMiddleware() []gin.HandlerFunc
}
```

Bootstrap performs two module passes: first install every `HTTPMiddlewareProvider` on the root engine, then register every module route. This guarantees audit middleware wraps routes regardless of module ordering. It collects `ComponentProvider` components before management and public HTTP servers, so shutdown stops HTTP traffic before draining the audit worker.

- [x] **Step 4: Verify GREEN and race safety**

Run:

```bash
go test ./internal/modules/audit/... -v
go test -race ./internal/modules/audit/... -v
```

Expected: all audit tests pass with no race reports.

## Task 5: HTTP status mapping and security boundaries

**Files:**
- Modify: `internal/shared/errors/errors.go`
- Modify: `internal/shared/response/response.go`
- Create: `internal/shared/response/response_test.go`
- Create: `internal/platform/http/middleware/security.go`
- Create: `internal/platform/http/middleware/security_test.go`
- Modify: `internal/platform/http/server.go`

**Interfaces:**
- Produces: `func StatusForCode(code int) int`
- Produces: `func Security(cfg config.HTTPConfig) gin.HandlerFunc`
- Produces: `func ConfigureTrustedProxies(engine *gin.Engine, proxies []string) error`

- [x] **Step 1: Write failing response and middleware tests**

Test exact mappings: invalid parameter to 400, unauthorized to 401, forbidden to 403, not found to 404, rate limited to 429, timeout to 504, internal error to 500. Test that an unknown wrapped error returns 500 and public message `internal error` without its cause.

```go
func TestFailUsesHTTPStatusAndHidesCause(t *testing.T) {
	r := gin.New()
	r.GET("/", func(c *gin.Context) {
		Fail(c, appErrs.Wrap(appErrs.CodeInternalError, "database failed", errors.New("secret DSN")))
	})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))
	if w.Code != http.StatusInternalServerError { t.Fatalf("status = %d", w.Code) }
	if strings.Contains(w.Body.String(), "database failed") || strings.Contains(w.Body.String(), "secret DSN") {
		t.Fatalf("response leaked internal error: %s", w.Body.String())
	}
}
```

Test that an oversized body returns 413, an unlisted Origin receives no allow-origin header, an allowed Origin is echoed with `Vary: Origin`, and OPTIONS returns 204.

- [x] **Step 2: Run and verify RED**

Run: `go test ./internal/shared/response ./internal/platform/http/middleware -v`

Expected: current `Fail` returns 200 and current CORS accepts `*`, causing assertions to fail.

- [x] **Step 3: Implement stable mappings and one security middleware**

Add `CodeRateLimited` and `CodeTimeout`. `response.Fail` maps known application errors to HTTP status, always hides internal messages, and includes request ID in the response body only through a new optional `request_id` field.

`Security` uses `http.MaxBytesReader`, exact origin membership, `Vary: Origin`, explicit allowed methods/headers, and handles preflight before calling downstream handlers. Configure Gin trusted proxies once during router creation. Populate all `http.Server` timeout fields from `HTTPConfig`.

Remove the existing wildcard `CORS()` from `middleware.go`. Replace the goroutine-based `Timeout` middleware with context deadline propagation that does not concurrently write through Gin; rely on server write timeout for hard network bounds and have handlers/services honor the context.

- [x] **Step 4: Verify GREEN**

Run:

```bash
go test ./internal/shared/response ./internal/platform/http/middleware ./internal/platform/http -v
go vet ./internal/shared/... ./internal/platform/http/...
```

Expected: status, secrecy, body-size, CORS and timeout configuration tests pass.

## Task 6: Typed JWTs and Redis refresh sessions

**Files:**
- Modify: `go.mod`
- Modify: `go.sum`
- Rewrite: `internal/platform/auth/jwt.go`
- Create: `internal/platform/auth/jwt_test.go`
- Create: `internal/platform/auth/session.go`
- Create: `internal/platform/auth/session_test.go`
- Modify: `internal/modules/auth/application/service.go`
- Create: `internal/modules/auth/application/service_test.go`
- Modify: `internal/modules/auth/domain/auth.go`
- Modify: `internal/modules/auth/interfaces/handler.go`
- Modify: `internal/modules/auth/interfaces/router.go`
- Modify: `internal/modules/auth/module.go`
- Modify: `internal/platform/auth/middleware.go`

**Interfaces:**
- Produces: `func (j *JWT) GenerateAccess(userID uint64, sessionID string) (string, error)`
- Produces: `func (j *JWT) GenerateRefresh(userID uint64, sessionID string) (token string, claims Claims, err error)`
- Produces: `func (j *JWT) Parse(token, expectedType string) (*Claims, error)`
- Produces: `type SessionStore interface { Create; Rotate; Revoke; RevokeAll }` with exact signatures below

- [x] **Step 1: Write JWT type-confusion tests**

```go
func TestParseRejectsWrongTokenType(t *testing.T) {
	j := New(strings.Repeat("s", 32), "jimu", 30, 7)
	refresh, _, err := j.GenerateRefresh(42, "session-1")
	if err != nil { t.Fatal(err) }
	if _, err := j.Parse(refresh, TokenTypeAccess); err == nil {
		t.Fatal("refresh token accepted as access token")
	}
}
```

Also assert issuer validation, nonempty subject/JTI/session ID, exact algorithm HS256, expiry, and access/refresh type separation.

- [x] **Step 2: Run JWT tests and verify RED**

Run: `go test ./internal/platform/auth -run TestParseRejectsWrongTokenType -v`

Expected: compile failure because the typed JWT API does not exist.

- [x] **Step 3: Implement typed JWT claims**

Use this claim shape:

```go
const ( TokenTypeAccess = "access"; TokenTypeRefresh = "refresh" )
type Claims struct {
	UserID    uint64 `json:"user_id"`
	SessionID string `json:"sid"`
	TokenType string `json:"token_type"`
	jwt.RegisteredClaims
}
```

Use `jwt.NewParser(jwt.WithValidMethods([]string{"HS256"}), jwt.WithIssuer(j.issuer))`. Put user ID in both `UserID` and decimal `Subject`, generate UUID JTI values, and reject claims whose token type differs from `expectedType`.

- [x] **Step 4: Write refresh-session rotation tests**

Add the pinned in-memory Redis test dependency:

```bash
go get github.com/alicebob/miniredis/v2@v2.38.0
```

Define the interface precisely:

```go
type SessionStore interface {
	Create(ctx context.Context, userID uint64, sessionID, tokenID string, ttl time.Duration) error
	Rotate(ctx context.Context, userID uint64, sessionID, oldTokenID, newTokenID string, ttl time.Duration) error
	Revoke(ctx context.Context, userID uint64, sessionID string) error
	RevokeAll(ctx context.Context, userID uint64) error
}
```

Use `miniredis` in unit tests. Prove that `Rotate` succeeds once, reusing `oldTokenID` fails, `Revoke` blocks future rotation, and `RevokeAll` removes every session index for one user without touching another user.

- [x] **Step 5: Run session tests and verify RED**

Run: `go test ./internal/platform/auth -run 'TestSession' -v`

Expected: compile failure because Redis session storage does not exist.

- [x] **Step 6: Implement atomic Redis sessions**

Store only token IDs and metadata, never raw refresh tokens. Use keys `jimu:auth:session:<sessionID>` and a Set `jimu:auth:user:<userID>:sessions`. Implement rotation with a Lua script that compares the stored token ID and updates it atomically while preserving the user index. Return exported sentinel errors `ErrSessionNotFound` and `ErrTokenReuse`.

- [x] **Step 7: Write service tests for login, refresh, and revocation**

Use fake user repository and fake session store to verify:

- Missing user and wrong password both return `CodeInvalidCredentials` with the same public message.
- Disabled user cannot log in.
- Login creates one refresh session.
- Refresh parses only refresh tokens and rotates the matching session.
- Logout revokes the current session.
- Logout-all revokes every session for the authenticated user.

The service API becomes:

```go
func (s *AuthService) Login(ctx context.Context, username, password string) (*TokenPair, error)
func (s *AuthService) Refresh(ctx context.Context, refreshToken string) (*TokenPair, error)
func (s *AuthService) Logout(ctx context.Context, userID uint64, sessionID string) error
func (s *AuthService) LogoutAll(ctx context.Context, userID uint64) error
```

- [x] **Step 8: Run service tests and verify RED**

Run: `go test ./internal/modules/auth/application -v`

Expected: tests fail because current login leaks user existence, ignores user status, and has no session store.

- [x] **Step 9: Implement service and HTTP behavior**

Normalize usernames with strings.TrimSpace and strings.ToLower before repository lookup. Add `CodeInvalidCredentials`. Generate a random session ID on login; create Redis state only after both tokens are signed. On refresh, verify token type first, generate a new token ID, then atomically rotate the session before returning the new pair.

Add authenticated `POST /api/v1/auth/logout` and `POST /api/v1/auth/logout-all`. Keep `/login` and `/refresh` public. Register `/register` only when `AuthConfig.PublicRegistration` is true. Authentication middleware parses only access tokens and stores `user_id` and `session_id` in Gin context.

- [x] **Step 10: Verify GREEN and race safety**

Run:

```bash
go test ./internal/platform/auth ./internal/modules/auth/... -v
go test -race ./internal/platform/auth ./internal/modules/auth/... -v
```

Expected: typed-token, rotation, reuse, logout, disabled-user and credential-secrecy tests pass without races.

## Task 7: Redis authentication rate limiting and public-registration mode

**Files:**
- Create: `internal/platform/auth/limiter.go`
- Create: `internal/platform/auth/limiter_test.go`
- Modify: `internal/modules/auth/interfaces/handler.go`
- Modify: `internal/modules/auth/interfaces/router.go`
- Modify: `internal/config/config.go`

**Interfaces:**
- Produces: `func (l *Limiter) Allow(ctx context.Context, scope, key string, limit int, window time.Duration) (bool, error)`
- Consumes: client IP and normalized username from login/register handlers

- [x] **Step 1: Write window-limit tests**

Using `miniredis`, assert the first N attempts are allowed, N+1 is rejected, login and registration scopes are isolated, and a Redis error fails closed in production mode.

```go
for i := 0; i < 5; i++ {
	ok, err := limiter.Allow(ctx, "login", "ip:127.0.0.1", 5, time.Minute)
	if err != nil || !ok { t.Fatalf("attempt %d: ok=%v err=%v", i, ok, err) }
}
ok, err := limiter.Allow(ctx, "login", "ip:127.0.0.1", 5, time.Minute)
if err != nil || ok { t.Fatalf("sixth attempt: ok=%v err=%v", ok, err) }
```

- [x] **Step 2: Run and verify RED**

Run: `go test ./internal/platform/auth -run TestLimiter -v`

Expected: compile failure because `Limiter` does not exist.

- [x] **Step 3: Implement Redis fixed-window limiter**

Use one Lua script containing `INCR` and first-write `PEXPIRE` so increment and expiry are atomic. Hash normalized usernames before adding them to keys; never expose usernames in Redis key names. Return `CodeRateLimited` and HTTP 429 with a generic message from handlers.

Apply both IP and username scopes before password verification. Apply IP scope to registration. Do not reuse the current unbounded in-memory `visitors` map for authentication endpoints.

- [x] **Step 4: Verify GREEN**

Run: `go test ./internal/platform/auth ./internal/modules/auth/... -v`

Expected: limiter and handler tests pass.

## Task 8: Phase verification and documentation update

**Files:**
- Create: `work/test_runtime_security.sh`
- Modify: `README.md`
- Modify: `.env.example`
- Modify: `docs/openapi/docs.go` through `make swagger`
- Modify: `Dockerfile` only where required for management/lifecycle behavior
- Modify: `docker-compose.yml` while preserving existing user edits

**Interfaces:**
- Documents: production configuration, registration mode, management endpoint isolation, Token lifecycle, and shutdown behavior

- [x] **Step 1: Add executable smoke checks**

Create executable `work/test_runtime_security.sh`. It must run:

```bash
docker compose config --quiet
go test ./...
go test -race ./internal/app ./internal/config ./internal/platform/auth ./internal/platform/http/... ./internal/modules/auth/... ./internal/modules/audit/...
go vet ./...
go build ./cmd/server ./cmd/cli
```

It must also start Compose, poll the private management `/livez` and `/readyz` from inside the server container, verify public port 8080 does not serve `/debug/pprof/`, then stop Compose without removing named data volumes.

- [x] **Step 2: Update configuration and API documentation**

Remove documented default administrator passwords. Document `JIMU__AUTH__JWT_SECRET`, registration switch, management listener, timeout fields, trusted proxies, allowed origins, audit queue settings, and logout endpoints. Regenerate Swagger and confirm no diff appears on a second generation.

- [x] **Step 3: Run full phase verification**

Run:

```bash
gofmt -w cmd internal
git diff --check
go test ./...
go test -race ./...
go vet ./...
go build ./cmd/server ./cmd/cli
docker compose config --quiet
docker build -t jimu:runtime-security-test .
```

Expected: every command exits 0. If Docker is unavailable, report Docker checks as unverified rather than claiming completion.

- [x] **Step 4: Review workspace scope**

Run `git status --short` and `git diff --stat`. Confirm the pre-existing `Dockerfile`, `docker-compose.yml`, and `server` changes are preserved and that every newly changed file traces to this phase. Do not stage or commit without explicit user instruction.
