# 能力可插拔 P1.6：auth 六能力 实现计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 把 `internal/capabilities/auth`（当前 2900+ 行混装包）按 §5.1 拆成 6 个 catalog 能力：`auth`（会话与凭证本体）、`mfa`（TOTP 二次验证 + 可信设备）、`passkey`（WebAuthn 通行密钥）、`breach`（泄露口令检查）、`captcha`（验证码）、tenancy（开通式注册迁入现有 `tenant` 能力）。对外 HTTP 路由/响应/schema/配置键不变。

**Architecture:** 五条 contract 端口消除拆分后的跨能力耦合：`MFAVerifier`（auth←mfa）、`LoginFinalizer`（passkey←auth）、`TenantProvisioner`（auth←tenant）、`BreachChecker`（auth←breach）、`CaptchaVerifier`（auth←captcha）。TOTP 从 `users.totp_*` 列迁到 mfa 自有 `user_mfa` 表（迁移 016，幂等+可回滚，密文原样搬迁）；`shared/totp` 迁入 mfa。`New(...deps ...interface{})` 显式 Deps 结构体替代 auth 现有的无类型变参 + type-assert。

**Tech Stack:** Go 1.26 · goose v3.27.3（Provider + WithTableName）· gorm · sqlmock · testify · miniredis · go-webauthn

**Spec:** `docs/design/2026-09-18-capability-plugins-design.md` §5.1（六能力去向表）、§5.4（tenancy=开通式注册；本计划裁定 #2 保留能力名 `tenant`）、§6.1（Descriptor）、§7（迁移归属）、§10（P1.6 行）；`docs/plans/2026-09-20-capability-seed-migrations.md` 裁定 #2（表归属：004 留 user 待 P1.6、013 归 mfa、011/012 留 auth、015 归 passkey）

## Global Constraints

- **禁止自动提交**：各任务 commit 步骤仅在用户明确说"提交"后执行（AGENTS.md 最高优先级）
- **零语义变化**：HTTP 路由、响应 JSON（含字段序、omitempty）、配置键、错误码、`capabilities.enabled` 语义全部不变；只搬代码不改逻辑
- **不新增第三方依赖**；**能力间只经 contract 端口调用**，禁止 import 其他能力的内部包（`domain/application/infrastructure/interfaces`）
- 分支：从 `release/v0.3.0` 切 `feature/auth-six-capabilities`，PR 目标 `release/v0.3.0`
- 提交信息 Conventional Commits 轻量格式，全英文小写祈使句
- 全绿门禁：`gofmt -l .`（无输出）、`go build ./...`、`go vet ./...`、`golangci-lint run ./...`、`make check-log-usage`、`go test ./... -count=1`、`make bench-ci`、`make release-check COMPOSE_ENV=.env.example`
- 迁移文件沿用原全局编号：新迁移用 016（15 已被 webauthn 占用）

## 设计裁定（执行前必读）

1. **六能力去向**（§5.1 落实）：

   | 现文件/符号 | 去向 |
   |---|---|
   | `service.go` Login/LoginWithTOTP/finishLogin/Refresh/Logout/LogoutAll/issueTokenPair/recordFailure/Register/checkRegistrationAvailable/ForgotPassword/ResetPassword/generateResetCode/checkPasswordReuse/recordPasswordHistory | `auth` |
   | `application/login_history.go` + `domain/login_history.go` + 迁移 011 | `auth`（登录审计属登录用例） |
   | `domain/password_history.go` + 迁移 012 | `auth`（密码策略属改密） |
   | `interfaces/handler.go` login/register/forgot/reset/refresh/logout 段 + `allow`/`writeAuthRateLimitHeaders` + `interfaces/router.go` 对应路由 | `auth` |
   | `interfaces/protected.go` + `ProtectedHTTPMiddleware` | `auth`（唯一受保护中间件提供者不变） |
   | `SetupTOTP/EnableTOTP/DisableTOTP` + `shared/totp` + TOTP 相关 handler 段 + 用户 TOTP 读写 + 迁移 016 | `mfa` |
   | `trusted_device.go` + `domain/trusted_device.go` + infra + 迁移 013 + 登录"跳过 TOTP"判定 | `mfa`（"跳过 MFA"的能力） |
   | `webauthn.go` + `domain/webauthn.go` + infra + `interfaces/webauthn_handler.go` + 迁移 015 | `passkey` |
   | `checkBreachedPassword`（对外服务依赖） | `breach`（已独立目录，仅 catalogize） |
   | `verifyCaptcha` + `RegisterCaptchaRoute` | `captcha`（已独立目录，仅 catalogize + 自挂路由） |
   | `provisioning.go`（GormTenantProvisioner + 模板角色 + generateTenantCode） | `tenant`（开通式注册 = 租户开户用例；能力名保留 tenant） |

2. **能力命名**：保留现有 `tenant` 名（用户裁定），不引入 `tenancy`。增加 `mfa`、`passkey`、`breach` 三个 catalog 能力；`captcha` 目录已存在，本轮 catalogize。

3. **catalog 顺序与 Requires**（依赖在前）：

   ```text
   user, role, permission, tenant, auth, mfa, passkey, audit, admin,
   oauth, apikey, queue, outbox, dataops, search, breach
   ```

   - `auth` Requires `["user","role","tenant"]`（不变）
   - `mfa` Requires `["user"]`（016 读 users 表）
   - `passkey` Requires `["user","auth"]`（FinalizeLogin 由 auth 提供）
   - `breach` 无 Requires、无迁移、无 Module 实例（基础设施能力，仅参与 catalog 清单；不走 `wiredCapabilities`）
   - `captcha` 无 Requires、无迁移；本轮新增 Module 实例并进 `wiredCapabilities`（自挂 `/api/v1/captcha` 公开路由）

