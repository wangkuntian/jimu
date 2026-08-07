# Missing Backend Capabilities Design

**Date:** 2026-08-07
**Status:** Approved

## Overview

为 jimu 补齐「通用 Go 后端框架」在生产复用 + 开源开箱即用两个维度上的关键缺口。分五个方向推进，按依赖顺序实施：

1. 消息队列抽象 + Kafka/RabbitMQ 适配器
2. 事件总线升级（Outbox → 消息队列，支持跨服务事件分发）
3. OAuth2/OIDC 第三方登录
4. 验证码（图形验证码，防刷）
5. Scheduler 持久化（MySQL 存储 + 多实例协调）

## 实施顺序与依赖

```
1. 消息队列抽象 + 适配器
      ↓ outbox publisher 重定向到 MQ
2. 事件总线升级（跨服务事件分发）
3. OAuth2/OIDC     ← 独立，可并行
4. 验证码          ← 独立，可并行
5. Scheduler 持久化 ← 独立，可并行
```

依赖关系：
- 1 → 2：Outbox 要跨服务分发，必须先有 MQ 抽象。
- 3、4、5 相互独立，可与 1/2 并行。

## 一、消息队列抽象 + 适配器

### 现状

`internal/platform/queue/redis_queue.go` 的 `RedisQueue` 绑定 Redis 具体实现，无接口。`contract.Module` 无队列注册能力。当前已有 queue 能力：Redis 实时队列（LPUSH/BRPOP）、延迟队列（ZSET）、worker 池、重试策略、死信队列、MySQL 持久化。

### 设计

抽取 `Queue` 接口，Redis 为默认实现，Kafka/RabbitMQ 为可选适配器。

```go
// internal/platform/queue/queue.go
type Queue interface {
    Submit(ctx context.Context, job *JobData) error
    SubmitDelayed(ctx context.Context, job *JobData, delay time.Duration) error
    Consume(ctx context.Context, timeout time.Duration) (*JobData, error)
    Ack(ctx context.Context, job *JobData) error
    Nack(ctx context.Context, job *JobData) error
}
```

- `RedisQueue` 实现该接口，保持现有行为。
- 新增 `KafkaQueue`、`RabbitMQQueue` 适配器，实现同一接口。
- 配置 `queue.type` 枚举（`redis`/`kafka`/`rabbitmq`），启动校验。
- 工厂 `queue.New(cfg)` 按类型返回实现。

### 变更文件

| 文件 | 动作 |
|------|------|
| `internal/platform/queue/queue.go` | 新增接口 |
| `internal/platform/queue/redis_queue.go` | 实现接口 |
| `internal/platform/queue/kafka_queue.go` | 新增适配器 |
| `internal/platform/queue/rabbitmq_queue.go` | 新增适配器 |
| `internal/platform/queue/factory.go` | 新增工厂 |
| `internal/config/config.go` | 加 `queue.type` 枚举校验 |
| `internal/config/validate.go` | 加枚举常量 |

### 测试

- `kafka_queue_test.go`、`rabbitmq_queue_test.go`：用内存假实现验证接口语义。
- 复用现有 `queue` 包 worker 测试，确认接口替换无回归。

## 二、事件总线升级（Outbox → MQ）

### 现状

`internal/platform/outbox/publisher.go` 的 `EventBusPublisher` 发布到内存事件总线，事件只在进程内分发，无法跨服务。Outbox 的语义是「跨服务事件分发」，内存总线不满足。

### 设计

- 抽象 `Publisher` 接口（已有），新增 `MQPublisher` 实现：发布到 `queue.Queue`（当前为 Redis queue，切换 `queue.type` 即换 Kafka/RabbitMQ）。
- 新增 dispatcher：轮询 `outbox_events` 表 → 发布到 MQ → 批量 `MarkPublished`。
- 内存事件总线保留，作为进程内同步事件的快速通道（如 `EventBusPublisher` 仅供单进程场景）。

数据流：

```
业务事务 → outbox_events 表 → dispatcher 轮询 → MQ 发布 → 订阅方消费
```

### 变更文件

| 文件 | 动作 |
|------|------|
| `internal/platform/outbox/publisher.go` | 新增 `MQPublisher`，保留 `EventBusPublisher` |
| `internal/platform/outbox/dispatcher.go` | 新增，轮询 outbox 表 |
| `internal/platform/queue/queue.go` | 队列接口供 outbox 使用 |

### 测试

- dispatcher 集成测试（内存假 MQ）。
- outbox 发布到 MQ 的端到端测试。

## 三、OAuth2/OIDC 第三方登录

### 现状

JWT + Casbin + Redis session 已有，无 OAuth 流程。

### 设计

- 引入 `golang.org/x/oauth2`。
- 抽象 Provider 接口：

```go
// internal/platform/oauth/provider.go
type Provider interface {
    Name() string
    AuthURL(state string) string
    Exchange(ctx context.Context, code string) (*UserInfo, error)
}
```

