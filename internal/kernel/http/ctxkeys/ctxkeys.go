// Package ctxkeys 持有中间件在 context 中读写身份信息时使用的键。
//
// 单独成包是为了让 HTTP 中间件不 import kernel/auth 与 kernel/tenant：
// 只要中间件依赖这两个包，0 能力项目也会把 JWT、会话、Redis 一并编进二进制。
// 键的类型是空 struct，跨包使用时按值比较，因此 auth/tenant 包与中间件看到的是同一个键。
package ctxkeys

// TenantID 是租户 ID 在 context 中的键（值为 uint64）。
type TenantID struct{}

// APIKeyID 是已认证 API Key 的 ID 在 context 中的键（值为 uint64）。
// 中间件只需要 ID 做限流分桶，不需要 auth 包的完整机制视图。
type APIKeyID struct{}