4. **能力无 Module 实例的边界**：`breach` 仍以 `breach.Checker` 库形态存在，由 container 构造后经 `contract.BreachChecker` 端口注入 auth；不进 `wiredCapabilities`，其 Descriptor 只进 catalog（维持"清单尾部基础设施能力只带迁移/声明、无实例"的既有约定，`mfa/passkey/captcha` 有实例）。

5. **TOTP 数据迁移（016）**：TOTP 从 `users.totp_secret/totp_enabled`（AES-GCM 密文）迁到 mfa 自有 `user_mfa` 表。两列均为 `encryption:"true"`，落库格式一致，迁移把密文**原样搬迁**（`INSERT ... SELECT`），不重新加解密；Down 反向搬回并删表。user 的 004 迁移**保留不动**（新库仍先建列，016 随后搬走再删列）；存量库 adopt 已登记 004，016 是新迁移、adopt 不登记，正常 `Up` 执行搬迁。`userdomain.User` 删除 `TOTPSecret/TOTPEnabled` 字段，`UserRepository.UpdateTOTP` 删除。

6. **trusted_devices 归属 mfa 但表不动**：013 迁移文件从 auth/migrations 移到 mfa/migrations，文件名沿用 `013_trusted_devices.sql`。表 `trusted_devices` 在 mfa 能力迁移里；TOTP 的"跳过"判定（`isTrustedDevice`）与可信设备 CRUD 全入 mfa 的 application，登录时 auth 经 `MFAVerifier` 端口询问"是否启用 MFA"、由 mfa 内部自主判断是否设备可信跳过。

7. **contract 端口五件套**（新增于 `internal/contract/ports.go`）：

   ```go
   // TokenPair 无密码登录（passkey）完成后的令牌视图（不含 device_token）。
   type TokenPair struct {
       AccessToken  string `json:"access_token"`
       RefreshToken string `json:"refresh_token"`
       ExpiresIn    int    `json:"expires_in"`
   }

   // LoginFinalizer auth 提供：为已验证用户签发令牌、建会话、记登录历史、发事件。
   type LoginFinalizer interface {
       FinalizeLogin(ctx context.Context, userID uint64) (*TokenPair, error)
   }

   // MFAVerifier mfa 提供：登录时判定是否启用二次验证并校验 TOTP 码。
   // 错误语义由实现返回 shared/errors 的 CodeMFARequired / CodeInvalidMFA。
   type MFAVerifier interface {
       Enabled(ctx context.Context, userID uint64) (bool, error)
       VerifyTOTP(ctx context.Context, userID uint64, code string) error
   }

   // TenantProvisioner tenant 提供：开通式注册（注册 = 开通新租户）。nil = 未启用。
   type TenantProvisioner interface {
       Provision(ctx context.Context, req ProvisionRequest) (*ProvisionResult, error)
   }

   // ProvisionRequest / ProvisionResult 端口自有视图（不含 gorm 标签）。
   type ProvisionRequest struct {
       Username, PasswordHash, Email, Phone, TenantName, TenantCode string
   }
   type ProvisionResult struct {
       Tenant ProvisionedTenant `json:"tenant"`
       User   ProvisionedUser   `json:"user"`
   }
   type ProvisionedTenant struct {
       ID   uint64 `json:"id"`
       Code string `json:"code"`
       Name string `json:"name"`
   }
   type ProvisionedUser struct {
       ID       uint64 `json:"id"`
       Username string `json:"username"`
       Email    string `json:"email"`
       Phone    string `json:"phone"`
       Status   int8   `json:"status"`
       TenantID uint64 `json:"tenant_id"`
   }

   // BreachChecker breach 提供：泄露口令检查。nil = 未启用。
   type BreachChecker interface {
       IsBreached(ctx context.Context, password string) (bool, error)
   }

   // CaptchaVerifier captcha 提供：登录/注册验证码校验。
   type CaptchaVerifier interface {
       Enabled() bool
       Verify(ctx context.Context, id, code string) error
   }

   // ClientInfo 登录请求客户端信息（IP/UA），passkey 与 auth 共用。
   type ClientInfo struct {
       IP        string
       UserAgent string
   }
   func WithClientInfo(ctx context.Context, ip, userAgent string) context.Context
   ```

   `ClientInfo`/`WithClientInfo` 实现放在 `internal/contract/client.go`（纯机制 context 助手，能力共享合法）。

---

### Task 1: contract 端口五件套

**Files:**
- Modify: `internal/contract/ports.go`（追加 MFAVerifier/LoginFinalizer/TenantProvisioner/TokenPair/BreachChecker/CaptchaVerifier 与请求/响应视图）
- Create: `internal/contract/client.go`（ClientInfo + WithClientInfo）
- Create: `internal/contract/ports_test.go`

**Interfaces:**
- Consumes: 无
- Produces: `contract.MFAVerifier` / `contract.LoginFinalizer` / `contract.TenantProvisioner` / `contract.TokenPair` / `contract.ProvisionRequest` / `contract.ProvisionResult` / `contract.BreachChecker` / `contract.CaptchaVerifier` / `contract.ClientInfo` / `contract.WithClientInfo`

- [ ] **Step 1: 写 contract/client.go 失败测试**

```go
// internal/contract/ports_test.go
package contract

import (
    "context"
    "testing"
)

func TestWithClientInfoPreservesExistingValues(t *testing.T) {
    ctx := WithClientInfo(context.Background(), "1.2.3.4", "ua")
    ctx = WithClientInfo(ctx, "5.6.7.8", "new-ua") // IP 覆盖、UA 覆盖
    // WithClientInfo 只写 IP/UA；此处验证通过 (见 Step 3 clientInfo helper)
    _ = ctx
}
```

> 实际断言点：`clientInfoFrom` 是包内私有函数，测试放在 `package contract` 内可直测。以上仅示意；Step 3 落地时用真实断言（`info := clientInfoFrom(ctx); info.IP == "5.6.7.8"; info.UserAgent == "new-ua"`）。

- [ ] **Step 2: 跑测试确认失败**

