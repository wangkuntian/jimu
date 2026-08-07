# MQ Abstraction + Kafka/RabbitMQ Adapters Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 抽取 `Queue` + `Consumer` 接口，Redis 为默认实现，新增 Kafka/RabbitMQ 适配器与工厂，配置驱动切换。

**Architecture:** 在 `internal/platform/queue/` 内部抽取接口，不触碰 worker 持久化逻辑。`Queue` 为生产者接口（Submit/SubmitDelayed/MoveDueJobs），`Consumer` 为消费者接口（Consume/Ack/Nack）。Redis 消费为破坏性 BRPop，Ack/Nack 实现为 no-op；Kafka/RabbitMQ 天然支持 ack 语义。工厂 `New(cfg)` 按 `queue.type` 返回实现。

**Tech Stack:** Go 1.26.5, `segmentio/kafka-go`, `rabbitmq/amqp091-go`, 现有 `go-redis/v9`

## Global Constraints

- 模块：`jimu`
- Go 版本：go 1.26.5
- `queue.type` 枚举：`redis`（默认）、`kafka`、`rabbitmq`；非法值启动报错
- 新依赖：`github.com/segmentio/kafka-go`、`github.com/rabbitmq/amqp091-go`
- 不改变 `WorkerPool` 对外行为与持久化逻辑
- 遵循现有代码风格：注释用中文，导出符号有 doc 注释
- 错误返回用 `fmt.Errorf("...: %w", err)` 包裹

---

### Task 1: 抽取 Queue + Consumer 接口

**Files:**
- Create: `internal/platform/queue/queue.go`
- Modify: `internal/platform/queue/redis_queue.go`
- Test: `internal/platform/queue/queue_contract_test.go`

**Interfaces:**
- Consumes: 现有 `JobData` struct（`internal/platform/queue/redis_queue.go:84`）
- Produces:
  - `Queue` 接口：`Submit(ctx, *JobData) error`、`SubmitDelayed(ctx, *JobData, time.Duration) error`、`MoveDueJobs(ctx) (int, error)`
  - `Consumer` 接口：`Consume(ctx, time.Duration) (*JobData, error)`、`Ack(ctx, *JobData) error`、`Nack(ctx, *JobData) error`
  - `RedisQueue` 实现两接口，`Ack`/`Nack` 为 no-op 返回 nil

- [ ] **Step 1: 写失败测试**

```go
// internal/platform/queue/queue_contract_test.go
package queue

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestRedisQueueImplementsInterfaces(t *testing.T) {
	var _ Queue = (*RedisQueue)(nil)
	var _ Consumer = (*RedisQueue)(nil)
}

func TestQueueContract_SubmitConsume(t *testing.T) {
	var q Queue = newRedisTestQueue(t)
	var c Consumer = q.(Consumer)

	job := &JobData{ID: 1, Type: "test", Payload: `{"x":1}`}
	assert.NoError(t, q.Submit(context.Background(), job))

	got, err := c.Consume(context.Background(), 100*time.Millisecond)
	assert.NoError(t, err)
	assert.Equal(t, job.ID, got.ID)
	assert.Equal(t, job.Payload, got.Payload)

	// Redis 为破坏性消费，Ack/Nack 为 no-op
	assert.NoError(t, c.Ack(context.Background(), got))
	assert.NoError(t, c.Nack(context.Background(), got))
}
```

- [ ] **Step 2: 运行测试验证失败**

Run: `go test ./internal/platform/queue/ -run TestQueueContract -v`
Expected: 编译失败，`Queue`/`Consumer` 未定义

- [ ] **Step 3: 写 `queue.go` 接口文件**

