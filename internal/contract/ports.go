// Package contract 端口定义：能力间只经此处接口调用（见 AGENTS.md 能力边界）。
package contract

import (
	"context"
	"errors"
	"time"
)

// ErrNotFound 端口查询目标不存在（由端口实现返回，消费方据此映射语义）。
var ErrNotFound = errors.New("record not found")

// Userinfo 用户信息端口视图（端口自有结构，不含 gorm 标签）。
type Userinfo struct {
	ID        uint64
	Username  string
	Status    int8
	TenantID  uint64
	CreatedAt time.Time
}

// UserinfoSource UserInfoService 所需的用户只读数据端口，
// 由 user 能力提供实现，替代消费方对 user/domain 的直接引用。
type UserinfoSource interface {
	// GetByID 按 ID 查用户；不存在时返回 ErrNotFound。
	GetByID(ctx context.Context, id uint64) (*Userinfo, error)
	// FindByUsername 按用户名查用户；不存在时返回 ErrNotFound。
	FindByUsername(ctx context.Context, username string) (*Userinfo, error)
	// List 分页查用户（page 从 1 起；按 id 降序，平台级视角不过滤租户）。
	List(ctx context.Context, page, pageSize int) ([]Userinfo, int64, error)
}

// TokenPair 登录完成后的令牌视图。
type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	// DeviceToken 仅在「记住此设备」成功签发时返回，客户端需安全保存。
	DeviceToken string `json:"device_token,omitempty"`
}

// 登录历史状态（auth 与 passkey 共用的登录审计结果字面量）。
const (
	LoginStatusSuccess = "success"
	LoginStatusFailed  = "failed"
	LoginStatusLocked  = "locked"
)

// LoginFinalizer auth 提供：为已验证用户签发令牌、建会话、记登录历史、发事件。
type LoginFinalizer interface {
	FinalizeLogin(ctx context.Context, userID uint64) (*TokenPair, error)
	// CheckLocked 登录前的失败锁定检查（未锁定返回 false）。
	CheckLocked(ctx context.Context, username string) (locked bool, remaining time.Duration, err error)
	// RecordFailure 记录一次失败尝试（不影响主流程）。
	RecordFailure(ctx context.Context, username string)
	// ResetFailure 登录成功后清除失败计数。
	ResetFailure(ctx context.Context, username string)
	// RecordLoginHistory 记录一次登录尝试（成功/失败/锁定）。
	RecordLoginHistory(ctx context.Context, userID, tenantID uint64, username, status, reason string)
}

// MFAVerifier mfa 提供：登录时判定是否启用二次验证并校验 TOTP 码。
// 错误语义由实现返回 shared/errors 的 CodeMFARequired / CodeInvalidMFA。
// 客户端信息（DeviceToken/RememberDevice）经 context 传入（contract.WithLoginDevice）。
type MFAVerifier interface {
	// Enabled 用户是否启用 MFA。
	Enabled(ctx context.Context, userID uint64) (bool, error)
	// VerifyTOTP 校验 TOTP 码；携带可信设备令牌且未过期时允许跳过（内部判定）。
	VerifyTOTP(ctx context.Context, userID, tenantID uint64, code string) error
	// MaybeIssueDevice 登录成功且客户端请求"记住此设备"时签发可信设备令牌；
	// 未启用 MFA 或未请求记住时返回空串（签发失败不阻断登录）。
	MaybeIssueDevice(ctx context.Context, userID, tenantID uint64) string
	// RevokeDevices 吊销该用户全部可信设备（改密/登出全部设备时调用，不影响主流程）。
	RevokeDevices(ctx context.Context, userID uint64)
}

// TenantProvisioner tenant 提供：开通式注册（注册 = 开通新租户）。nil = 未启用。
type TenantProvisioner interface {
	Provision(ctx context.Context, req ProvisionRequest) (*ProvisionResult, error)
}

// ProvisionRequest 开通式注册请求（端口自有视图，不含 gorm 标签）。
type ProvisionRequest struct {
	Username     string
	PasswordHash string
	Email        string
	Phone        string
	TenantName   string
	TenantCode   string
}

// ProvisionResult 开通式注册结果：新租户 + owner 用户。
type ProvisionResult struct {
	Tenant ProvisionedTenant `json:"tenant"`
	User   ProvisionedUser   `json:"user"`
}

// ProvisionedTenant 开通结果中的租户视图（序列化字段与原 tenant 域模型对齐）。
type ProvisionedTenant struct {
	ID        uint64    `json:"id"`
	Code      string    `json:"code"`
	Name      string    `json:"name"`
	Status    int8      `json:"status"`
	PlanID    uint64    `json:"plan_id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ProvisionedUser 开通结果中的 owner 用户视图（序列化字段与原 user 域模型对齐；
// TOTP 已迁出 user 域，故不含 totp_enabled 字段）。
type ProvisionedUser struct {
	ID        uint64    `json:"id"`
	Username  string    `json:"username"`
	Status    int8      `json:"status"`
	Email     string    `json:"email"`
	Phone     string    `json:"phone"`
	TenantID  uint64    `json:"tenant_id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// BreachChecker breach 提供：泄露口令检查。nil = 未启用。
type BreachChecker interface {
	// IsBreached 返回口令是否已泄露；网络/解析失败返回 error，由调用方决定放行还是拒绝。
	IsBreached(ctx context.Context, password string) (bool, error)
}

// CaptchaVerifier captcha 提供：登录/注册验证码校验。
type CaptchaVerifier interface {
	Enabled() bool
	Verify(ctx context.Context, id, code string) error
}

// UserRoleAssigner access 提供：替换用户的全部角色（user_roles 表所有者）。
// 供 user 能力的管理面用例经端口调用，避免跨能力写他人的表。
type UserRoleAssigner interface {
	// AssignRoles 用 roleNames 替换该用户的全部角色；角色名在用户所属租户内解析。
	AssignRoles(ctx context.Context, userID uint64, roleNames []string) error
}