`go test ./internal/contract/ -run TestWithClientInfo -count=1` → FAIL（`clientInfoFrom` 未定义）

- [ ] **Step 3: 落地端口与 client 助手**

`internal/contract/client.go`：

```go
package contract

import "context"

// ClientInfo 登录请求客户端信息（IP/UA），供 auth/passkey 的登录历史使用。
type ClientInfo struct {
    IP        string
    UserAgent string
}

type clientInfoKey struct{}

// WithClientInfo 把客户端信息写入上下文（保留已设置的其他字段；未涉及字段不动）。
func WithClientInfo(ctx context.Context, ip, userAgent string) context.Context {
    info, _ := ctx.Value(clientInfoKey{}).(ClientInfo)
    info.IP = ip
    info.UserAgent = userAgent
    return context.WithValue(ctx, clientInfoKey{}, info)
}

func clientInfoFrom(ctx context.Context) ClientInfo {
    if v, ok := ctx.Value(clientInfoKey{}).(ClientInfo); ok {
        return v
    }
    return ClientInfo{}
}
```

在 `internal/contract/ports.go` 追加设计裁定 #7 的六个接口 + 三个视图结构 + `TokenPair`（直接按裁定 #7 的完整代码写，不占位）。

- [ ] **Step 4: 跑测试确认通过**

`go test ./internal/contract/ -count=1` → PASS

- [ ] **Step 5: 提交**（用户授权后执行）

`git add -A && git commit -m "feat(contract): add mfa passkey provisioning breach captcha ports"`

---

### Task 2: mfa 能力（TOTP + 可信设备 + 迁移 016）

**Files:**
- Create: `internal/capabilities/mfa/{module.go,domain/totp.go,domain/trusted_device.go,application/service.go,application/trusted_device.go,infrastructure/mysql_mfa.go,migrations/mysql/013_trusted_devices.sql,migrations/mysql/016_user_mfa.sql,migrations/postgres/013_trusted_devices.sql,migrations/postgres/016_user_mfa.sql,interfaces/handler.go,interfaces/router.go}`（含各对应 `*_test.go`，从 auth 或 shared/totp 迁入后改 package 名与 import 路径）
- Move: `internal/shared/totp/{totp.go,totp_test.go}` → `internal/capabilities/mfa/totp/`
- Move: `internal/capabilities/auth/migrations/{mysql,postgres}/013_trusted_devices.sql` → `internal/capabilities/mfa/migrations/{mysql,postgres}/`
- Move: `internal/capabilities/auth/application/trusted_device.go(+test)`、`internal/capabilities/auth/domain/trusted_device.go`、`internal/capabilities/auth/infrastructure/mysql_trusted_device.go(+test)` → mfa 对应层
- Modify: `internal/capabilities/user/domain/{user.go,repository.go}`、`internal/capabilities/user/infrastructure/mysql_repository.go`（删 TOTP 字段与 `UpdateTOTP`）

**Interfaces:**
- Consumes: `contract.MFAVerifier`（实现之）、`user` 迁移 004（users 表）
- Produces: `mfa.Module` / `mfa.New(db, deps...)`（`deps` 里接受 `*encryption.Cipher` 用于 user_mfa 密钥加解密）；`mfa.Descriptor{Name:"mfa",Requires:["user"],Mount:MountSelfManaged,Permissions:...}`

- [ ] **Step 1: 建 016 迁移（含 Down）**

`mysql/016_user_mfa.sql`：

```sql
-- +goose Up
CREATE TABLE user_mfa (
    id BIGINT UNSIGNED NOT NULL,
    tenant_id BIGINT UNSIGNED NOT NULL DEFAULT 0,
    user_id BIGINT UNSIGNED NOT NULL DEFAULT 0,
    totp_secret TEXT NULL,
    totp_enabled TINYINT(1) NOT NULL DEFAULT 0,
    created_at DATETIME(3) NULL,
    updated_at DATETIME(3) NULL,
    PRIMARY KEY (id),
    KEY idx_user_mfa_user_id (user_id),
    KEY idx_user_mfa_tenant_id (tenant_id)
);
-- 密文原样搬迁（两列均 AES-GCM 密文，格式一致，不重新加解密）
INSERT INTO user_mfa (tenant_id, user_id, totp_secret, totp_enabled)
    SELECT tenant_id, id, totp_secret, totp_enabled FROM users WHERE totp_enabled = 1;
ALTER TABLE users
    DROP COLUMN totp_enabled,
    DROP COLUMN totp_secret;

-- +goose Down
ALTER TABLE users
    ADD COLUMN totp_secret TEXT NULL AFTER phone_hash,
    ADD COLUMN totp_enabled TINYINT(1) NOT NULL DEFAULT 0 AFTER totp_secret;
UPDATE users u JOIN user_mfa m ON m.user_id = u.id
    SET u.totp_secret = m.totp_secret, u.totp_enabled = m.totp_enabled
    WHERE m.totp_enabled = 1;
DROP TABLE user_mfa;
```

> postgres 版本用 `TEXT`/`BOOLEAN`/`TIMESTAMP`，`AFTER phone_hash` 去掉（PG 不支持列定位），`UPDATE ... FROM user_mfa m WHERE m.user_id = users.id`。

- [ ] **Step 2: user 删 TOTP 字段与仓库方法**

删除 `userdomain.User.TOTPSecret/TOTPEnabled`、`UserRepository.UpdateTOTP`、`mysql_repository.go` 的 `UpdateTOTP` 实现；`RESOLVE 编译错误`：`auth/application/service.go` 与 `service_test.go` 对 TOTP 的调用暂注释（Task 3 统一接管）——但本任务结束前必须全绿，故顺序改为：**先搬 mfa 落地、再一步删 user 字段并同步 auth 的 LoginWithTOTP/Setup/Enable/Disable 与测试**（见 Step 3 收口）。

- [ ] **Step 3: 搬 TOTP 与可信设备到 mfa**

