package tenant

import (
	"context"

	"github.com/gin-gonic/gin"
)

// Tenant 租户信息
type Tenant struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// key 用于在 Gin context 中存储租户信息
type key struct{}

var tenantKey = key{}

// WithTenant 将租户信息注入 Gin context
func WithTenant(c *gin.Context, tenant Tenant) {
	c.Set(tenantKey, tenant)
}

// FromContext 从 Gin context 获取租户信息
func FromContext(c *gin.Context) (Tenant, bool) {
	v, ok := c.Get(tenantKey)
	if !ok {
		return Tenant{}, false
	}
	t, ok := v.(Tenant)
	return t, ok
}

// MustFromContext 从 Gin context 获取租户信息（必须存在）
func MustFromContext(c *gin.Context) Tenant {
	t, _ := FromContext(c)
	return t
}

// Middleware 多租户中间件
// 从 Header、Query 或 Subdomain 提取租户 ID
func Middleware(headerName string) gin.HandlerFunc {
	if headerName == "" {
		headerName = "X-Tenant-ID"
	}

	return func(c *gin.Context) {
		tenantID := c.GetHeader(headerName)
		if tenantID == "" {
			tenantID = c.Query("tenant_id")
		}

		if tenantID != "" {
			WithTenant(c, Tenant{ID: tenantID})
		}

		c.Next()
	}
}

// Scope 创建带租户过滤的查询作用域
// 用于 Gorm 的 Scopes 模式
func Scope(tenantID string) func(db interface{ Where(query interface{}, args ...interface{}) interface{} }) interface{} {
	return func(db interface{ Where(query interface{}, args ...interface{}) interface{} }) interface{} {
		return db.Where("tenant_id = ?", tenantID)
	}
}

// TenantKey 返回 context key（用于自定义存储）
func TenantKey() context.Context {
	return context.Background()
}
