package outbox

import (
	"context"
	"time"

	"gorm.io/gorm"
)

// MySQLStore 基于 MySQL 的 Outbox 存储
type MySQLStore struct {
	db *gorm.DB
}

// NewMySQLStore 创建 MySQL Outbox 存储
func NewMySQLStore(db *gorm.DB) *MySQLStore {
	return &MySQLStore{db: db}
}

// Add 在事务中添加事件。
// tx 为 nil 时使用 store 自身连接独立落库（业务事务已提交的场景，可靠性优先于原子性）；
// 业务事务内调用时传入 *gorm.DB 保证事件与业务同事务。
func (s *MySQLStore) Add(ctx context.Context, tx interface{}, events ...Event) error {
	db := s.db
	if t, ok := tx.(*gorm.DB); ok && t != nil {
		db = t
	}

	now := time.Now()
	for i := range events {
		events[i].CreatedAt = now
		events[i].RetryCount = 0
	}

	return db.WithContext(ctx).Create(&events).Error
}

// FetchUnpublish 获取待发布事件（按创建时间排序）
func (s *MySQLStore) FetchUnpublish(ctx context.Context, limit int) ([]Event, error) {
	var events []Event
	err := s.db.WithContext(ctx).
		Where("published_at IS NULL").
		Order("created_at ASC").
		Limit(limit).
		Find(&events).Error
	return events, err
}

// MarkPublished 标记事件已发布
func (s *MySQLStore) MarkPublished(ctx context.Context, ids []uint64) error {
	if len(ids) == 0 {
		return nil
	}
	now := time.Now()
	return s.db.WithContext(ctx).
		Model(&Event{}).
		Where("id IN ?", ids).
		Update("published_at", now).Error
}

// MarkFailed 标记事件发布失败
func (s *MySQLStore) MarkFailed(ctx context.Context, id uint64, publishErr error) error {
	return s.db.WithContext(ctx).
		Model(&Event{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"retry_count": gorm.Expr("retry_count + 1"),
		}).Error
}

var _ Store = (*MySQLStore)(nil)