- `shared/totp` 整体 `git mv` 到 `internal/capabilities/mfa/totp`，改 package 注释（仍 `package totp`）。
- 建 `mfa/domain/totp.go`：

```go
package domain

import "context"

// UserMFA 用户 TOTP 二次验证状态。密钥 AES-GCM 落库，读取时解密回明文。
type UserMFA struct {
    ID          uint64 `gorm:"primaryKey" json:"id"`
    TenantID    uint64 `gorm:"column:tenant_id;default:0;index" json:"tenant_id"`
    UserID      uint64 `gorm:"column:user_id;default:0;index" json:"user_id"`
    TOTPSecret  string `gorm:"column:totp_secret;type:text" encryption:"true" json:"-"`
    TOTPEnabled bool   `gorm:"column:totp_enabled;default:false" json:"totp_enabled"`
}
func (UserMFA) TableName() string { return "user_mfa" }

type MFARepository interface {
    FindByUser(ctx context.Context, userID uint64) (*UserMFA, error) // 不存在返回 gorm.ErrRecordNotFound
    UpsertSecret(ctx context.Context, userID uint64, secret string, enabled bool) error
}
```

- `mfa/application/service.go` 提供 `SetupTOTP/EnableTOTP/DisableTOTP`（逻辑逐行照搬 auth `service.go` 原实现，只是把 `userRepo.UpdateTOTP` 换成 `mfaRepo.UpsertSecret`，`userRepo.FindByID` 换成从 mfa 自持记录 + 需要 username 时仍用 `UserinfoSource` 端口——若不便，先按原实现用 `contract.UserinfoSource` 查 username）、`Enabled`/`VerifyTOTP`（实现 `contract.MFAVerifier`）。
- `mfa/application/trusted_device.go` 逐行照搬（`isTrustedDevice/issueTrustedDevice/List/Revoke/RevokeAll/Cleanup`），并把 login 里"跳过 TOTP"的判定移入 `VerifyTOTP` 的前置：**auth 只调 `Enabled` + `VerifyTOTP`；mfa 内部先查可信设备令牌即可跳过**。`hashDeviceToken`、`WithLoginDevice` 随迁。
- `mfa/module.go`：

```go
package mfa

import (
    "embed"
    "jimu/internal/capabilities/mfa/application"
    "jimu/internal/capabilities/mfa/interfaces"
    "jimu/internal/contract"
    "gorm.io/gorm"
)

type Module struct { svc *application.MFAService }
func New(db *gorm.DB, _ ...interface{}) *Module {
    return &Module{svc: application.NewMFAService(db)}
}
func (m *Module) Name() string { return "mfa" }

//go:embed migrations
var migrationsFS embed.FS

var Descriptor = contract.Descriptor{
    Name:       "mfa",
    Requires:   []string{"user"},
    Migrations: migrationsFS,
    Mount:      contract.MountSelfManaged,
    Permissions: []contract.Permission{
        {Name: "MFA 配置", Resource: "/api/v1/auth/mfa/setup", Action: "POST"},
        {Name: "MFA 启用", Resource: "/api/v1/auth/mfa/enable", Action: "POST"},
        {Name: "MFA 关闭", Resource: "/api/v1/auth/mfa/disable", Action: "POST"},
        {Name: "可信设备列表", Resource: "/api/v1/auth/devices", Action: "GET"},
        {Name: "可信设备吊销", Resource: "/api/v1/auth/devices", Action: "DELETE"},
        {Name: "可信设备详情", Resource: "/api/v1/auth/devices/*", Action: "DELETE"},
    },
}
func (m *Module) Descriptor() contract.Descriptor { return Descriptor }
func (m *Module) RegisterHTTP(r contract.Router) { interfaces.RegisterMFARoutes(r.Group("/api/v1"), m.svc) }
func (m *Module) RegisterJobs(j contract.JobRegistry) {}
func (m *Module) RegisterEvents(e contract.EventBus) {}
```

- [ ] **Step 4: auth 收口经 MFAVerifier**

`auth/application/service.go`：

```go
// LoginWithTOTP 改造：user.TOTPEnabled 判定替换为端口调用
mfaEnabled, err := s.mfa.Enabled(ctx, user.ID) // s.mfa contract.MFAVerifier，nil=无二次验证
if err != nil { ... } else if mfaEnabled {
    if err := s.mfa.VerifyTOTP(ctx, user.ID, totpCode); err != nil {
        s.recordFailure(ctx, normalized)
        s.recordLoginHistory(ctx, user.ID, user.TenantID, normalized, authdomain.LoginStatusFailed, "totp verification failed")
        return nil, err
    }
}
```

删除 auth 内 `SetupTOTP/EnableTOTP/DisableTOTP` 与 TOTP 相关 handler/routes（`/auth/mfa/*`、`/auth/devices*` 从 auth `interfaces/router.go` 移除，)也就从 auth handler 删除对应 handler 段）。删 `shared/totp` import。`NewAuthService` 增加 `mfa contract.MFAVerifier` 字段 + `case contract.MFAVerifier:` 分支。

- [ ] **Step 5: 同步测试归位**

- 迁 `service_test.go` 的 `TestLoginRequiresTOTPWhenEnabled/TestDisableTOTP` 到 `mfa/application`，把 `fakeUserRepo.UpdateTOTP` 换成 mfa 假仓储；`totpCurrentCode/totpTestCode` 改用 `mfa/totp`。
- `trusted_device_test.go` 整套搬到 mfa（改 package + import `trusteddevice` 域类型路径）。登录侧"跳过 TOTP"的用例改为直接测 `mfa.VerifyTOTP` 对可信设备令牌的短路。
- auth `handler_test.go`/`router_test.go` 删 TOTP/devices/webauthn 断言（webauthn 归 passkey，留 Task 3）。

- [ ] **Step 6: 验证全绿**

```bash
gofmt -l . && go build ./... && go vet ./...
go test ./internal/capabilities/mfa/... ./internal/capabilities/auth/... ./internal/capabilities/user/... -count=1
```

