package access

import (
	"context"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"jimu/internal/contract"
	"jimu/internal/shared/errors"
	"jimu/internal/shared/response"

	"github.com/casbin/casbin/v3"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type Policy struct {
	Role     string
	Resource string
	Action   string
}

type RoleIdentity struct {
	ID    uint64
	Name  string
	Scope contract.RoleScope
}

func RoleSubject(roleID uint64) string {
	return "role:" + strconv.FormatUint(roleID, 10)
}

type AuthorizationStore interface {
	RolesForUser(ctx context.Context, userID uint64) ([]RoleIdentity, error)
	Policies(ctx context.Context) ([]Policy, error)
}

type DBAuthorizationStore struct {
	db *gorm.DB
}

func NewDBAuthorizationStore(db *gorm.DB) *DBAuthorizationStore {
	return &DBAuthorizationStore{db: db}
}

func (s *DBAuthorizationStore) RolesForUser(ctx context.Context, userID uint64) ([]RoleIdentity, error) {
	var rows []struct {
		ID    uint64
		Name  string
		Scope string
	}
	err := s.db.WithContext(ctx).
		Table("roles").
		Select("roles.id, roles.name, roles.scope").
		Joins("JOIN user_roles ur ON ur.role_id = roles.id").
		Where("ur.user_id = ? AND roles.deleted_at IS NULL AND roles.status = 1", userID).
		Order("roles.id").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	roles := make([]RoleIdentity, 0, len(rows))
	for _, row := range rows {
		scope := contract.RoleScope(row.Scope)
		if scope == "" {
			scope = contract.RoleScopeTenant
		}
		roles = append(roles, RoleIdentity{
			ID:    row.ID,
			Name:  row.Name,
			Scope: scope,
		})
	}
	return roles, nil
}

func (s *DBAuthorizationStore) Policies(ctx context.Context) ([]Policy, error) {
	var rows []struct {
		RoleID   uint64
		Resource string
		Action   string
	}
	err := s.db.WithContext(ctx).
		Table("roles").
		Select("roles.id AS role_id, permissions.resource, permissions.action").
		Joins("JOIN role_permissions rp ON rp.role_id = roles.id").
		Joins("JOIN permissions ON permissions.id = rp.permission_id").
		Where("roles.deleted_at IS NULL AND roles.status = 1 AND permissions.deleted_at IS NULL").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	policies := make([]Policy, 0, len(rows))
	for _, row := range rows {
		policies = append(policies, Policy{
			Role:     RoleSubject(row.RoleID),
			Resource: row.Resource,
			Action:   row.Action,
		})
	}
	return policies, nil
}

// PolicyCacheTTL 策略缓存有效期。
// 权限变更经 TTL 后自动生效；用户角色（roles）每请求实时查询，撤销/分配角色即时反映。
// 测试可通过 PolicyCacheTTL 缩短 TTL 验证权限变更生效路径。
var PolicyCacheTTL = 30 * time.Second

var policyGeneration atomic.Uint64

// InvalidatePolicyCache 让下一次鉴权忽略已有策略缓存。
// 角色名称/状态或角色权限发生变化时调用，确保授权变更立即生效。
func InvalidatePolicyCache() {
	policyGeneration.Add(1)
}

func AuthorizationMiddleware(store AuthorizationStore, enforcer *casbin.Enforcer) gin.HandlerFunc {
	// mu 串行化 enforcer 重建（casbin 非并发安全）
	var mu sync.Mutex
	// 策略缓存：避免每请求全量 DB 查询 + 重建 enforcer
	var cacheMu sync.Mutex
	var cachedPolicies []Policy
	var cachedAt time.Time
	var cachedGeneration uint64

	loadPolicies := func(ctx context.Context) ([]Policy, error) {
		cacheMu.Lock()
		defer cacheMu.Unlock()
		generation := policyGeneration.Load()
		if time.Since(cachedAt) < PolicyCacheTTL && cachedPolicies != nil && cachedGeneration == generation {
			return cachedPolicies, nil
		}
		policies, err := store.Policies(ctx)
		if err != nil {
			return nil, err
		}
		cachedPolicies = policies
		cachedAt = time.Now()
		cachedGeneration = generation
		return policies, nil
	}

	return func(c *gin.Context) {
		userID, _ := c.Get("user_id")
		id, ok := userID.(uint64)
		if !ok || id == 0 {
			response.Fail(c, errors.New(errors.CodeUnauthorized, "missing user context"))
			c.Abort()
			return
		}

		roles, err := store.RolesForUser(c.Request.Context(), id)
		if err != nil {
			response.Fail(c, errors.Wrap(errors.CodeInternalError, "failed to load roles", err))
			c.Abort()
			return
		}
		if len(roles) == 0 {
			response.Fail(c, errors.New(errors.CodeForbidden, "no roles assigned"))
			c.Abort()
			return
		}
		roleNames := make([]string, 0, len(roles))
		roleSubjects := make([]string, 0, len(roles))
		platformScope := false
		for _, role := range roles {
			roleNames = append(roleNames, role.Name)
			roleSubjects = append(roleSubjects, RoleSubject(role.ID))
			if role.Scope == contract.RoleScopePlatform {
				platformScope = true
			}
		}
		c.Set("roles", roleNames)

		policies, err := loadPolicies(c.Request.Context())
		if err != nil {
			response.Fail(c, errors.Wrap(errors.CodeInternalError, "failed to load policies", err))
			c.Abort()
			return
		}

		mu.Lock()
		defer mu.Unlock()
		enforcer.ClearPolicy()
		for _, subject := range roleSubjects {
			_, _ = enforcer.AddGroupingPolicy(subject, subject)
		}
		for _, policy := range policies {
			_, _ = enforcer.AddPolicy(policy.Role, policy.Resource, policy.Action)
		}

		allowed := false
		for _, subject := range roleSubjects {
			ok, err := enforcer.Enforce(subject, c.Request.URL.Path, c.Request.Method)
			if err == nil && ok {
				allowed = true
				break
			}
		}
		if !allowed {
			response.Fail(c, errors.New(errors.CodeForbidden, "permission denied"))
			c.Abort()
			return
		}
		c.Set("platform_scope", platformScope)
		c.Next()
	}
}
