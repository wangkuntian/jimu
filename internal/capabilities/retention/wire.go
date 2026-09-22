package retention

import (
	"context"
	"fmt"
	"time"

	"jimu/internal/assembly"
	"jimu/internal/contract"
	"jimu/internal/kernel/scheduler"
)

// Wire 装配数据保留能力：解码 retention 段，向组合根贡献两个定时任务 ——
// cleanup（增长型历史表清理，每日 03:00）与 retention（按策略保留，默认每日 03:30）。
// 保留不属 catalog 能力，「是否启用」由本段自身的 enabled 决定。
func Wire(ctx *assembly.Context) (contract.Module, error) {
	cfg, err := Load(ctx.Sections())
	if err != nil {
		return nil, fmt.Errorf("init retention config: %w", err)
	}
	db := ctx.DB()
	if db != nil {
		cleanupSvc := NewCleanupService(db, DefaultCleanupConfig())
		if err := ctx.RegisterJob(scheduler.Job{ID: "cleanup", Name: "Data Cleanup", Spec: "0 3 * * *", Run: func() {
			results, err := cleanupSvc.Run(context.Background())
			if err != nil {
				ctx.Logger().Errorw("cleanup job failed", "error", err.Error())
				return
			}
			for _, r := range results {
				if r.Deleted > 0 {
					ctx.Logger().Infow("cleanup completed", "table", r.Table, "deleted", r.Deleted)
				}
			}
		}}); err != nil {
			return nil, err
		}
	}
	if db != nil && cfg.Enabled {
		retentionSvc := NewRetentionService(db, *cfg)
		spec := cfg.Cron
		if spec == "" {
			spec = "30 3 * * *"
		}
		if err := ctx.RegisterJob(scheduler.Job{ID: "retention", Name: "History Retention", Spec: spec, Run: func() {
			runCtx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
			defer cancel()
			results, err := retentionSvc.Run(runCtx)
			if err != nil {
				ctx.Logger().Errorw("retention job failed", "error", err.Error())
				return
			}
			for _, r := range results {
				if r.Deleted > 0 {
					ctx.Logger().Infow("retention completed", "table", r.Table, "deleted", r.Deleted)
				}
			}
		}}); err != nil {
			return nil, err
		}
	}
	return nil, nil
}
