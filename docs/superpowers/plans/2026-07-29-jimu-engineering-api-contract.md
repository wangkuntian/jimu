# Jimu Engineering Quality and API Contract Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 建立可测试、稳定、可由 CI 兜底的 API 契约与工程质量基线。

**Architecture:** 保留现有 Gin/Gorm/Redis 模块结构。API 契约集中在 `internal/shared/response`、`internal/shared/pagination`、HTTP handler 测试和 OpenAPI 生成检查；集成 smoke 使用 Docker Compose 启动 MariaDB/Redis/server，不增加新必需服务。

**Tech Stack:** Go 1.26、Gin、Gorm、MariaDB、Redis、swaggo/swag v1.8.12、Docker Compose、GitHub Actions。

## Global Constraints

- 公共 API 保持在 `/api/v1` 下。
- 响应继续使用 `{code,message,data}` 外形。
- HTTP 状态码表达协议结果，稳定业务错误码表达业务结果。
- 对外错误不包含 SQL、文件路径、堆栈或包装后的基础设施错误。
- 分页、排序和过滤使用统一契约，并限制最大页大小。
- Request ID 写入响应 Header，并贯穿日志、审计和错误响应。
- CI 重新生成 OpenAPI；生成结果与仓库文件不一致时失败。
- 不新增 MariaDB 和 Redis 之外的必需服务。
- 所有行为修改遵循 TDD：先看到聚焦测试正确失败，再写最小实现。
- 未经用户明确要求，不执行 `git commit`。

---

## File Structure

- `internal/shared/response/response_contract_test.go`：锁定响应外形、HTTP status、request ID、错误脱敏。
- `internal/shared/pagination/pagination.go`：统一分页、排序、过滤查询契约和默认值。
- `internal/shared/pagination/pagination_test.go`：覆盖默认页码、最大页大小、排序 allow-list 和 filter trim。
- `internal/modules/user/interfaces/handler.go`：使用统一查询契约，传递默认分页。
- `internal/modules/audit/interfaces/handler.go`：替换 `gin.Error{}`，使用稳定错误码。
- `internal/modules/user/interfaces/handler_test.go`：通过 `httptest` 验证参数校验、分页响应和 request ID。
- `internal/modules/audit/interfaces/handler_test.go`：验证 audit handler 不泄露内部错误并返回稳定 status。
- `work/smoke_api_contract.sh`：Compose 级 smoke，覆盖迁移、公开注册模式、登录、刷新、受保护 API、logout。
- `.github/workflows/ci.yml`：固定 golangci-lint 版本，增加 OpenAPI diff、race、build、Docker image 和 smoke 脚本检查。
- `README.md`：记录 API 契约、分页参数、OpenAPI 生成和 smoke 流程。

## Task 1: Response Contract and Handler Error Hygiene

**Files:**
- Create: `internal/shared/response/response_contract_test.go`
- Modify: `internal/modules/audit/interfaces/handler.go`
- Create: `internal/modules/audit/interfaces/handler_test.go`

**Interfaces:**
- Consumes: `response.OK`, `response.Fail`, `response.Page`
- Produces: stable `{code,message,data,request_id}` and paginated `{code,message,data,total,page,page_size,request_id}`

- [ ] **Step 1: Write failing response contract tests**

Add `internal/shared/response/response_contract_test.go`:

```go
package response

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	appErrs "jimu/internal/shared/errors"

	"github.com/gin-gonic/gin"
)

func TestOKIncludesStableEnvelopeAndRequestID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/", func(c *gin.Context) {
		c.Set("request_id", "rid-1")
		OK(c, gin.H{"name": "alice"})
	})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d", w.Code)
	}
	var body map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["code"].(float64) != 0 || body["message"] != "ok" || body["request_id"] != "rid-1" {
		t.Fatalf("body = %#v", body)
	}
	if _, ok := body["data"].(map[string]any); !ok {
		t.Fatalf("data missing: %#v", body)
	}
}

func TestPageIncludesStableEnvelopeAndPagination(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/", func(c *gin.Context) {
		c.Set("request_id", "rid-2")
		Page(c, []string{"a"}, 10, 2, 5)
	})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))
	var body map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["total"].(float64) != 10 || body["page"].(float64) != 2 || body["page_size"].(float64) != 5 || body["request_id"] != "rid-2" {
		t.Fatalf("body = %#v", body)
	}
}

func TestFailDoesNotLeakInfrastructureDetails(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/", func(c *gin.Context) {
		c.Set("request_id", "rid-3")
		Fail(c, appErrs.Wrap(appErrs.CodeInternalError, "sql: password=secret", assertErr("dsn /tmp/secret")))
	})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d", w.Code)
	}
	if got := w.Body.String(); got != `{"code":1005,"message":"internal error","request_id":"rid-3"}`+"\n" {
		t.Fatalf("body = %s", got)
	}
}

type assertErr string

func (e assertErr) Error() string { return string(e) }
```

