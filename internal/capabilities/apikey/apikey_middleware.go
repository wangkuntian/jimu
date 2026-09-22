package apikey

import (
	auth "jimu/internal/kernel/auth"
	"jimu/internal/kernel/tenant"
	"jimu/internal/shared/errors"
	"jimu/internal/shared/response"

	"github.com/gin-gonic/gin"
)

// APIKeyHeader API Key 请求头
const APIKeyHeader = "X-API-Key"

// ScopeProtected 受保护路由（受保护挂载点的全部能力）要求的最小 scope：
// machine 形态无 auth/会话，服务密钥需显式携带它（"*" 视为全权，见 APIKey.HasScope）。
const ScopeProtected = "api:access"

// APIKeyAuthMiddleware API Key 认证中间件（服务/机器间调用）
// 验证 X-API-Key 头，通过后把 APIKey 注入 context（APIKeyFromContext 读取），
// 并按 Key 自身归属租户注入请求上下文（与 JWT 路径的 tenant_id 约定一致）。
func APIKeyAuthMiddleware(verifier *APIKeyVerifier) gin.HandlerFunc {
	return func(c *gin.Context) {
		providedKey := c.GetHeader(APIKeyHeader)
		if providedKey == "" {
			response.Fail(c, errors.New(errors.CodeUnauthorized, "missing api key"))
			c.Abort()
			return
		}

		apiKey, err := verifier.Verify(c.Request.Context(), providedKey)
		if err != nil {
			response.Fail(c, errors.New(errors.CodeUnauthorized, "invalid api key"))
			c.Abort()
			return
		}

		// 注入已验证的 API Key 与 scope
		c.Set("api_key", apiKey)
		ctx := auth.ContextWithAPIKey(c.Request.Context(), apiKey)
		// 租户只来自 Key 归属，不接受客户端 header/query 传入
		if apiKey.TenantID != 0 {
			c.Set("tenant_id", apiKey.TenantID)
			ctx = tenant.WithTenant(ctx, apiKey.TenantID)
		}
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}

// RequireScope 校验已认证的 API Key 是否具备指定 scope（需前置 APIKeyAuthMiddleware）。
// 语义：空 scopes 拒绝一切，只有显式包含 "*" 才代表全权（见 APIKey.HasScope）。
func RequireScope(scope string) gin.HandlerFunc {
	return func(c *gin.Context) {
		apiKey, ok := auth.APIKeyFromContext(c.Request.Context())
		if !ok {
			response.Fail(c, errors.New(errors.CodeUnauthorized, "api key required"))
			c.Abort()
			return
		}
		if !apiKey.HasScope(scope) {
			response.Fail(c, errors.New(errors.CodeForbidden, "insufficient scope: "+scope))
			c.Abort()
			return
		}
		c.Next()
	}
}
