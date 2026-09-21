package application

import "context"

// TenantQuota 租户配额校验（由 tenant 能力实现；nil = 未启用配额）。
// 只声明本能力用到的能力，避免跨能力依赖具体类型。
type TenantQuota interface {
	// CheckAPIKeyQuota API Key 数量达到套餐上限时返回 CodeQuotaExceeded
	CheckAPIKeyQuota(ctx context.Context, tenantID uint64) error
}
