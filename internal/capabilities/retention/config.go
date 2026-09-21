package retention

import (
	"errors"
	"strings"

	"jimu/internal/config"
)

// ConfigKey 本包在 app.yaml 中的配置段键。
const ConfigKey = "retention"

// Config 保留策略配置（原 config.RetentionConfig，P2.1 下沉）。
type Config struct {
	Enabled         bool   `mapstructure:"enabled"`
	Cron            string `mapstructure:"cron"`              // 调度表达式（默认每天 03:30）
	BatchSize       int    `mapstructure:"batch_size"`        // 每批删除行数（默认 500）
	AuditLogDays    int    `mapstructure:"audit_log_days"`    // 审计日志保留天数，0=不清理
	JobDays         int    `mapstructure:"job_days"`          // 已终态任务保留天数
	JobHistoryDays  int    `mapstructure:"job_history_days"`  // 任务执行历史保留天数
	DeadLetterDays  int    `mapstructure:"dead_letter_days"`  // 已处理死信保留天数
	OutboxEventDays int    `mapstructure:"outbox_event_days"` // 已发布 outbox 事件保留天数
	ImportJobDays   int    `mapstructure:"import_job_days"`   // 已结束导入任务保留天数
	// 失效可信设备的保留天数（按 expires_at 计，留出审计窗口后清理）
	TrustedDeviceDays int `mapstructure:"trusted_device_days"`
}

// ApplyDefaults 本包无配置层默认值：每批行数缺省由 NewRetentionService 兜底
// （batch<=0 → defaultRetentionBatchSize），与下沉前一致。
func (c *Config) ApplyDefaults() {}

// Validate 校验本配置段。保留能力不属 catalog 能力，「不启用」由本段自身的
// enabled 决定，因此校验条件与下沉前逐字一致。
func (c Config) Validate() error {
	if c.Enabled {
		if strings.TrimSpace(c.Cron) == "" {
			return errors.New("retention.cron: required when retention.enabled is true")
		}
		if c.BatchSize < 0 {
			return errors.New("retention.batch_size: must not be negative")
		}
	}
	return nil
}

// Load 解码并校验本包配置段。保留不属 catalog 能力，由组合根无条件加载。
func Load(dec config.SectionDecoder) (*Config, error) {
	var c Config
	if err := config.LoadSection(dec, ConfigKey, &c); err != nil {
		return nil, err
	}
	return &c, nil
}
