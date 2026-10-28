package application

import "context"

// TenantQuota 租户配额校验（由 tenant 模块的 QuotaService 实现；nil = 未启用配额）
type TenantQuota interface {
	// CheckRoleQuota 角色数达到套餐上限时返回 CodeQuotaExceeded
	CheckRoleQuota(ctx context.Context, tenantID uint64) error
	// TenantExists 用于平台管理员创建角色时校验目标租户。
	TenantExists(ctx context.Context, tenantID uint64) (bool, error)
}