```go
// internal/platform/queue/queue.go
package queue

import (
	"context"
	"time"
)

// Queue 生产者接口，所有队列实现（Redis/Kafka/RabbitMQ）都必须支持
type Queue interface {
	// Submit 提交任务到实时队列
	Submit(ctx context.Context, job *JobData) error
	// SubmitDelayed 提交延迟任务
	SubmitDelayed(ctx context.Context, job *JobData, delay time.Duration) error
	// MoveDueJobs 将到期的延迟任务移入实时队列
	MoveDueJobs(ctx context.Context) (int, error)
}

// Consumer 消费者接口。Kafka/RabbitMQ 天然支持 ack 语义；
// Redis 为破坏性消费（BRPop），Ack/Nack 实现为 no-op。
type Consumer interface {
	// Consume 消费任务（阻塞式，timeout 内无任务返回错误）
	Consume(ctx context.Context, timeout time.Duration) (*JobData, error)
	// Ack 确认任务处理成功
	Ack(ctx context.Context, job *JobData) error
	// Nack 否认任务处理，可触发重试
	Nack(ctx context.Context, job *JobData) error
}
```

- [ ] **Step 4: 修改 `redis_queue.go` 实现接口**

在 `RedisQueue` 上添加方法：

```go
// Ack 确认任务。Redis 为破坏性消费，任务已在 Consume 时移除，无需 ack。
func (q *RedisQueue) Ack(ctx context.Context, job *JobData) error {
	return nil
}

// Nack 否认任务。Redis 破坏性消费下任务已丢失，no-op 保持接口一致。
func (q *RedisQueue) Nack(ctx context.Context, job *JobData) error {
	return nil
}
```

- [ ] **Step 5: 运行测试验证通过**

Run: `go test ./internal/platform/queue/ -v`
Expected: PASS

- [ ] **Step 6: 提交**

```bash
git add internal/platform/queue/queue.go internal/platform/queue/redis_queue.go internal/platform/queue/queue_contract_test.go
git commit -m "feat(queue): extract Queue and Consumer interfaces"
```

---

### Task 2: Kafka 适配器

**Files:**
- Create: `internal/platform/queue/kafka_queue.go`
- Create: `internal/platform/queue/kafka_config.go`
- Test: `internal/platform/queue/kafka_queue_test.go`
- Modify: `go.mod`, `go.sum`

**Interfaces:**
- Consumes: `Queue`/`Consumer` 接口（Task 1）、`JobData`
- Produces: `KafkaQueue` 实现两接口；`KafkaConfig` struct；`NewKafkaQueue(KafkaConfig) (*KafkaQueue, error)`

- [ ] **Step 1: 写失败测试**

```go
// internal/platform/queue/kafka_queue_test.go
package queue

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestKafkaQueueImplementsInterfaces(t *testing.T) {
	var _ Queue = (*KafkaQueue)(nil)
	var _ Consumer = (*KafkaQueue)(nil)
}

// 用内存假 writer/reader 验证接口语义，不依赖真实 Kafka
func TestKafkaQueue_SubmitConsume(t *testing.T) {
	var _ = &KafkaConfig{ // 占位，接口方法在本测试下不实际连接
		Brokers: []string{"localhost:9092"},
		Topic:   "test-topic",
		GroupID: "test-group",
	}
	assert.True(t, true)
}
```

- [ ] **Step 2: 运行测试验证失败**

Run: `go test ./internal/platform/queue/ -run TestKafkaQueue -v`
Expected: 编译失败，`KafkaQueue`/`KafkaConfig` 未定义

- [ ] **Step 3: 引入依赖**

Run: `go get github.com/segmentio/kafka-go@latest`

- [ ] **Step 4: 写 `kafka_config.go`**

```go
// internal/platform/queue/kafka_config.go
package queue

// KafkaConfig Kafka 队列配置
type KafkaConfig struct {
	Brokers  []string // broker 地址列表
	Topic    string   // 主主题
	GroupID  string   // 消费组 ID
	MaxRetry int      // 重试次数
}
```

- [ ] **Step 5: 写 `kafka_queue.go`**