Expected: 全绿；auth 包不再 import `shared/totp`、不再含 `SetupTOTP`；user 域不再有 `TOTPSecret/UpdateTOTP`；`go test ./... -count=1` 仍需在 Task 6 收口（catalog/main 尚未接线，此时不跑全量，避免断档）。

- [ ] **Step 7: 提交**（用户授权后执行）

`git add -A && git commit -m "feat(mfa): extract totp and trusted devices into the mfa capability"`

---

### Task 3: passkey 能力（WebAuthn + 迁移 015）

**Files:**
- Move: `internal/capabilities/auth/application/webauthn.go(+test)`、`internal/capabilities/auth/domain/webauthn.go`、`internal/capabilities/auth/infrastructure/mysql_webauthn.go(+test)`、`internal/capabilities/auth/interfaces/webauthn_handler.go(+test)` → `internal/capabilities/passkey/{application,domain,infrastructure,interfaces}/`
- Move: `internal/capabilities/auth/migrations/{mysql,postgres}/015_webauthn_credentials.sql` → `internal/capabilities/passkey/migrations/`
- Create: `internal/capabilities/passkey/module.go`
- Modify: `internal/capabilities/auth/{module.go,application/service.go}`（删 webauthn 字段与依赖；改 `LoginFinalizer` 提供方）

**Interfaces:**
- Consumes: `contract.LoginFinalizer`、`contract.TokenPair`、`contract.WithClientInfo`、`userdomain.UserRepository`（经 `contract.UserinfoSource` 端口，避免 passkey 直 import user/domain）、`kernel/auth` JWT/limiter 机制
- Produces: `passkey.Module` / `passkey.New(db, rdb, cfg, finalizer, userinfo, ...)` / `passkey.Descriptor{Name:"passkey",Requires:["user","auth"],Mount:MountSelfManaged,Permissions:...}`

- [ ] **Step 1: 定义 auth 的 LoginFinalizer 实现（先写失败测试）**

在 `auth/application` 增加：

```go
// FinalizeLogin 实现 contract.LoginFinalizer：签发令牌 + 建会话 + 记历史 + 发事件。
func (s *AuthService) FinalizeLogin(ctx context.Context, userID uint64) (*contract.TokenPair, error) {
    user, err := s.userRepo.FindByID(ctx, userID)
    if err != nil || user.Status != 1 {
        return nil, invalidCredentials()
    }
    pair, err := s.finishLogin(ctx, user)
    if err != nil {
        return nil, err
    }
    return &contract.TokenPair{AccessToken: pair.AccessToken, RefreshToken: pair.RefreshToken, ExpiresIn: pair.ExpiresIn}, nil
}
```

> auth 的 `finishLogin` 已含 token/session/outbox/登录历史，直接复用；`DeviceToken` 被丢弃（passkey 不签发可信设备）。

- [ ] **Step 2: 搬 webauthn 到 passkey（up）**

包内所有 `authdomain` 改 `passkeydomain`；`s.userRepo` 改为 `s.users contract.UserinfoSource`；`s.lockout` 改为 `s.finalizer`（`CheckLocked/RecordFailure/RecordLoginHistory` 通过扩展后的 `LoginFinalizer` 端口拿，见 Step 3）。`finishLogin` 调用改为 `s.finalizer.FinalizeLogin`。

> 若"锁定/记失败/记历史"三件事也要端口化，`LoginFinalizer` 扩展为 `CheckLocked(ctx,username)(bool,time.Duration,error)` + `RecordFailure(ctx,username)` + `RecordLoginHistory(ctx,userID,tenantID,username,status,reason)`（auth 侧从 `recordFailure/recordLoginHistory/lockout` 直接转发）。保持端口最小：三者都是 auth 的既有公开行为，直接扩进 `LoginFinalizer`（改名 `LoginPort` 亦可，但为省改动保留 `LoginFinalizer` 名并追加方法）。Step 1 的实现同步扩。

- [ ] **Step 3: 扩展 contract.LoginFinalizer（追加锁定/失败/历史）**

`internal/contract/ports.go` 里 `LoginFinalizer` 追加：

```go
type LoginFinalizer interface {
    FinalizeLogin(ctx context.Context, userID uint64) (*TokenPair, error)
    CheckLocked(ctx context.Context, username string) (bool, time.Duration, error)
    RecordFailure(ctx context.Context, username string)
    RecordLoginHistory(ctx context.Context, userID, tenantID uint64, username, status, reason string)
}
```

auth 侧新增同名方法分别转发到既有 `lockout.CheckLocked`、`recordFailure`、`recordLoginHistory`。

- [ ] **Step 4: passkey 的 module + Descriptor**

```go
var Descriptor = contract.Descriptor{
    Name:       "passkey",
    Requires:   []string{"user", "auth"},
    Migrations: migrationsFS,
    Mount:      contract.MountSelfManaged,
    Permissions: []contract.Permission{
        {Name: "通行密钥登录开始", Resource: "/api/v1/auth/webauthn/login/begin", Action: "POST"},
        {Name: "通行密钥登录完成", Resource: "/api/v1/auth/webauthn/login/finish", Action: "POST"},
        {Name: "通行密钥注册开始", Resource: "/api/v1/auth/webauthn/register/begin", Action: "POST"},
        {Name: "通行密钥注册完成", Resource: "/api/v1/auth/webauthn/register/finish", Action: "POST"},
        {Name: "通行密钥列表", Resource: "/api/v1/auth/webauthn/credentials", Action: "GET"},
        {Name: "通行密钥改名", Resource: "/api/v1/auth/webauthn/credentials/*", Action: "PUT"},
        {Name: "通行密钥删除", Resource: "/api/v1/auth/webauthn/credentials/*", Action: "DELETE"},
    },
}
```

`RegisterHTTP` 只用中间件链注册 webauthn 路由（`r.Group("/api/v1/auth/webauthn")`），`MountSelfManaged`。