- 实现 `GoogleProvider`、`GitHubProvider`、`WeChatProvider`（配置驱动）。
- 登录流程：
  1. `GET /api/v1/oauth/{provider}/login` → 302 到 Provider AuthURL
  2. Provider 回调 `GET /api/v1/oauth/{provider}/callback` → Exchange code → UserInfo
  3. 匹配或创建用户 → 签发 JWT（复用现有 auth 模块）
- 配置 `oauth.providers.{name}.{client_id,client_secret,redirect_url}`。
- 新错误码：`3xxx` OAuth 模块。

### 变更文件

| 文件 | 动作 |
|------|------|
| `internal/platform/oauth/provider.go` | 新增接口 + UserInfo |
| `internal/platform/oauth/google.go` | 新增 |
| `internal/platform/oauth/github.go` | 新增 |
| `internal/platform/oauth/wechat.go` | 新增 |
| `internal/platform/oauth/factory.go` | 新增 |
| `internal/modules/oauth/module.go` | 新增模块 |
| `internal/modules/oauth/interfaces/handler.go` | 新增 |
| `internal/modules/oauth/interfaces/router.go` | 新增 |
| `internal/modules/oauth/application/service.go` | 新增 |
| `internal/config/config.go` | 加 oauth 配置 |

### 测试

- Provider mock 测试。
- handler 集成测试。

## 四、验证码

### 现状

登录/注册仅限流，无验证码。

### 设计

- 图形验证码：采用 `github.com/mojocn/base64Captcha`，返回 base64 图片。不采用自绘 SVG——第三方库成熟、可配字符干扰与验证强度，贴合框架「开箱即用」定位。
- 流程：
  1. `GET /api/v1/captcha` → 生成验证码，存 Redis（key `jimu:captcha:{id}`，TTL 5min），返回图片 base64 + captcha_id。
  2. 登录/注册带 `captcha_id` + `captcha_code`，服务端校验。
- 复用现有限流中间件（`/api/v1/captcha` 前置限流防刷）。
- 新错误码：`4xxx` 验证码模块。

### 变更文件

| 文件 | 动作 |
|------|------|
| `internal/platform/captcha/captcha.go` | 新增 |
| `internal/modules/auth/interfaces/handler.go` | 登录/注册加验证码校验 |
| `internal/config/config.go` | 加 captcha 配置 |

### 测试

- captcha 生成 + 校验单测。
- 登录集成测试带验证码。

## 五、Scheduler 持久化

### 现状

`CronScheduler` 纯内存（`jobs map[string]*JobInfo`），重启丢任务。`contract.JobRegistry` 目前只有 `AddFunc(spec, cmd)`，无任务名、无持久化语义。

### 设计

- 抽象 `Store` 接口：

```go
// internal/platform/scheduler/store.go
type Store interface {
    List(ctx context.Context) ([]JobDef, error)
    Save(ctx context.Context, job JobDef) error
    Delete(ctx context.Context, id string) error
}
```

- `JobDef` 含 `id`、`name`、`cron`、`enabled`、`created_at`、`updated_at`。
- `MemoryStore`（现有行为）+ `MySQLStore`（新表 `scheduled_jobs`）。
- 配置 `scheduler.store` 枚举（`memory`/`mysql`）。
- 启动时从 Store 加载任务注册到 cron；每次变更写 Store。
- 多实例协调：分布式锁（复用 `internal/platform/redis/lock.go`）确保同一任务单实例执行。
- 扩展 `JobRegistry`：

```go
type JobRegistry interface {
    AddFunc(spec string, cmd func()) error
    Register(name string, spec string, cmd func()) error // 命名任务，支持持久化
}
```

现有模块 `RegisterJobs` 用 `AddFunc` 不传名字——保持兼容。

### 变更文件

| 文件 | 动作 |
|------|------|
| `internal/platform/scheduler/store.go` | 新增接口 + JobDef |
| `internal/platform/scheduler/memory_store.go` | 新增 |
| `internal/platform/scheduler/mysql_store.go` | 新增 |
| `internal/platform/scheduler/scheduler.go` | 改造：加载/持久化 |
| `internal/contract/module.go` | `JobRegistry` 加 `Register` |
| `internal/config/config.go` | 加 `scheduler.store` |
| `migrations/` | 新增 `scheduled_jobs` 表 |

### 测试

- MySQL store 集成测试。
- scheduler 持久化单测。

## 全局变更

- `internal/config/config.go`：新增 `queue.type`、`oauth.providers`、`captcha`、`scheduler.store` 配置项及枚举校验。
- `README.md`：同步更新配置表、API 示例、项目结构。
- Swagger 注解：新增 API 使用中文 `@Summary`、`@Description`。

## 验证

- `make release-check COMPOSE_ENV=.env.example`。
- 各方向独立测试：`go test ./internal/platform/queue/... ./internal/platform/outbox/... ./internal/platform/oauth/... ./internal/platform/captcha/... ./internal/platform/scheduler/...`。

## 说明

- OAuth 回调地址、验证码 Redis TTL、scheduler 多实例协调策略等默认值在实现时确定并写入配置。
- 第三、四、五方向与第一、二方向无代码依赖，可并行开发。