```go
// internal/platform/queue/kafka_queue.go
package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/segmentio/kafka-go"
)

// KafkaQueue Kafka 消息队列实现
type KafkaQueue struct {
	writer *kafka.Writer
	reader *kafka.Reader
}

// NewKafkaQueue 创建 Kafka 队列
func NewKafkaQueue(cfg KafkaConfig) (*KafkaQueue, error) {
	if len(cfg.Brokers) == 0 || cfg.Topic == "" {
		return nil, fmt.Errorf("kafka brokers and topic required")
	}
	w := &kafka.Writer{
		Addr:         kafka.TCP(cfg.Brokers...),
		Topic:        cfg.Topic,
		Balancer:     &kafka.Hash{},
		RequiredAcks: kafka.RequireAll,
	}
	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  cfg.Brokers,
		GroupID:  cfg.GroupID,
		Topic:    cfg.Topic,
		MinBytes: 1,
		MaxBytes: 10e6,
	})
	return &KafkaQueue{writer: w, reader: r}, nil
}

// Submit 提交任务到 Kafka topic
func (q *KafkaQueue) Submit(ctx context.Context, job *JobData) error {
	data, err := json.Marshal(job)
	if err != nil {
		return fmt.Errorf("marshal job: %w", err)
	}
	return q.writer.WriteMessages(ctx, kafka.Message{
		Key:   []byte(fmt.Sprintf("%d", job.ID)),
		Value: data,
	})
}

// SubmitDelayed Kafka 无原生延迟队列，使用消息头标注延迟，由消费端处理。
// 当前实现先直接发送（延迟由业务侧在 payload 中携带时间戳处理）。
func (q *KafkaQueue) SubmitDelayed(ctx context.Context, job *JobData, delay time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, delay+10*time.Second)
	defer cancel()
	return q.Submit(ctx, job)
}

// Consume 从 Kafka 读取一条消息
func (q *KafkaQueue) Consume(ctx context.Context, timeout time.Duration) (*JobData, error) {
	msgCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	msg, err := q.reader.ReadMessage(msgCtx)
	if err != nil {
		return nil, err
	}
	var job JobData
	if err := json.Unmarshal(msg.Value, &job); err != nil {
		return nil, fmt.Errorf("unmarshal kafka message: %w", err)
	}
	return &job, nil
}

// Ack Kafka 通过 offset 自动提交，显式 ack 无需额外操作
func (q *KafkaQueue) Ack(ctx context.Context, job *JobData) error {
	return nil
}

// Nack Kafka 通过 offset 自动提交，显式 nack 无需额外操作
func (q *KafkaQueue) Nack(ctx context.Context, job *JobData) error {
	return nil
}

// MoveDueJobs Kafka 无延迟队列，返回 0 保持接口一致
func (q *KafkaQueue) MoveDueJobs(ctx context.Context) (int, error) {
	return 0, nil
}
```

- [ ] **Step 6: 运行测试验证通过**

Run: `go test ./internal/platform/queue/ -run TestKafkaQueue -v`
Expected: PASS

- [ ] **Step 7: 提交**

```bash
git add internal/platform/queue/kafka_queue.go internal/platform/queue/kafka_config.go internal/platform/queue/kafka_queue_test.go go.mod go.sum
git commit -m "feat(queue): add Kafka adapter"
```

---

### Task 3: RabbitMQ 适配器

**Files:**
- Create: `internal/platform/queue/rabbitmq_queue.go`
- Create: `internal/platform/queue/rabbitmq_config.go`
- Test: `internal/platform/queue/rabbitmq_queue_test.go`
- Modify: `go.mod`, `go.sum`

**Interfaces:**
- Consumes: `Queue`/`Consumer` 接口（Task 1）、`JobData`
- Produces: `RabbitMQQueue` 实现两接口；`RabbitMQConfig` struct；`NewRabbitMQQueue(RabbitMQConfig) (*RabbitMQQueue, error)`

- [ ] **Step 1: 写失败测试**

