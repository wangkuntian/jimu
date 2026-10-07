package audit

import (
	"context"
	"time"

	auditdomain "jimu/internal/capabilities/audit/domain"
	"jimu/internal/kernel/logger"
	"jimu/internal/kernel/scheduler"
	"jimu/internal/shared/dbpurge"

	"gorm.io/gorm"
)

func auditRetentionRules(cfg RetentionConfig) []dbpurge.Rule {
	return []dbpurge.Rule{{
		Table:      "audit_logs",
		Model:      &auditdomain.AuditLog{},
		TimeColumn: "created_at",
		Condition:  "entry_hash = ''",
		Days:       cfg.AuditLogDays,
	}}
}

func newAuditRetentionJob(db *gorm.DB, cfg RetentionConfig, log *logger.Logger) (scheduler.Job, bool) {
	if db == nil || !cfg.Enabled {
		return scheduler.Job{}, false
	}
	spec := cfg.Cron
	if spec == "" {
		spec = "30 3 * * *"
	}
	service := dbpurge.New(db, cfg.BatchSize)
	return scheduler.Job{
		ID:   "audit_retention",
		Name: "Audit Log Retention",
		Spec: spec,
		Run: func() {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
			defer cancel()
			results, err := service.Run(ctx, auditRetentionRules(cfg))
			if err != nil {
				if log != nil {
					log.Errorw("audit retention job failed", "error", err.Error())
				}
				return
			}
			for _, result := range results {
				if result.Deleted > 0 && log != nil {
					log.Infow("audit retention completed", "table", result.Table, "deleted", result.Deleted)
				}
			}
		},
	}, true
}