- [ ] **Step 5: auth 删除 webauthn 全部残留**

`auth/module.go` 删 `webauthnRepo/webauthn handle/webAuthnSessionTTL`；`service.go` 删 `webauthn/webauthnCreds/webauthnSessionTTL` 字段与 `case *webauthn.WebAuthn`/`WebAuthnCredentialRepository`/`webAuthnSessionTTL` 分支；`interfaces/router.go` 删 `/auth/webauthn/*` 路由。`interfaces/webauthn_handler.go` 整体移走。

- [ ] **Step 6: 验证**

```bash
gofmt -l . && go build ./...
go test ./internal/capabilities/passkey/... ./internal/capabilities/auth/... -count=1
```

Expected: 全绿；auth 不再 import go-webauthn；passkey 经端口调 auth，直 import user/domain 为零（经 `contract.UserinfoSource`）。

- [ ] **Step 7: 提交**（用户授权后执行）

`git add -A && git commit -m "feat(passkey): extract webauthn into the passkey capability"`

---

### Task 4: captcha / breach catalogize

**Files:**
- Modify: `internal/capabilities/captcha/`（补 `module.go`：`Module` 自挂 `/api/v1/captcha` 公开路由 + Descriptor；`captcha.go` 补充 `Enabled() bool`，`NewService` 增 enabled 入参或新增 `NewCaptchaVerifier` 包装实现 `contract.CaptchaVerifier`）
- Create: `internal/capabilities/breach/capability.go`（仅 `Descriptor{Name:"breach"}`，无 Module/Migrations，供 catalog 清单与 `contract.BreachChecker` 类型断言）
- Modify: `internal/capabilities/catalog/catalog.go`（加 `mfa/passkey/breach` import 与 entries；`captcha` 作为有 Module 实例的能力加入）

**Interfaces:**
- Consumes: `contract.CaptchaVerifier` / `contract.BreachChecker`（实现之）
- Produces: `captcha.Module` / `captcha.Descriptor`、`breach.Descriptor`

- [ ] **Step 1: captcha 模块 + 端口包装**

```go
// captcha/module.go
type Module struct{ svc *Service }
func New(rdb redistore.Client, ttl time.Duration, enabled bool) *Module {
    return &Module{svc: NewService(rdb, ttl, enabled)}
}
func (m *Module) Name() string { return "captcha" }
var Descriptor = contract.Descriptor{Name: "captcha", Mount: contract.MountPublic}
func (m *Module) Descriptor() contract.Descriptor { return Descriptor }
func (m *Module) RegisterHTTP(r contract.Router) {
    r.GET("/api/v1/captcha", NewCaptchaHandler(m.svc).Generate)
}
```

`Service` 增 `enabled bool` 字段与 `Enabled() bool`；`Verify` 在 `!enabled` 时返回 nil（auth 侧不再管 captchaConfig.Enabled）。

- [ ] **Step 2: breach Descriptor**

```go
// breach/capability.go
var Descriptor = contract.Descriptor{Name: "breach"}
func (b *hibpChecker) Descriptor() contract.Descriptor { return Descriptor }
```

- [ ] **Step 3: catalog entries**

```go
var entries = []contract.Descriptor{
    user.Descriptor, role.Descriptor, permission.Descriptor, tenantmodule.Descriptor,
    authmodule.Descriptor, mfamodule.Descriptor, passkeymodule.Descriptor, auditmodule.Descriptor,
    adminmodule.Descriptor, oauthmodule.Descriptor, apikey.Descriptor, queue.Descriptor,
    outbox.Descriptor, dataops.Descriptor, search.Descriptor,
    captcha.Descriptor, breach.Descriptor,
}
```

（breach 无 Module 实例，仍只进 catalog 不接线；captcha 本轮开始接线。）

- [ ] **Step 4: 验证**

```bash
gofmt -l . && go build ./internal/capabilities/captcha/... ./internal/capabilities/breach/... ./internal/capabilities/catalog/...
go test ./internal/capabilities/catalog/... -run TestDescriptorsAreWellFormed -count=1  # 会失败：夹具未同步
```

Expected: catalog 夹具更新前`TestResolveEmptyMeansAll`/`TestCatalogMigrationsShape`/`TestDescriptorPermissionsCoverBusinessRoutes` 失败——**这是预期**，Task 6 统一同步夹具。本任务只保证 `go build` 通过 + 新增包编译。

- [ ] **Step 5: 提交**（用户授权后执行）

`git add -A && git commit -m "feat(capabilities): catalogize captcha and breach"`

---

### Task 5: tenancy（provisioning 迁入 tenant）

**Files:**
- Move: `internal/capabilities/auth/application/provisioning.go(+test)` → `internal/capabilities/tenant/application/provisioning.go(+test)`
- Modify: `internal/capabilities/tenant/module.go`（增 `Provisioner()` 访问器暴露 `tenant/application.TenantProvisioner`；实现 `contract.TenantProvisioner`）
- Modify: `internal/contract/ports.go`（Task 1 已加视图，此处仅补 `tenant` 侧适配）
- Modify: `internal/capabilities/auth/application/service.go`（`RegisterProvisioned` 改用 `contract.TenantProvisioner` 端口 + `contract.ProvisionRequest/Result` 视图；删 `ProvisionParams/ProvisionResult/TenantProvisioner/GormTenantProvisioner`）

**Interfaces:**
- Consumes: `contract.TenantProvisioner`、`contract.ProvisionRequest/Result`、`contract.ProvisionedUser/Tenant`
- Produces: `tenant.Module.Provisioner() contract.TenantProvisioner`

- [ ] **Step 1: provisioning 搬入 tenant，改端口视图**

