package outbox

import (
	"context"
	"encoding/json"
	"time"

	"jimu/internal/platform/observability"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var outboxEventsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
	Namespace: "jimu",
	Subsystem: "outbox",
	Name:      "events_total",
	Help:      "Total number of outbox events published",
}, []string{"result"})

// Event 待发布事件
type Event struct {
	ID          uint64          `gorm:"primaryKey" json:"id"`
	AggregateID string          `gorm:"size:128;index" json:"aggregate_id"`  // 聚合根 ID
	EventType   string          `gorm:"size:128;index" json:"event_type"`    // 事件类型
	Payload     json.RawMessage `gorm:"type:json" json:"payload"`            // 事件数据
	Metadata    json.RawMessage `gorm:"type:json" json:"metadata,omitempty"` // 元数据
	CreatedAt   time.Time       `json:"created_at"`
	PublishedAt *time.Time      `json:"published_at,omitempty"` // 发布时间（NULL 表示未发布）
	RetryCount  int             `json:"retry_count"`            // 重试次数
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

// Add 记录业务事件到 Outbox（供业务模块调用）。
// 将调用方 ctx 的 trace 上下文注入事件 Metadata（traceparent/tracestate），
// 供发布时跨服务透传；已有同名 key 不覆盖。
func (o *Outbox) Add(ctx context.Context, tx interface{}, events ...Event) error {
	if len(events) == 0 {
		return nil
	}
	injected := make([]Event, len(events))
	copy(injected, events)
	tp, ts := observability.TraceFromContext(ctx)
	if tp != "" || ts != "" {
		for i := range injected {
			mergeTrace(&injected[i], tp, ts)
		}
	}
	return o.store.Add(ctx, tx, injected...)
}

// mergeTrace 将 traceparent/tracestate 合并进事件 Metadata（保留原有字段）。
func mergeTrace(e *Event, traceparent, tracestate string) {
	meta := map[string]interface{}{}
	if len(e.Metadata) > 0 {
		if err := json.Unmarshal(e.Metadata, &meta); err != nil {
			meta = map[string]interface{}{}
		}
	}
	if _, ok := meta["traceparent"]; !ok && traceparent != "" {
		meta["traceparent"] = traceparent
	}
	if _, ok := meta["tracestate"]; !ok && tracestate != "" {
		meta["tracestate"] = tracestate
	}
	b, err := json.Marshal(meta)
	if err != nil {
		return
	}
	e.Metadata = b
}

// Store 返回底层存储（供需要直接操作时使用）
func (o *Outbox) Store() Store {
	return o.store
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
		outboxEventsTotal.WithLabelValues("failed").Add(float64(len(events)))
		return 0, err
	}
	outboxEventsTotal.WithLabelValues("published").Add(float64(len(events)))

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