- [ ] **Step 2: Run RED**

Run:

```bash
go test ./internal/shared/response -run 'Test(OKIncludesStableEnvelope|PageIncludesStableEnvelope|FailDoesNotLeak)' -v
```

Expected: pass if Task 5 already covers it. If it fails, fix only `internal/shared/response/response.go`.

- [ ] **Step 3: Write failing audit handler tests**

Add `internal/modules/audit/interfaces/handler_test.go`:

```go
package interfaces

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestAuditListInvalidQueryReturnsStableBadRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/audit", NewAuditHandler(nil).List)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/audit?page=0", nil))
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
	if strings.Contains(w.Body.String(), "gin.Error") {
		t.Fatalf("leaked gin error: %s", w.Body.String())
	}
}

func TestAuditGetInvalidIDReturnsStableBadRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/audit/:id", NewAuditHandler(nil).Get)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/audit/not-number", nil))
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}
```

- [ ] **Step 4: Run RED**

Run:

```bash
go test ./internal/modules/audit/interfaces -run TestAudit -v
```

Expected: fail because handler currently calls `response.Fail(c, gin.Error{})`, which maps to internal error.

- [ ] **Step 5: Implement minimal audit error hygiene**

Change `internal/modules/audit/interfaces/handler.go`:

```go
import (
	"strconv"

	"jimu/internal/modules/audit/application"
	"jimu/internal/shared/errors"
	"jimu/internal/shared/pagination"
	"jimu/internal/shared/response"

	"github.com/gin-gonic/gin"
)
```

For invalid query:

```go
if err := c.ShouldBindQuery(&p); err != nil {
	response.Fail(c, errors.New(errors.CodeInvalidParam, err.Error()))
	return
}
```

For service error:

```go
if err != nil {
	response.Fail(c, err)
	return
}
```

For invalid ID:

```go
if err != nil {
	response.Fail(c, errors.New(errors.CodeInvalidParam, "invalid id"))
	return
}
```

- [ ] **Step 6: Verify GREEN**

Run:

```bash
go test ./internal/shared/response ./internal/modules/audit/interfaces -v
```

Expected: all tests pass.

## Task 2: Unified Pagination, Sort, and Filter Contract

**Files:**
- Modify: `internal/shared/pagination/pagination.go`
- Create: `internal/shared/pagination/pagination_test.go`
- Modify: `internal/modules/user/interfaces/handler.go`
- Modify: `internal/modules/audit/interfaces/handler.go`

**Interfaces:**
- Produces: `func (p *Pagination) Normalize(allowedSorts ...string) error`
- Produces: fields `Page`, `PageSize`, `Sort`, `Order`, `Filter`
- Consumes: handlers call `p.Normalize("id", "username", "created_at")` before service calls

- [ ] **Step 1: Write failing pagination tests**

Add `internal/shared/pagination/pagination_test.go`:

```go
package pagination

import "testing"

func TestNormalizeDefaultsAndCapsPageSize(t *testing.T) {
	p := Pagination{}
	if err := p.Normalize("id", "created_at"); err != nil {
		t.Fatal(err)
	}
	if p.Page != 1 || p.PageSize != 20 || p.Order != "desc" || p.Sort != "id" {
		t.Fatalf("pagination = %#v", p)
	}

	p = Pagination{Page: 2, PageSize: 500, Sort: "created_at", Order: "ASC", Filter: " alice "}
	if err := p.Normalize("id", "created_at"); err != nil {
		t.Fatal(err)
	}
	if p.PageSize != 100 || p.Order != "asc" || p.Filter != "alice" {
		t.Fatalf("pagination = %#v", p)
	}
}

func TestNormalizeRejectsInvalidSortAndOrder(t *testing.T) {
	for _, p := range []Pagination{
		{Sort: "password"},
		{Order: "drop table"},
	} {
		if err := p.Normalize("id", "created_at"); err == nil {
			t.Fatalf("expected error for %#v", p)
		}
	}
}
```

