package db

import (
	"context"
	"errors"
	"fmt"
	"time"

	"jimu/internal/config"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"gorm.io/gorm"
)

var retentionDeletedTotal = promauto.NewCounterVec(prometheus.CounterOpts{
	Namespace: "jimu",
	Subsystem: "retention",
	Name:      "deleted_total",
	Help:      "Total number of rows deleted by the retention job",
}, []string{"table"})

const (
	defaultRetentionBatchSize = 500
	maxRetentionBatches       = 1000 // 单表单次运行的批次上限，避免误配置导致长时间空转
)

// 以下为各历史表的最小模型：只声明清理所需字段与表名。
// 刻意不依赖业务模块的领域模型（platform 不应反向依赖 modules）。
type (
	auditLogRow struct {
		ID        uint64    `gorm:"primaryKey"`
		CreatedAt time.Time `gorm:"column:created_at"`
	}
	jobRow struct {
		ID        uint64    `gorm:"primaryKey"`
		UpdatedAt time.Time `gorm:"column:updated_at"`
	}
	jobHistoryRow struct {
		ID      uint64    `gorm:"primaryKey"`
		EndedAt time.Time `gorm:"column:ended_at"`
	}
	deadLetterRow struct {
		ID         uint64    `gorm:"primaryKey"`
		ResolvedAt time.Time `gorm:"column:resolved_at"`
	}
	outboxEventRow struct {
		ID          uint64     `gorm:"primaryKey"`
		PublishedAt *time.Time `gorm:"column:published_at"`
	}
	importJobRow struct {
		ID        uint64    `gorm:"primaryKey"`
		CreatedAt time.Time `gorm:"column:created_at"`
	}
	trustedDeviceRow struct {
		ID        uint64    `gorm:"primaryKey"`
		ExpiresAt time.Time `gorm:"column:expires_at"`
	}
)

func (auditLogRow) TableName() string      { return "audit_logs" }
func (jobRow) TableName() string           { return "jobs" }
func (jobHistoryRow) TableName() string    { return "job_history" }
func (deadLetterRow) TableName() string    { return "dead_letters" }
func (outboxEventRow) TableName() string   { return "outbox_events" }
func (importJobRow) TableName() string     { return "import_jobs" }
func (trustedDeviceRow) TableName() string { return "trusted_devices" }

// RetentionRule 单表保留策略
type RetentionRule struct {
	Table      string      // 表名（日志与指标标签）
	Model      interface{} // GORM 模型（提供表名）
	TimeColumn string      // 判定时间列
	Condition  string      // 附加条件（可空），只清理该范围内的行
	Days       int         // 保留天数；<=0 表示不清理该表
}

// DefaultRetentionRules 依据配置生成保留策略，Days<=0 的表自动跳过。
// 条件用方言中立写法（TRUE / IN / IS NOT NULL），MySQL 与 PostgreSQL 通用。
func DefaultRetentionRules(cfg config.RetentionConfig) []RetentionRule {
	all := []RetentionRule{
		{Table: "audit_logs", Model: &auditLogRow{}, TimeColumn: "created_at", Days: cfg.AuditLogDays},
		{Table: "jobs", Model: &jobRow{}, TimeColumn: "updated_at", Condition: "status IN ('success', 'dead')", Days: cfg.JobDays},
		{Table: "job_history", Model: &jobHistoryRow{}, TimeColumn: "ended_at", Days: cfg.JobHistoryDays},
		{Table: "dead_letters", Model: &deadLetterRow{}, TimeColumn: "resolved_at", Condition: "resolved = TRUE", Days: cfg.DeadLetterDays},
		{Table: "outbox_events", Model: &outboxEventRow{}, TimeColumn: "published_at", Condition: "published_at IS NOT NULL", Days: cfg.OutboxEventDays},
		{Table: "import_jobs", Model: &importJobRow{}, TimeColumn: "created_at", Condition: "status IN ('completed', 'failed')", Days: cfg.ImportJobDays},
		{Table: "trusted_devices", Model: &trustedDeviceRow{}, TimeColumn: "expires_at", Days: cfg.TrustedDeviceDays},
	}
	rules := make([]RetentionRule, 0, len(all))
	for _, r := range all {
		if r.Days > 0 {
			rules = append(rules, r)
		}
	}
	return rules
}

// RetentionResult 单表清理结果
type RetentionResult struct {
	Table   string `json:"table"`
	Deleted int64  `json:"deleted"`
}

// RetentionService 历史数据保留服务：按规则分批硬删除过期行，避免审计/任务/事件表无限增长。
type RetentionService struct {
	db        *gorm.DB
	rules     []RetentionRule
	batchSize int
}

// NewRetentionService 创建保留服务（rules 为 nil 时按配置生成默认规则）
func NewRetentionService(db *gorm.DB, cfg config.RetentionConfig) *RetentionService {
	batch := cfg.BatchSize
	if batch <= 0 {
		batch = defaultRetentionBatchSize
	}
	return &RetentionService{db: db, rules: DefaultRetentionRules(cfg), batchSize: batch}
}

// NewRetentionServiceWithRules 使用自定义规则创建保留服务（业务自有表可复用同一套分批清理逻辑）
func NewRetentionServiceWithRules(db *gorm.DB, rules []RetentionRule, batchSize int) *RetentionService {
	if batchSize <= 0 {
		batchSize = defaultRetentionBatchSize
	}
	return &RetentionService{db: db, rules: rules, batchSize: batchSize}
}

// Run 执行全部规则；单表失败不阻断其他表，错误聚合返回
func (s *RetentionService) Run(ctx context.Context) ([]RetentionResult, error) {
	if s.db == nil {
		return nil, errors.New("db: retention requires a database")
	}
	now := time.Now()
	results := make([]RetentionResult, 0, len(s.rules))
	var errs []error

	for _, rule := range s.rules {
		if rule.Days <= 0 {
			continue
		}
		cutoff := now.AddDate(0, 0, -rule.Days)
		deleted, err := s.purge(ctx, rule, cutoff)
		if deleted > 0 {
			retentionDeletedTotal.WithLabelValues(rule.Table).Add(float64(deleted))
		}
		if err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", rule.Table, err))
			continue
		}
		results = append(results, RetentionResult{Table: rule.Table, Deleted: deleted})
	}
	return results, errors.Join(errs...)
}

// purge 分批硬删除过期行（Unscoped 绕过软删除，软删数据也一并清理）。
// 子查询必须包一层派生表：MySQL/MariaDB 不支持 IN (SELECT ... LIMIT n)。
func (s *RetentionService) purge(ctx context.Context, rule RetentionRule, cutoff time.Time) (int64, error) {
	var total int64
	for i := 0; i < maxRetentionBatches; i++ {
		sub := s.db.WithContext(ctx).Unscoped().Model(rule.Model).
			Select("id").
			Where(rule.TimeColumn+" < ?", cutoff)
		if rule.Condition != "" {
			sub = sub.Where(rule.Condition)
		}
		sub = sub.Limit(s.batchSize)

		// MySQL/MariaDB 不支持 `IN (SELECT ... LIMIT n)`（错误 1235），需再包一层派生表
		derived := s.db.WithContext(ctx).Table("(?) AS batch", sub).Select("id")
		res := s.db.WithContext(ctx).Unscoped().Where("id IN (?)", derived).Delete(rule.Model)
		if res.Error != nil {
			return total, res.Error
		}
		total += res.RowsAffected
		if res.RowsAffected < int64(s.batchSize) {
			break
		}
	}
	return total, nil
}
