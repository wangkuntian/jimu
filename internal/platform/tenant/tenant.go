// Package tenant 提供租户上下文的注入与提取。
//
// 租户身份来源于 JWT claim（登录时按用户归属确定），经 AuthMiddleware 写入
// gin context，再由 Middleware 写入 request context；业务层只从 context 读取，
// 不接受客户端 header/query 传入，杜绝租户伪造。
package tenant

import (
	"context"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"
)

// DefaultTenantID 默认租户 ID，与迁移 005_add_tenants.sql 写入的初始租户一致。
// 上下文中无租户（tid=0，如滚动升级期的旧 token）时创建的资源归属默认租户，
// 查询不做租户过滤（平台级视角）。
const DefaultTenantID uint64 = 1

// tenantCodePattern 租户编码格式：1-64 位字母、数字、短横线、下划线。
// 统一在此校验与归一化（单一事实源），供 tenant 模块与开通式注册共用。
var tenantCodePattern = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

// NormalizeCode 归一化租户编码：转小写（编码不区分大小写语义，为子域名/URL 场景兜底）。
func NormalizeCode(code string) string {
	return strings.ToLower(code)
}

// ValidCode 校验租户编码格式：1-64 位字母、数字、短横线、下划线。
func ValidCode(code string) bool {
	return len(code) <= 64 && tenantCodePattern.MatchString(code)
}

type ctxKey struct{}

// WithTenant 将租户 ID 注入 context。
func WithTenant(ctx context.Context, tenantID uint64) context.Context {
	return context.WithValue(ctx, ctxKey{}, tenantID)
}

// FromContext 从 context 提取租户 ID；不存在时返回 0（未隔离，视为平台级访问）。
func FromContext(ctx context.Context) uint64 {
	if v, ok := ctx.Value(ctxKey{}).(uint64); ok {
		return v
	}
	return 0
}

// Visible 判断资源归属租户对上下文租户是否可见：
// 上下文租户为 0（平台级视角）或资源未归属（0）时可见，否则须为同一租户。
func Visible(resourceTenant, ctxTenant uint64) bool {
	return ctxTenant == 0 || resourceTenant == 0 || resourceTenant == ctxTenant
}

// Middleware 将 gin context 中的 tenant_id（由 AuthMiddleware 从 JWT claim 注入）
// 写入 request context，供业务层通过 FromContext 读取。
func Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if v, ok := c.Get("tenant_id"); ok {
			if id, ok := v.(uint64); ok && id != 0 {
				c.Request = c.Request.WithContext(WithTenant(c.Request.Context(), id))
			}
		}
		c.Next()
	}
}