- [ ] **Step 2: Run RED**

Run:

```bash
go test ./internal/shared/pagination -v
```

Expected: compile failure because `Normalize`, `Sort`, `Order`, and `Filter` do not exist.

- [ ] **Step 3: Implement minimal pagination contract**

Replace `internal/shared/pagination/pagination.go` with:

```go
package pagination

import (
	"fmt"
	"strings"
)

const (
	DefaultPage     = 1
	DefaultPageSize = 20
	MaxPageSize     = 100
	DefaultSort     = "id"
	DefaultOrder    = "desc"
)

type Pagination struct {
	Page     int    `form:"page" binding:"omitempty,min=1"`
	PageSize int    `form:"page_size" binding:"omitempty,min=1,max=100"`
	Sort     string `form:"sort"`
	Order    string `form:"order"`
	Filter   string `form:"filter"`
}

func (p *Pagination) Normalize(allowedSorts ...string) error {
	if p.Page == 0 {
		p.Page = DefaultPage
	}
	if p.PageSize == 0 {
		p.PageSize = DefaultPageSize
	}
	if p.PageSize > MaxPageSize {
		p.PageSize = MaxPageSize
	}
	p.Filter = strings.TrimSpace(p.Filter)
	p.Sort = strings.TrimSpace(p.Sort)
	if p.Sort == "" {
		p.Sort = DefaultSort
	}
	if len(allowedSorts) > 0 && !contains(allowedSorts, p.Sort) {
		return fmt.Errorf("invalid sort")
	}
	p.Order = strings.ToLower(strings.TrimSpace(p.Order))
	if p.Order == "" {
		p.Order = DefaultOrder
	}
	if p.Order != "asc" && p.Order != "desc" {
		return fmt.Errorf("invalid order")
	}
	return nil
}

func (p Pagination) GetOffset() int {
	return (p.Page - 1) * p.PageSize
}

func (p Pagination) GetLimit() int {
	return p.PageSize
}

func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
```

- [ ] **Step 4: Update handlers to normalize**

In `internal/modules/user/interfaces/handler.go`, after `ShouldBindQuery`:

```go
if err := p.Normalize("id", "username", "created_at"); err != nil {
	response.Fail(c, errors.New(errors.CodeInvalidParam, err.Error()))
	return
}
```

In `internal/modules/audit/interfaces/handler.go`, after `ShouldBindQuery`:

```go
if err := p.Normalize("id", "created_at"); err != nil {
	response.Fail(c, errors.New(errors.CodeInvalidParam, err.Error()))
	return
}
```

- [ ] **Step 5: Verify GREEN**

Run:

```bash
go test ./internal/shared/pagination ./internal/modules/user/interfaces ./internal/modules/audit/interfaces -v
```

Expected: tests pass and missing `page/page_size` no longer fails binding.

## Task 3: HTTP Contract Tests for User List and Auth Routes

**Files:**
- Create: `internal/modules/user/interfaces/handler_test.go`
- Modify: `internal/modules/auth/interfaces/router_test.go`

**Interfaces:**
- Consumes: `pagination.Pagination.Normalize`
- Produces: HTTP tests for request ID, status codes, default pagination and route exposure

- [ ] **Step 1: Write user list contract test**

Add `internal/modules/user/interfaces/handler_test.go`:

```go
package interfaces

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestUserListRejectsInvalidPaginationContract(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/users", NewUserHandler(nil).List)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/users?sort=password", nil))
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestUserListUsesDefaultPaginationBeforeService(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/users", func(c *gin.Context) {
		c.Set("request_id", "rid-list")
		NewUserHandler(nil).List(c)
	})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/users", nil))
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want nil service to fail after pagination, got %d", w.Code)
	}
	var body map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["request_id"] != "rid-list" {
		t.Fatalf("body = %#v", body)
	}
}
```