`tenant/application/provisioning.go`：`GormTenantProvisioner.Provision` 签名改成 `contract.ProvisionRequest → (*contract.ProvisionResult, error)`，内部逻辑照搬（事务建租户 + owner 用户 + 模板角色），结束后把 `tenantdomain.Tenant/userdomain.User` 映射到 `contract.ProvisionResult{ Tenant: contract.ProvisionedTenant{...}, User: contract.ProvisionedUser{...} }`。`isRecordNotFound/isDuplicateKeyErr/generateTenantCode/provisionTemplateRoles` 随迁。`provisioning_test.go` 断言同步改视图字段。

映射（HTTP JSON 字节等价）：
- `ProvisionedUser` 序列化字段与 `userdomain.User` 的 json tag 对齐：`id/username/email/phone/status/tenant_id`（`password` 已于 User 用 `json:"-"`，视图中不含）。
- `ProvisionedTenant` 与 `tenantdomain.Tenant` 对齐：`id/code/name`（status/plan_id/version/时间戳原本也因 `Tenant` json tag 输出——核对现有响应，`tenantdomain.Tenant.Status` 默认输出；若契约测试覆盖字段则补齐到视图，见下）。

> **动作**：先读 `internal/capabilities/tenant/domain/tenant.go` 的 json tag，把 `ProvisionedTenant` 字段补齐至与原 `Tenant` 序列化一致（至少 `id/code/name/status/plan_id`）。`ProvisionedUser` 同理补齐至与 `User` 一致（`id/username/email/phone/status/tenant_id`）。不新增字段到响应外。

- [ ] **Step 2: tenant.Module 暴露 Provisioner**

```go
// tenant/module.go 增：
func (m *Module) Provisioner() application.TenantProvisioner_contract { return ... }
```

> 直接让 `tenant/application` 导出一个 `NewProvisioner(db, cfg) *GormTenantProvisioner`，`Module` 持 `provisioner *GormTenantProvisioner`，暴露 `Provisioner() contract.TenantProvisioner`。`tenant.New` 签名在 main 装配时传入 `cfg.Provisioning`。

- [ ] **Step 3: auth 收口 RegisterProvisioned 走端口**

`auth/application/service.go`：

```go
func (s *AuthService) RegisterProvisioned(ctx context.Context, req RegisterTenantRequest) (*contract.ProvisionResult, error) {
    if s.provisioner == nil { return nil, errors.New(errors.CodeInvalidParam, "tenant provisioning is disabled") }
    if req.TenantName == "" { return nil, errors.New(errors.CodeInvalidParam, "tenant_name is required for provisioned registration") }
    username := normalizeUsername(req.Username)
    if err := s.checkRegistrationAvailable(ctx, username, req.Email); err != nil { return nil, err }
    if err := s.checkBreachedPassword(ctx, req.Password); err != nil { return nil, err }
    hashed, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
    if err != nil { return nil, errors.Wrap(errors.CodeInternalError, "failed to hash password", err) }
    // 删掉 tenant.NormalizeCode/ValidCode 与事务建租户逻辑——已归租户侧
    return s.provisioner.Provision(ctx, contract.ProvisionRequest{Username: username, PasswordHash: string(hashed), Email: req.Email, Phone: req.Phone, TenantName: req.TenantName, TenantCode: req.TenantCode})
}
```

`interfaces/handler.go` 的 Register 分支 `res.User/res.Tenant` 改为 `res.User/res.Tenant`（视图结构，json 等价）。删除 `service.go` 里 `RegisterTenantRequest` 的 `TenantCode` 归一化校验（归 tenant provisioner 内部）。

- [ ] **Step 4: 验证**

```bash
gofmt -l . && go build ./internal/capabilities/tenant/... ./internal/capabilities/auth/...
go test ./internal/capabilities/tenant/application/... ./internal/capabilities/auth/application/... -count=1
```

Expected: 全绿；auth 不再含 `GormTenantProvisioner` 与 `tenantdomain/roledomain` import；provisioning 测试全搬 tenant。

- [ ] **Step 5: 提交**（用户授权后执行）

`git add -A && git commit -m "refactor(tenant): move provisioned registration into the tenant capability"`

---

### Task 6: 装配收口 + catalog 夹具 + e2e + 文档

**Files:**
- Modify: `cmd/server/main.go`（wiredCapabilities 增 `mfa/passkey/captcha`；all map 加对应实例；auth 实例改新 Deps；tenant 实例传 provisioning；passkey 注入 finalizer/userinfo）
- Modify: `internal/app/container.go`（BreachChecker 字段类型改 `contract.BreachChecker`；captcha 构造入口改）
- Modify: `internal/e2e/api_contract_test.go`、`internal/capabilities/ws/ws_integration_test.go`（auth.New 签名同步；mfa/passkey/captcha 模块加入装配或按需）
- Modify: `internal/capabilities/catalog/catalog_test.go`（fixture 加 mfa/passkey/breach/captcha，13→17 条目）、`internal/capabilities/auth/module_test.go`（如引用旧 New）
- Modify: `README.md`（能力树、`shared/totp` 删除、TOTP/mfa 段落、目录树 capabilities 行）
- Modify: `AGENTS.md`（能力清单若提及 auth 范围，补 mfa/passkey）

**Interfaces:**
- Consumes: 上述全部端口
- Produces: 可启动的完整能力图

- [ ] **Step 1: main.go wiring**

```go
var wiredCapabilities = []string{
    "user", "role", "permission", "tenant", "auth", "mfa", "passkey",
    "audit", "admin", "oauth", "captcha",
}

all := map[string]contract.Module{
    "user":     user.New(container.DB, *cfg, container.Redis, container.Outbox),
    "role":     role.New(container.DB, tenantMod.Quota()),
    "permission": permission.New(container.DB),
    "tenant":   tenantMod, // 增 Provisioner 暴露
    "auth":     authmodule.New(container.DB, container.Redis, cfg.Auth, cfg.HTTP.Mode == config.HTTPModeRelease,
        container.Outbox, container.Notification, container.Cipher, tenantMod.Quota(),
        mfaVerifier, tenantMod.Provisioner(), container.BreachChecker, container.CaptchaVerifier),
    "mfa":      mfamodule.New(container.DB, container.Cipher),
    "passkey":  passkeymodule.New(container.DB, container.Redis, cfg.Auth, authFinalizer, userinfoSource),
    "audit":    auditmodule.New(container.DB, cfg.Audit, container.Logger),
    "admin":    adminmodule.New(...), // 不变
    "oauth":    oauthmodule.New(...), // 不变
    "captcha":  captchamodule.New(container.Redis, time.Duration(cfg.Captcha.TTLMin)*time.Minute, cfg.Captcha.Enabled),
}
```

