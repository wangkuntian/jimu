package queue

import (
	"context"
	"errors"
	"strings"
	"time"

	"jimu/internal/capabilities/queue/domain"
	"jimu/internal/kernel/logger"
	"jimu/internal/kernel/scheduler"
	"jimu/internal/shared/dbpurge"

	"gorm.io/gorm"
)

// RetentionConfig controls cleanup of queue-owned records.
type RetentionConfig struct {
	Enabled        bool   `mapstructure:"enabled"`
	Cron           string `mapstructure:"cron"`
	BatchSize      int    `mapstructure:"batch_size"`
	JobDays        int    `mapstructure:"job_days"`
	JobHistoryDays int    `mapstructure:"job_history_days"`
	DeadLetterDays int    `mapstructure:"dead_letter_days"`
}

func (c RetentionConfig) Validate() error {
	if c.Enabled && strings.TrimSpace(c.Cron) == "" {
		return errors.New("queue.retention.cron is required when enabled")
	}
	if c.BatchSize < 0 || c.JobDays < 0 || c.JobHistoryDays < 0 || c.DeadLetterDays < 0 {
		return errors.New("queue.retention values must not be negative")
	}
	return nil
}

func queueRetentionRules(cfg RetentionConfig) []dbpurge.Rule {
	return []dbpurge.Rule{
		{Table: "jobs", Model: &domain.Job{}, TimeColumn: "updated_at", Condition: "status IN ?", Args: []any{[]string{domain.JobStatusSuccess, domain.JobStatusDead}}, Days: cfg.JobDays},
		{Table: "job_history", Model: &domain.JobHistory{}, TimeColumn: "ended_at", Days: cfg.JobHistoryDays},
		{Table: "dead_letters", Model: &domain.DeadLetter{}, TimeColumn: "resolved_at", Condition: "resolved = ?", Args: []any{true}, Days: cfg.DeadLetterDays},
	}
}

func newQueueRetentionJob(db *gorm.DB, cfg RetentionConfig, log *logger.Logger) (scheduler.Job, bool) {
	if db == nil || !cfg.Enabled {
		return scheduler.Job{}, false
	}
	spec := cfg.Cron
	if spec == "" {
		spec = "30 3 * * *"
	}
	service := dbpurge.New(db, cfg.BatchSize)
	return scheduler.Job{
		ID:   "queue_retention",
		Name: "Queue History Retention",
		Spec: spec,
		Run: func() {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
			defer cancel()
			results, err := service.Run(ctx, queueRetentionRules(cfg))
			if err != nil {
				if log != nil {
					log.Errorw("queue retention job failed", "error", err.Error())
				}
				return
			}
			for _, result := range results {
				if result.Deleted > 0 && log != nil {
					log.Infow("queue retention completed", "table", result.Table, "deleted", result.Deleted)
				}
			}
		},
	}, true
}
