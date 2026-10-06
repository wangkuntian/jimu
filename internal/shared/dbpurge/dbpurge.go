package dbpurge

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"gorm.io/gorm"
)

var deletedTotal = promauto.NewCounterVec(prometheus.CounterOpts{
	Namespace: "jimu",
	Subsystem: "retention",
	Name:      "deleted_total",
	Help:      "Total number of rows deleted by retention cleanup",
}, []string{"table"})

const maxBatches = 1000

// Rule describes one caller-owned table purge policy.
type Rule struct {
	Table      string
	Model      any
	TimeColumn string
	Condition  string
	Args       []any
	Days       int
}

// Result reports rows deleted by a table rule.
type Result struct {
	Table   string `json:"table"`
	Deleted int64  `json:"deleted"`
}

// Service applies caller-provided purge rules with bounded delete batches.
type Service struct {
	db        *gorm.DB
	batchSize int
}

// New creates a purge service. A non-positive batch size uses 500 rows.
func New(db *gorm.DB, batchSize int) *Service {
	if batchSize <= 0 {
		batchSize = 500
	}
	return &Service{db: db, batchSize: batchSize}
}

// Run purges expired rows for enabled rules. Days <= 0 skips the rule.
func (s *Service) Run(ctx context.Context, rules []Rule) ([]Result, error) {
	if s.db == nil {
		return nil, errors.New("dbpurge: database is required")
	}
	now := time.Now()
	results := make([]Result, 0, len(rules))
	var errs []error
	for _, rule := range rules {
		if rule.Days <= 0 {
			continue
		}
		cutoff := now.AddDate(0, 0, -rule.Days)
		deleted, err := s.purge(ctx, rule, cutoff)
		RecordDeleted(rule.Table, deleted)
		if err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", rule.Table, err))
			continue
		}
		results = append(results, Result{Table: rule.Table, Deleted: deleted})
	}
	return results, errors.Join(errs...)
}

// RecordDeleted adds rows purged by callers that still own their scheduling and rules.
func RecordDeleted(table string, deleted int64) {
	if deleted > 0 {
		deletedTotal.WithLabelValues(table).Add(float64(deleted))
	}
}

func (s *Service) purge(ctx context.Context, rule Rule, cutoff time.Time) (int64, error) {
	var total int64
	for i := 0; i < maxBatches; i++ {
		sub := s.db.WithContext(ctx).Unscoped().Model(rule.Model).
			Select("id").
			Where(rule.TimeColumn+" < ?", cutoff)
		if rule.Condition != "" {
			sub = sub.Where(rule.Condition, rule.Args...)
		}
		sub = sub.Limit(s.batchSize)

		// MySQL/MariaDB do not support LIMIT inside IN subqueries; the derived table works across supported dialects.
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