> auth 的 `mfaVerifier` 来自 `mfaModule` 实例的 `Verifier()`；`authFinalizer`/`userinfoSource` 来自 auth/user 实例；`container.BreachChecker` 改为 `contract.BreachChecker`；`container.CaptchaVerifier` 来自 captcha 实例。装配顺序：先建 tenantMod、userMod、captchaMod，再建 authMod（拿 finalizer），再建 mfaMod/passkeyMod（auth 需要 mfaVerifier 时 mfa 先建）。

- [ ] **Step 2: 同步 e2e / ws 测试装配**

`api_contract_test.go` 与 `api_contract_test.go` 的 `authmodule.New(...)` 改新签名；`newTestAppWithDB` 里补 `mfa/passkey/captcha` 模块（若链路需要 TOTP/webauthn/验证码契约）；`ws_integration_test.go` 同改。`TestAuthRefreshLogout`/`TestAPIContract` 行为不变。

- [ ] **Step 3: catalog 夹具同步**

- `fixture()` 增 `mfa/passkey/breach/captcha`（顺序与真实 entries 一致：captcha 无 Requires/Migrations、breach 无 Requires/Migrations；mfa Requires user；passkey Requires user, auth）。
- `TestCatalogMigrationsShape` want map：auth 由 `true`→保留 `true`（011/012 仍在），增加 `mfa:true, passkey:true, breach:false, captcha:false`。
- `TestDescriptorPermissionsCoverBusinessRoutes`：在 required 增 mfa 6 条 + passkey 7 条（与 Descriptor.Permissions 逐值对齐）；total 由 32 变 32+13=45（按实际 Descriptor 声明数）。
- `TestResolveEmptyMeansAll`：13→17。

- [ ] **Step 4: README / AGENTS 同步**

- 能力树增 `mfa/`、`passkey/`、`breach/`、`captcha/` 与备注（auth 只留会话/凭证/登录历史/密码历史）。
- `shared/totp` 行删除，TOTP 段落改为「TOTP 二次验证归 `mfa` 能力（`internal/capabilities/mfa/totp`，密钥落 `user_mfa` 表 AES-GCM 加密）」。
- 目录树 `capabilities/` 注释随迁移更新。

- [ ] **Step 5: 全量回归**

```bash
gofmt -l . && go build ./... && go vet ./... && golangci-lint run ./...
make check-log-usage && go test ./... -count=1 && make bench-ci && make release-check COMPOSE_ENV=.env.example
```

Expected: `All checks passed`；`go test ./...` 全绿；`grep -rn 'shared/totp' --include='*.go' .` 为空；`grep -rn 'capabilities/auth/domain' --include='*.go' internal/capabilities/oauth` 仅剩既有 TokenPair（P1.7 处理，不属本阶段）。

- [ ] **Step 6: 提交**（用户授权后执行）

`git add -A && git commit -m "feat(capabilities): wire the six auth capabilities into the runtime"`

---

## Self-Review

**1. Spec 覆盖**：§5.1 六能力去向表逐行落实（auth/mfa/passkey/breach/captcha/tenancy→tenant 保留名）；§5.4 tenancy=开通式注册（裁定 #2 保留 tenant 名）；§6.1 Descriptor（mfa/passkey 声明 Requires+Permissions+Migrations）；§7 迁移归属（011/012 留 auth、013 归 mfa、015 归 passkey、004 留 user + 016 新增、scheduled_jobs 不涉及）；§10 P1.6 目标（模块间 import 经 5 条端口归零）。`trusted_devices` 迁移文件的 owner 由 auth 改 mfa（§7 明文 trusted_devices→mfa）。

**2. 占位符扫描**：Task 2 Step 3 的 `UserinfoSource` 用法有兜底（若不便可用 `contract.UserinfoSource` 查 username）；Task 5 Step 1 的 ProvisionedTenant/User 字段补齐有明确动作指引（读 tenant.go/user.go json tag 后补足）。无 TBD/TODO。

**3. 类型一致性**：`contract.MFAVerifier.Enabled/VerifyTOTP`、`contract.LoginFinalizer` 四方法、`contract.TenantProvisioner.Provision(ProvisionRequest)(*ProvisionResult,error)`、`contract.CaptchaVerifier.Enabled/Verify`、`contract.BreachChecker.IsBreached` 在 Task 1 定义、Task 2–5 消费，命名全仓唯一；`TokenPair` 单独定义于 contract，不与 `auth/domain.TokenPair`（含 DeviceToken）混用。

**4. 风险**：① `auth` 的 `New(...deps ...interface{})` 改法要逐 type-assert 对齐 user 模块既有模式，qna 直判；② `passkey` 的锁定/记历史端口扩面后，auth 的 `FinishWebAuthnLogin` 原逻辑要完整平移且登录失败三种路径（lock/cred/assert）不丢审计；③ 016 搬迁密文要求 `users.totp_secret` 与 `user_mfa.totp_secret` 同为 encryption:"true"（已核实 user/domain 与 encryption hook 实现一致），否则需在迁移里显式加解密——实现时用真实 MariaDB 与 PostgreSQL 各验证一次「adopt 后 Up 不丢数据、Down 可回滚」；④ catalog 夹具 13→17 与权限点 total 32→45 需与实际 Descriptor 严格逐值对齐（TestDescriptorPermissionsCoverBusinessRoutes 会钉死）。