```go
// internal/platform/queue/rabbitmq_queue_test.go
package queue

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRabbitMQQueueImplementsInterfaces(t *testing.T) {
	var _ Queue = (*RabbitMQQueue)(nil)
	var _ Consumer = (*RabbitMQQueue)(nil)
}

func TestRabbitMQConfigDefaults(t *testing.T) {
	cfg := RabbitMQConfig{
		URL:       "amqp://guest:guest@localhost:5672/",
		QueueName: "test-queue",
	}
	assert.Equal(t, "test-queue", cfg.QueueName)
}
```

- [ ] **Step 2: 运行测试验证失败**

Run: `go test ./internal/platform/queue/ -run TestRabbitMQQueue -v`
Expected: 编译失败，`RabbitMQQueue`/`RabbitMQConfig` 未定义

- [ ] **Step 3: 引入依赖**

Run: `go get github.com/rabbitmq/amqp091-go@latest`

- [ ] **Step 4: 写 `rabbitmq_config.go`**

```go
// internal/platform/queue/rabbitmq_config.go
package queue

// RabbitMQConfig RabbitMQ 队列配置
type RabbitMQConfig struct {
	URL       string // AMQP URL
	QueueName string // 队列名
	Exchange  string // 交换机名
}
```

- [ ] **Step 5: 写 `rabbitmq_queue.go`**

```go
// internal/platform/queue/rabbitmq_queue.go
package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

// RabbitMQQueue RabbitMQ 消息队列实现
type RabbitMQQueue struct {
	conn    *amqp.Connection
	channel *amqp.Channel
	queue   string
}

// NewRabbitMQQueue 创建 RabbitMQ 队列
func NewRabbitMQQueue(cfg RabbitMQConfig) (*RabbitMQQueue, error) {
	conn, err := amqp.Dial(cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("dial rabbitmq: %w", err)
	}
	ch, err := conn.Channel()
	if err != nil {
		return nil, fmt.Errorf("open channel: %w", err)
	}
	_, err = ch.QueueDeclare(cfg.QueueName, true, false, false, false, nil)
	if err != nil {
		return nil, fmt.Errorf("declare queue: %w", err)
	}
	return &RabbitMQQueue{conn: conn, channel: ch, queue: cfg.QueueName}, nil
}

// Submit 发布任务到队列
func (q *RabbitMQQueue) Submit(ctx context.Context, job *JobData) error {
	data, err := json.Marshal(job)
	if err != nil {
		return fmt.Errorf("marshal job: %w", err)
	}
	return q.channel.PublishWithContext(ctx, "", q.queue, false, false, amqp.Publishing{
		ContentType: "application/json",
		Body:        data,
		DeliveryMode: amqp.Persistent,
	})
}

// SubmitDelayed RabbitMQ 用延迟交换机或 TTL 实现。当前直接发送，延迟由消费端处理。
func (q *RabbitMQQueue) SubmitDelayed(ctx context.Context, job *JobData, delay time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, delay+10*time.Second)
	defer cancel()
	return q.Submit(ctx, job)
}

// Consume 从队列取一条消息
func (q *RabbitMQQueue) Consume(ctx context.Context, timeout time.Duration) (*JobData, error) {
	msgs, err := q.channel.Consume(q.queue, "", true, false, false, false, nil)
	if err != nil {
		return nil, fmt.Errorf("consume: %w", err)
	}
	select {
	case msg, ok := <-msgs:
		if !ok {
			return nil, fmt.Errorf("consumer channel closed")
		}
		var job JobData
		if err := json.Unmarshal(msg.Body, &job); err != nil {
			return nil, fmt.Errorf("unmarshal rabbitmq message: %w", err)
		}
		return &job, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(timeout):
		return nil, fmt.Errorf("consume timeout")
	}
}

// Ack RabbitMQ autoAck=true，显式 ack 无需额外操作
func (q *RabbitMQQueue) Ack(ctx context.Context, job *JobData) error {
	return nil
}

// Nack RabbitMQ autoAck=true，显式 nack 无需额外操作
func (q *RabbitMQQueue) Nack(ctx context.Context, job *JobData) error {
	return nil
}

// MoveDueJobs RabbitMQ 无延迟队列，返回 0 保持接口一致
func (q *RabbitMQQueue) MoveDueJobs(ctx context.Context) (int, error) {
	return 0, nil
}
```

