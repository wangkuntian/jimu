package application

import "context"

// TenantQuota 租户配额校验（由 tenant 模块的 QuotaService 实现；nil = 未启用配额）
type TenantQuota interface {
	// CheckRoleQuota 角色数达到套餐上限时返回 CodeQuotaExceeded
	CheckRoleQuota(ctx context.Context, tenantID uint64) error
}
