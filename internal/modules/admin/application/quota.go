package application

import "context"

// TenantQuota 租户配额校验（由 tenant 模块的 QuotaService 实现；nil = 未启用配额）。
// 只声明本模块用到的能力，避免跨模块依赖具体类型。
type TenantQuota interface {
	// CheckUserQuota 用户数达到套餐上限时返回 CodeQuotaExceeded
	CheckUserQuota(ctx context.Context, tenantID uint64) error
	// CheckAPIKeyQuota API Key 数量达到套餐上限时返回 CodeQuotaExceeded
	CheckAPIKeyQuota(ctx context.Context, tenantID uint64) error
}
