package audit

import (
	"jimu/internal/assembly"
	"jimu/internal/contract"
)

// Wire 装配审计能力：按 audit 段构造模块（段不存在时取零值配置），日志器取内核件。
func Wire(ctx *assembly.Context) (contract.Module, error) {
	cfg := assembly.MustSectionValue[Config](ctx, ConfigKey)
	if job, ok := newAuditRetentionJob(ctx.DB(), cfg.Retention, ctx.Logger()); ok {
		if err := ctx.RegisterJob(job); err != nil {
			return nil, err
		}
	}
	return New(ctx.DB(), cfg, ctx.Logger()), nil
}
