package outbox

import (
	"context"
	"encoding/json"
	"time"
)

// Event 待发布事件
type Event struct {
	ID          uint64          `gorm:"primaryKey" json:"id"`
	AggregateID string          `gorm:"size:128;index" json:"aggregate_id"` // 聚合根 ID
	EventType   string          `gorm:"size:128;index" json:"event_type"`    // 事件类型
	Payload     json.RawMessage `gorm:"type:json" json:"payload"`            // 事件数据
	Metadata    json.RawMessage `gorm:"type:json" json:"metadata,omitempty"` // 元数据
	CreatedAt   time.Time       `json:"created_at"`
	PublishedAt *time.Time      `json:"published_at,omitempty"` // 发布时间（NULL 表示未发布）
	RetryCount  int             `json:"retry_count"`           // 重试次数
}

// TableName 表名
func (Event) TableName() string {
	return "outbox_events"
}

// Store Outbox 存储接口
type Store interface {
	// Add 在事务中添加事件（与业务数据同一事务）
	Add(ctx context.Context, tx interface{}, events ...Event) error

	// FetchUnpublished 获取待发布事件
	FetchUnpublish(ctx context.Context, limit int) ([]Event, error)

	// MarkPublished 标记事件已发布
	MarkPublished(ctx context.Context, ids []uint64) error

	// MarkFailed 标记事件发布失败（增加重试计数）
	MarkFailed(ctx context.Context, id uint64, err error) error
}

// Publisher 事件发布器接口
type Publisher interface {
	// Publish 发布事件到消息队列/事件总线
	Publish(ctx context.Context, events ...Event) error
}

// Outbox Outbox 模式协调器
type Outbox struct {
	store     Store
	publisher Publisher
}

// New 创建 Outbox
func New(store Store, publisher Publisher) *Outbox {
	return &Outbox{
		store:     store,
		publisher: publisher,
	}
}

// Process 处理待发布事件（由定时任务调用）
func (o *Outbox) Process(ctx context.Context, batchSize int) (int, error) {
	events, err := o.store.FetchUnpublish(ctx, batchSize)
	if err != nil {
		return 0, err
	}

	if len(events) == 0 {
		return 0, nil
	}

	// 发布事件
	if err := o.publisher.Publish(ctx, events...); err != nil {
		// 标记失败
		for _, e := range events {
			_ = o.store.MarkFailed(ctx, e.ID, err)
		}
		return 0, err
	}

	// 标记已发布
	ids := make([]uint64, len(events))
	for i, e := range events {
		ids[i] = e.ID
	}
	if err := o.store.MarkPublished(ctx, ids); err != nil {
		return 0, err
	}

	return len(events), nil
}