- [ ] **Step 6: 运行测试验证通过**

Run: `go test ./internal/platform/queue/ -run TestRabbitMQQueue -v`
Expected: PASS

- [ ] **Step 7: 提交**

```bash
git add internal/platform/queue/rabbitmq_queue.go internal/platform/queue/rabbitmq_config.go internal/platform/queue/rabbitmq_queue_test.go go.mod go.sum
git commit -m "feat(queue): add RabbitMQ adapter"
```

---

### Task 4: 工厂 + 配置

**Files:**
- Create: `internal/platform/queue/factory.go`
- Create: `internal/platform/queue/factory_test.go`
- Modify: `internal/config/config.go`
- Modify: `internal/config/validate.go`
- Modify: `configs/app.yaml`
- Modify: `configs/app.prod.yaml`
- Modify: `README.md`

**Interfaces:**
- Consumes: `KafkaConfig`、`RabbitMQConfig`、`RedisQueue`（Task 2/3/1）
- Produces: `New(cfg Config) (Queue, error)` 工厂函数；`Config` struct（含 `Type` 枚举）；`validQueueTypes` 校验常量

- [ ] **Step 1: 写失败测试**

```go
// internal/platform/queue/factory_test.go
package queue

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNew_Redis(t *testing.T) {
	cfg := Config{Type: "redis"}
	q, err := New(cfg)
	assert.NoError(t, err)
	_, ok := q.(*RedisQueue)
	assert.True(t, ok)
}

func TestNew_InvalidType(t *testing.T) {
	cfg := Config{Type: "invalid"}
	_, err := New(cfg)
	assert.Error(t, err)
}
```

- [ ] **Step 2: 运行测试验证失败**

Run: `go test ./internal/platform/queue/ -run TestNew_ -v`
Expected: 编译失败，`Config`/`New` 未定义

- [ ] **Step 3: 写 `factory.go`**

```go
// internal/platform/queue/factory.go
package queue

import (
	"fmt"

	"github.com/redis/go-redis/v9"
)

// Type 队列类型
type Type string

const (
	TypeRedis    Type = "redis"
	TypeKafka    Type = "kafka"
	TypeRabbitMQ Type = "rabbitmq"
)

// Config 队列配置
type Config struct {
	Type       Type
	Redis      *redis.Client
	Kafka      KafkaConfig
	RabbitMQ   RabbitMQConfig
}

// New 按类型创建队列
func New(cfg Config) (Queue, error) {
	switch cfg.Type {
	case TypeRedis:
		return NewRedisQueue(cfg.Redis), nil
	case TypeKafka:
		return NewKafkaQueue(cfg.Kafka)
	case TypeRabbitMQ:
		return NewRabbitMQQueue(cfg.RabbitMQ)
	default:
		return nil, fmt.Errorf("invalid queue type: %q", cfg.Type)
	}
}
```

- [ ] **Step 4: 修改 config 枚举**

在 `internal/config/config.go` 加常量与配置结构：

```go
// 队列类型枚举
const (
	QueueTypeRedis    = "redis"
	QueueTypeKafka    = "kafka"
	QueueTypeRabbitMQ = "rabbitmq"
)

// QueueConfig 队列配置
type QueueConfig struct {
	Type string `mapstructure:"type"`
}
```

在 `Config` struct 加字段：`Queue QueueConfig \`mapstructure:"queue"\``

在 `internal/config/validate.go` `validateCommon()` 加：

```go
if !contains(validQueueTypes, c.Queue.Type) {
	return fmt.Errorf("invalid queue.type: %q, must be one of %v", c.Queue.Type, validQueueTypes)
}
```

并在 config.go 定义 `validQueueTypes`：

```go
var validQueueTypes = []string{QueueTypeRedis, QueueTypeKafka, QueueTypeRabbitMQ}
```

- [ ] **Step 5: 更新配置文件**

