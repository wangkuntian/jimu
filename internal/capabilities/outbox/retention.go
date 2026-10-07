package outbox

import (
	"context"
	"errors"
	"strings"
	"time"

	"jimu/internal/kernel/logger"
	"jimu/internal/kernel/scheduler"
	"jimu/internal/shared/dbpurge"

	"gorm.io/gorm"
)

// RetentionConfig controls cleanup of published outbox events.
type RetentionConfig struct {
	Enabled   bool   `mapstructure:"enabled"`
	Cron      string `mapstructure:"cron"`
	BatchSize int    `mapstructure:"batch_size"`
	EventDays int    `mapstructure:"outbox_event_days"`
}

func (c RetentionConfig) Validate() error {
	if c.Enabled && strings.TrimSpace(c.Cron) == "" {
		return errors.New("outbox.retention.cron is required when enabled")
	}
	if c.BatchSize < 0 || c.EventDays < 0 {
		return errors.New("outbox.retention values must not be negative")
	}
	return nil
}

func outboxRetentionRules(cfg RetentionConfig) []dbpurge.Rule {
	return []dbpurge.Rule{{
		Table:      "outbox_events",
		Model:      &Event{},
		TimeColumn: "published_at",
		Condition:  "published_at IS NOT NULL",
		Days:       cfg.EventDays,
	}}
}

func newOutboxRetentionJob(db *gorm.DB, cfg RetentionConfig, log *logger.Logger) (scheduler.Job, bool) {
	if db == nil || !cfg.Enabled {
		return scheduler.Job{}, false
	}
	spec := cfg.Cron
	if spec == "" {
		spec = "30 3 * * *"
	}
	service := dbpurge.New(db, cfg.BatchSize)
	return scheduler.Job{
		ID:   "outbox_retention",
		Name: "Published Outbox Event Retention",
		Spec: spec,
		Run: func() {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
			defer cancel()
			results, err := service.Run(ctx, outboxRetentionRules(cfg))
			if err != nil {
				if log != nil {
					log.Errorw("outbox retention job failed", "error", err.Error())
				}
				return
			}
			for _, result := range results {
				if result.Deleted > 0 && log != nil {
					log.Infow("outbox retention completed", "table", result.Table, "deleted", result.Deleted)
				}
			}
		},
	}, true
}
