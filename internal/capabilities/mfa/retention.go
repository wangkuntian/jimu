package mfa

import (
	"context"
	"strings"
	"time"

	mfadomain "jimu/internal/capabilities/mfa/domain"
	"jimu/internal/kernel/logger"
	"jimu/internal/kernel/scheduler"
	"jimu/internal/shared/dbpurge"

	"gorm.io/gorm"
)

func mfaRetentionRules(cfg RetentionConfig) []dbpurge.Rule {
	return []dbpurge.Rule{{
		Table:      "trusted_devices",
		Model:      &mfadomain.TrustedDevice{},
		TimeColumn: "expires_at",
		Days:       cfg.ExpiredDeviceDays,
	}}
}

func newMFARetentionJob(db *gorm.DB, cfg RetentionConfig, log *logger.Logger) (scheduler.Job, bool) {
	if db == nil || !cfg.Enabled {
		return scheduler.Job{}, false
	}
	spec := strings.TrimSpace(cfg.Cron)
	if spec == "" {
		spec = "30 3 * * *"
	}
	service := dbpurge.New(db, cfg.BatchSize)
	return scheduler.Job{
		ID:   "mfa_retention",
		Name: "Expired Trusted Device Retention",
		Spec: spec,
		Run: func() {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
			defer cancel()
			results, err := service.Run(ctx, mfaRetentionRules(cfg))
			if err != nil {
				if log != nil {
					log.Errorw("mfa retention job failed", "error", err.Error())
				}
				return
			}
			for _, result := range results {
				if result.Deleted > 0 && log != nil {
					log.Infow("mfa retention completed", "table", result.Table, "deleted", result.Deleted)
				}
			}
		},
	}, true
}
