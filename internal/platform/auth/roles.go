package auth

import (
	"context"
	"sync"
	"time"

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

type AuthorizationStore interface {
	RolesForUser(ctx context.Context, userID uint64) ([]string, error)
	Policies(ctx context.Context) ([]Policy, error)
}

type DBAuthorizationStore struct {
	db *gorm.DB
}

func NewDBAuthorizationStore(db *gorm.DB) *DBAuthorizationStore {
	return &DBAuthorizationStore{db: db}
}

func (s *DBAuthorizationStore) RolesForUser(ctx context.Context, userID uint64) ([]string, error) {
	var roles []string
	err := s.db.WithContext(ctx).
		Table("roles").
		Select("roles.name").
		Joins("JOIN user_roles ur ON ur.role_id = roles.id").
		Where("ur.user_id = ?", userID).
		Pluck("roles.name", &roles).Error
	return roles, err
}

func (s *DBAuthorizationStore) Policies(ctx context.Context) ([]Policy, error) {
	var policies []Policy
	err := s.db.WithContext(ctx).
		Table("roles").
		Select("roles.name AS role, permissions.resource, permissions.action").
		Joins("JOIN role_permissions rp ON rp.role_id = roles.id").
		Joins("JOIN permissions ON permissions.id = rp.permission_id").
		Scan(&policies).Error
	return policies, err
}

// policyCacheTTL 策略缓存有效期。
// 权限变更经 TTL 后自动生效；用户角色（roles）每请求实时查询，撤销/分配角色即时反映。
const policyCacheTTL = 30 * time.Second

func AuthorizationMiddleware(store AuthorizationStore, enforcer *casbin.Enforcer) gin.HandlerFunc {
	// mu 串行化 enforcer 重建（casbin 非并发安全）
	var mu sync.Mutex
	// 策略缓存：避免每请求全量 DB 查询 + 重建 enforcer
	var cacheMu sync.Mutex
	var cachedPolicies []Policy
	var cachedAt time.Time

	loadPolicies := func(ctx context.Context) ([]Policy, error) {
		cacheMu.Lock()
		defer cacheMu.Unlock()
		if time.Since(cachedAt) < policyCacheTTL && cachedPolicies != nil {
			return cachedPolicies, nil
		}
		policies, err := store.Policies(ctx)
		if err != nil {
			return nil, err
		}
		cachedPolicies = policies
		cachedAt = time.Now()
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
		c.Set("roles", roles)

		policies, err := loadPolicies(c.Request.Context())
		if err != nil {
			response.Fail(c, errors.Wrap(errors.CodeInternalError, "failed to load policies", err))
			c.Abort()
			return
		}

		mu.Lock()
		defer mu.Unlock()
		enforcer.ClearPolicy()
		for _, role := range roles {
			_, _ = enforcer.AddGroupingPolicy(role, role)
		}
		for _, policy := range policies {
			_, _ = enforcer.AddPolicy(policy.Role, policy.Resource, policy.Action)
		}

		allowed := false
		for _, role := range roles {
			ok, err := enforcer.Enforce(role, c.Request.URL.Path, c.Request.Method)
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
		c.Next()
	}
}
