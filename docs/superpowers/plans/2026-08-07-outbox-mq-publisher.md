# Outbox → MQ Publisher Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 新增 `MQPublisher`，让 Outbox 事件跨服务分发（发布到 `queue.Queue`），保留 `EventBusPublisher` 作为进程内快速通道，配置驱动切换。

**Architecture:** 复用 `Publisher` 接口与 `Outbox.Process` 调度（已存在，由 scheduler `outbox_process` 每 10s 驱动）。新增 `MQPublisher` 实现 `Publisher`，将事件序列化后提交到 `queue.Queue`。装配时按配置选 publisher：内存场景用 `EventBusPublisher`，跨服务场景用 `MQPublisher`。

**Tech Stack:** Go 1.26.5, 现有 queue 包（Task 1 的 `Queue` 接口）

## Global Constraints

- 模块：`jimu`
- 依赖 `internal/platform/queue` 的 `Queue` 接口（来自 MQ abstraction plan）
- 不改变 `Outbox.Process` 与 `Store` 接口
- 新增错误处理：发布失败时 `MarkFailed` 递增重试计数（已有逻辑）
- 遵循现有代码风格：中文注释、导出符号 doc 注释

---

### Task 1: 新增 MQPublisher

**Files:**
- Create: `internal/platform/outbox/mq_publisher.go`
- Create: `internal/platform/outbox/mq_publisher_test.go`

**Interfaces:**
- Consumes: `Publisher` 接口（`internal/platform/outbox/outbox.go:39`）、`queue.Queue` 接口（`internal/platform/queue/queue.go`）、`Event` struct
- Produces: `NewMQPublisher(queue queue.Queue) *MQPublisher`，实现 `Publisher.Publish`

- [ ] **Step 1: 写失败测试**

```go
// internal/platform/outbox/mq_publisher_test.go
package outbox

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

// fakeQueue 内存假队列，验证 MQPublisher 发布语义
type fakeQueue struct {
	submitted []*jobPayload
}

type jobPayload struct {
	ID      uint64 `json:"id"`
	Type    string `json:"type"`
	Payload string `json:"payload"`
}

func (f *fakeQueue) Submit(ctx context.Context, job interface{ MarshalJSON() ([]byte, error) }) error {
	return nil
}
```

- [ ] **Step 2: 运行测试验证失败**

Run: `go test ./internal/platform/outbox/ -run TestMQPublisher -v`
Expected: 编译失败，`MQPublisher`/`NewMQPublisher` 未定义

- [ ] **Step 3: 写 `mq_publisher.go`**

```go
// internal/platform/outbox/mq_publisher.go
package outbox

import (
	"context"
	"encoding/json"
	"fmt"

	"jimu/internal/platform/queue"
)

// MQPublisher 发布 Outbox 事件到消息队列，支持跨服务分发
type MQPublisher struct {
	queue queue.Queue
}

// NewMQPublisher 创建 MQ 发布器
func NewMQPublisher(q queue.Queue) *MQPublisher {
	return &MQPublisher{queue: q}
}

// Publish 发布事件到消息队列
func (p *MQPublisher) Publish(ctx context.Context, events ...Event) error {
	for _, e := range events {
		payload, err := json.Marshal(e)
		if err != nil {
			return fmt.Errorf("marshal outbox event %d: %w", e.ID, err)
		}
		job := &queue.JobData{
			ID:      e.ID,
			Type:    "outbox:" + e.EventType,
			Payload: string(payload),
		}
		if err := p.queue.Submit(ctx, job); err != nil {
			return fmt.Errorf("submit outbox event %d: %w", e.ID, err)
		}
	}
	return nil
}

var _ Publisher = (*MQPublisher)(nil)
```

- [ ] **Step 4: 运行测试验证通过**

Run: `go test ./internal/platform/outbox/ -run TestMQPublisher -v`
Expected: PASS

- [ ] **Step 5: 提交**

```bash
git add internal/platform/outbox/mq_publisher.go internal/platform/outbox/mq_publisher_test.go
git commit -m "feat(outbox): add MQ publisher for cross-service events"
```