- [ ] **Step 2: Run RED**

Run:

```bash
go test ./internal/modules/user/interfaces -run TestUserList -v
```

Expected: fail before Task 2; pass after Task 2 except nil service may panic. If it panics, add a nil-service guard returning `CodeInternalError` instead of panic.

- [ ] **Step 3: Extend auth route contract tests**

In `internal/modules/auth/interfaces/router_test.go`, add:

```go
func TestRefreshRouteStaysPublic(t *testing.T) {
	r := testRouter(false)
	RegisterAuthRoutes(r.Group("/api/v1"), nil, auth.New(strings.Repeat("s", 32), "jimu", 30, 7), testAuthConfig(), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", strings.NewReader("{}")))
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}
```

- [ ] **Step 4: Verify GREEN**

Run:

```bash
go test ./internal/modules/user/interfaces ./internal/modules/auth/interfaces -v
```

Expected: tests pass.

## Task 4: Compose API Smoke Test

**Files:**
- Create: `work/smoke_api_contract.sh`
- Modify: `README.md`

**Interfaces:**
- Consumes: Docker Compose service names `server`, `mariadb`, `redis`
- Produces: executable smoke covering config, migration, health, public registration mode, login, refresh, protected API, logout

- [ ] **Step 1: Create smoke script**

Create `work/smoke_api_contract.sh`:

```bash
#!/usr/bin/env bash
set -euo pipefail

COMPOSE=${COMPOSE:-"docker compose"}
HTTP_PORT=${HTTP_PORT:-18081}
BASE_URL="http://127.0.0.1:${HTTP_PORT}/api/v1"
CREATED_ENV=0
export HTTP_PORT

cleanup() {
  $COMPOSE down
  if [ "$CREATED_ENV" = "1" ]; then
    rm -f .env
  fi
}
trap cleanup EXIT

if [ ! -f .env ]; then
  CREATED_ENV=1
  cat > .env <<'ENV'
JIMU_ENV=prod
DB_ROOT_PASSWORD=jimu-root-contract
DB_USER=jimu
DB_PASSWORD=jimu-db-contract
DB_DATABASE=jimu
JIMU__DB__HOST=mariadb
JIMU__DB__PORT=3306
JIMU__DB__USER=jimu
JIMU__DB__PASSWORD=jimu-db-contract
JIMU__REDIS__ADDR=redis:6379
JIMU__AUTH__JWT_SECRET=01234567890123456789012345678901
JIMU__AUTH__PUBLIC_REGISTRATION=true
JIMU__AUTH__LOGIN_RATE_LIMIT=100
JIMU__AUTH__REGISTER_RATE_LIMIT=100
ENV
fi

$COMPOSE up -d --build
$COMPOSE exec -T server ./jimu migrate up

for _ in $(seq 1 60); do
  if curl -fsS "http://127.0.0.1:${HTTP_PORT}/api/v1/auth/register" -X POST \
    -H "Content-Type: application/json" \
    -d '{"username":"smoke_user","password":"secret123"}' >/tmp/jimu-register.json; then
    break
  fi
  sleep 2
done

LOGIN=$(curl -fsS "${BASE_URL}/auth/login" -X POST -H "Content-Type: application/json" -d '{"username":"smoke_user","password":"secret123"}')
ACCESS=$(printf "%s" "$LOGIN" | sed -n 's/.*"access_token":"\([^"]*\)".*/\1/p')
REFRESH=$(printf "%s" "$LOGIN" | sed -n 's/.*"refresh_token":"\([^"]*\)".*/\1/p')
test -n "$ACCESS"
test -n "$REFRESH"

curl -fsS "${BASE_URL}/auth/refresh" -X POST -H "Content-Type: application/json" -d "{\"refresh_token\":\"${REFRESH}\"}" >/tmp/jimu-refresh.json
curl -fsS "${BASE_URL}/users" -H "Authorization: Bearer ${ACCESS}" >/tmp/jimu-users.json
curl -fsS "${BASE_URL}/auth/logout" -X POST -H "Authorization: Bearer ${ACCESS}" >/tmp/jimu-logout.json

grep -q '"code":0' /tmp/jimu-refresh.json
grep -q '"code":0' /tmp/jimu-users.json
grep -q '"code":0' /tmp/jimu-logout.json
```

