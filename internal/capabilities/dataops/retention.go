package dataops

import (
	"context"
	"strings"
	"time"

	"jimu/internal/capabilities/dataops/domain"
	"jimu/internal/kernel/logger"
	"jimu/internal/kernel/scheduler"
	"jimu/internal/shared/dbpurge"

	"gorm.io/gorm"
)

func dataopsRetentionRules(cfg RetentionConfig) []dbpurge.Rule {
	return []dbpurge.Rule{{
		Table:      "import_jobs",
		Model:      &domain.ImportJob{},
		TimeColumn: "created_at",
		Condition:  "status IN ?",
		Args:       []any{[]string{domain.ImportJobCompleted, domain.ImportJobFailed}},
		Days:       cfg.ImportJobDays,
	}}
}

func newDataopsRetentionJob(db *gorm.DB, cfg RetentionConfig, log *logger.Logger) (scheduler.Job, bool) {
	if db == nil || !cfg.Enabled {
		return scheduler.Job{}, false
	}
	spec := strings.TrimSpace(cfg.Cron)
	if spec == "" {
		spec = "30 3 * * *"
	}
	service := dbpurge.New(db, cfg.BatchSize)
	return scheduler.Job{
		ID:   "dataops_retention",
		Name: "Import Job Retention",
		Spec: spec,
		Run: func() {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
			defer cancel()
			results, err := service.Run(ctx, dataopsRetentionRules(cfg))
			if err != nil {
				if log != nil {
					log.Errorw("dataops retention job failed", "error", err.Error())
				}
				return
			}
			for _, result := range results {
				if result.Deleted > 0 && log != nil {
					log.Infow("dataops retention completed", "table", result.Table, "deleted", result.Deleted)
				}
			}
		},
	}, true
}