---

### Task 2: 装配 MQPublisher

**Files:**
- Modify: `internal/app/container.go`
- Modify: `internal/config/config.go`
- Modify: `internal/config/validate.go`
- Modify: `configs/app.yaml`
- Modify: `configs/app.prod.yaml`
- Modify: `README.md`

**Interfaces:**
- Consumes: `MQPublisher`（Task 1）、`queue.New`/`queue.Config`（MQ abstraction plan Task 4）、`outbox.New`
- Produces: `container.go` 按配置选择 publisher；新增配置 `outbox.publisher` 枚举（`event_bus`/`mq`）

- [ ] **Step 1: 修改 config 加枚举**

在 `internal/config/config.go` 加常量：

```go
// Outbox 发布器类型
const (
	OutboxPublisherEventBus = "event_bus"
	OutboxPublisherMQ       = "mq"
)
```

加配置结构：

```go
// OutboxConfig Outbox 配置
type OutboxConfig struct {
	Publisher string `mapstructure:"publisher"` // event_bus / mq
}
```

在 `Config` struct 加：`Outbox OutboxConfig \`mapstructure:"outbox"\``

在 `internal/config/validate.go` `validateCommon()` 加：

```go
if !contains(validOutboxPublishers, c.Outbox.Publisher) {
	return fmt.Errorf("invalid outbox.publisher: %q, must be one of %v", c.Outbox.Publisher, validOutboxPublishers)
}
```

config.go 定义：

```go
var validOutboxPublishers = []string{OutboxPublisherEventBus, OutboxPublisherMQ}
```

- [ ] **Step 2: 更新配置文件**

`configs/app.yaml` 与 `configs/app.prod.yaml` 加：

```yaml
outbox:
  publisher: event_bus
```

- [ ] **Step 3: 修改 container.go 装配**

替换 `internal/app/container.go:118-120`：

```go
	// Outbox
	outboxStore := outbox.NewMySQLStore(dbConn)
	var outboxPublisher outbox.Publisher
	switch cfg.Outbox.Publisher {
	case config.OutboxPublisherMQ:
		q, err := queue.New(queue.Config{
			Type:    queue.Type(cfg.Queue.Type),
			Redis:   rdb,
			Kafka:   queue.KafkaConfig{},
			RabbitMQ: queue.RabbitMQConfig{},
		})
		if err != nil {
			return nil, fmt.Errorf("init outbox queue: %w", err)
		}
		outboxPublisher = outbox.NewMQPublisher(q)
	default:
		outboxPublisher = outbox.NewEventBusPublisher(eventBus)
	}
	outboxProcessor := outbox.New(outboxStore, outboxPublisher)
```

注意：`cfg.Queue.Type` 需先确认 `QueueConfig` 字段存在（MQ abstraction plan Task 4 已加）。

- [ ] **Step 4: 验证编译**

Run: `go build ./...`
Expected: 编译通过

- [ ] **Step 5: 更新 README.md**

配置表加 `outbox.publisher` 行，特性列表注明 Outbox 支持 MQ 跨服务发布。

- [ ] **Step 6: 提交**

```bash
git add internal/app/container.go internal/config/config.go internal/config/validate.go configs/app.yaml configs/app.prod.yaml README.md
git commit -m "feat(outbox): wire MQ publisher by config"
```

---

### Self-Review 记录

**1. Spec 覆盖：**
- MQPublisher 发布到 Queue → Task 1
- 保留 EventBusPublisher → Task 2 装配
- 配置切换 → Task 2
- Outbox.Process 调度不变（已有 scheduler 驱动）→ 无需改动

**2. Placeholder 扫描：** 无 TBD/TODO。

**3. Type 一致性：**
- `queue.JobData` 引用与 MQ abstraction plan Task 1 定义一致。
- `queue.Config`/`queue.Type` 引用与 MQ abstraction plan Task 4 定义一致。
- `Publisher` 接口与现有 outbox.go 一致。