Run:

```bash
chmod +x work/smoke_api_contract.sh
bash -n work/smoke_api_contract.sh
```

- [ ] **Step 2: Run RED or identify blocker**

Run:

```bash
work/smoke_api_contract.sh
```

Expected: may fail if migrations/seed/auth protection are incomplete. Fix only in-scope contract or script issues. If Docker unavailable, report as unverified.

- [ ] **Step 3: Update README smoke section**

Add a section:

```markdown
## API 契约检查

```bash
work/test_runtime_security.sh
work/smoke_api_contract.sh
make swagger
make swagger
git diff -- docs/openapi
```
```

- [ ] **Step 4: Verify GREEN**

Run:

```bash
bash -n work/smoke_api_contract.sh
work/smoke_api_contract.sh
```

Expected: script exits 0 and stops Compose without removing named data volumes.

## Task 5: CI Quality Gates and OpenAPI Diff

**Files:**
- Modify: `.github/workflows/ci.yml`
- Modify: `Makefile`

**Interfaces:**
- Consumes: `make swagger`
- Produces: CI gates for gofmt, vet, fixed golangci-lint, tests, race, OpenAPI diff, build, Docker image, smoke script syntax

- [ ] **Step 1: Update CI workflow**

Modify `.github/workflows/ci.yml`:

```yaml
env:
  GO_VERSION: "1.26.5"
  GOLANGCI_LINT_VERSION: "v2.7.2"
```

In `lint` job, replace `version: latest`:

```yaml
      - name: golangci-lint
        uses: golangci/golangci-lint-action@v8
        with:
          version: ${{ env.GOLANGCI_LINT_VERSION }}
          args: --timeout=5m
```

In `test` job, use:

```yaml
      - name: Run tests
        run: go test ./...

      - name: Run race tests
        run: go test -race ./internal/app ./internal/config ./internal/platform/auth ./internal/platform/http/... ./internal/modules/auth/... ./internal/modules/audit/...
```

Add OpenAPI check:

```yaml
      - name: Check OpenAPI is current
        run: |
          make swagger
          git diff --exit-code docs/openapi
```

Add script syntax check:

```yaml
      - name: Check smoke scripts
        run: |
          bash -n work/test_runtime_security.sh
          bash -n work/smoke_api_contract.sh
```

- [ ] **Step 2: Run workflow-adjacent checks locally**

Run:

```bash
make swagger
make swagger
git diff -- docs/openapi
bash -n work/test_runtime_security.sh
bash -n work/smoke_api_contract.sh
go test ./...
go test -race ./internal/app ./internal/config ./internal/platform/auth ./internal/platform/http/... ./internal/modules/auth/... ./internal/modules/audit/...
go vet ./...
go build ./cmd/server ./cmd/cli
docker build -t jimu:api-contract-test .
```

Expected: all commands exit 0. If `make swagger` twice changes files beyond already intended OpenAPI output, keep generated OpenAPI changes and re-run until stable.

## Task 6: Phase Verification and Scope Review

**Files:**
- Modify: `docs/superpowers/plans/2026-07-29-jimu-engineering-api-contract.md`

**Interfaces:**
- Produces: verified phase completion state

- [ ] **Step 1: Run full verification**

Run:

```bash
gofmt -w cmd internal
git diff --check
go test ./...
go test -race ./...
go vet ./...
go build ./cmd/server ./cmd/cli
docker compose config --quiet
docker build -t jimu:api-contract-test .
work/test_runtime_security.sh
work/smoke_api_contract.sh
```

Expected: all commands exit 0. If Docker unavailable, report Docker checks as unverified, not passed.

- [ ] **Step 2: Review workspace scope**

Run:

```bash
git status --short --untracked-files=all
git diff --stat
```

Expected: all changed files trace to runtime/security baseline or engineering/API contract phase. No generated logs remain in `git status`.

- [ ] **Step 3: Mark plan checkboxes**

After verification succeeds, change every `- [ ]` in this plan to `- [x]`.

Run:

```bash
rg -n "^- \\[ \\]" docs/superpowers/plans/2026-07-29-jimu-engineering-api-contract.md
```

Expected: no output.