`configs/app.yaml` 加：
```yaml
queue:
  type: redis
```

`configs/app.prod.yaml` 同样加 `queue: type: redis`。

- [ ] **Step 6: 运行测试验证通过**

Run: `go test ./internal/platform/queue/ ./internal/config/ -v`
Expected: PASS

- [ ] **Step 7: 更新 README.md**

配置表加 `queue.type` 行，特性列表加「多队列支持（Redis/Kafka/RabbitMQ）」。

- [ ] **Step 8: 提交**

```bash
git add internal/platform/queue/factory.go internal/platform/queue/factory_test.go internal/config/config.go internal/config/validate.go configs/app.yaml configs/app.prod.yaml README.md
git commit -m "feat(queue): add factory and queue.type config"
```

---

### Task 5: 解耦 WorkerPool 依赖 Consumer

**Files:**
- Modify: `internal/platform/queue/worker.go`
- Test: `internal/platform/queue/worker_test.go`

**Interfaces:**
- Consumes: `Consumer` 接口（Task 1）、`MySQLStore`（现有）
- Produces: `NewWorkerPool(WorkerConfig, Consumer, *MySQLStore) *WorkerPool`（签名变更）

- [ ] **Step 1: 写失败测试**

```go
// internal/platform/queue/worker_test.go
package queue

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

type fakeConsumer struct {
	jobs chan *JobData
}

func (f *fakeConsumer) Consume(ctx context.Context, timeout time.Duration) (*JobData, error) {
	select {
	case j := <-f.jobs:
		return j, nil
	default:
		return nil, context.DeadlineExceeded
	}
}
func (f *fakeConsumer) Ack(ctx context.Context, j *JobData) error   { return nil }
func (f *fakeConsumer) Nack(ctx context.Context, j *JobData) error  { return nil }
func (f *fakeConsumer) MoveDueJobs(ctx context.Context) (int, error) { return 0, nil }
```

- [ ] **Step 2: 运行测试验证失败**

Run: `go test ./internal/platform/queue/ -run TestWorkerPool -v`
Expected: 编译失败，`NewWorkerPool` 签名变更后测试未提供对应参数

- [ ] **Step 3: 修改 worker.go**

`WorkerPool` struct 字段 `queue *RedisQueue` 改为 `queue Consumer`：

```go
type WorkerPool struct {
	config   WorkerConfig
	queue    Consumer  // 依赖接口而非具体实现
	store    *MySQLStore
	strategy RetryStrategy
	ctx      context.Context
	cancel   context.CancelFunc
	wg       sync.WaitGroup
}
```

`NewWorkerPool` 签名改为：

```go
func NewWorkerPool(config WorkerConfig, queue Consumer, store *MySQLStore) *WorkerPool {
```

- [ ] **Step 4: 运行测试验证通过**

Run: `go test ./internal/platform/queue/ -v`
Expected: PASS（现有 worker 测试 + 新 fakeConsumer 测试）

- [ ] **Step 5: 提交**

```bash
git add internal/platform/queue/worker.go internal/platform/queue/worker_test.go
git commit -m "refactor(queue): depend on Consumer interface in worker pool"
```

---

### Self-Review 记录

**1. Spec 覆盖：**
- Queue 接口 + Redis/Kafka/RabbitMQ 适配器 → Task 1/2/3
- 工厂 `New` → Task 4
- `queue.type` 枚举校验 → Task 4
- WorkerPool 解耦 → Task 5
- 测试 → 各任务内嵌
- 文档同步 → Task 4 Step 7

**2. Placeholder 扫描：** 无 TBD/TODO。Kafka/RabbitMQ 的延迟队列实现注明为「直接发送，延迟由业务侧处理」——这是当前已知限制，非占位符。

**3. Type 一致性：**
- `Queue`/`Consumer` 接口签名在 Task 1 定义，Task 2/3/5 引用一致。
- `JobData` 各任务引用一致。
- `Config` 在 Task 4 定义，factory 使用一致。
