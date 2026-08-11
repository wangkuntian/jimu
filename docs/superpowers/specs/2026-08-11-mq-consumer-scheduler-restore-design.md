# MQ 消费者接线 + Scheduler 持久化恢复 — 设计

> 日期：2026-08-11
> 范围：两个功能接线，兑现半成品能力。

## 背景

jimu 已实现发布端与存储端，但消费/恢复链路未接线：

1. **MQ 消费者**：`outbox.MQPublisher` 能发布 Outbox 事件到 MQ（Kafka/RabbitMQ），但 `queue.RegisterWorker` 无任何调用者，WorkerPool 从未启动。跨服务消费事件 → 触发业务逻辑的链路缺失。
2. **Scheduler RestoreFromStore**：`CronScheduler.RestoreFromStore` API 已实现，bootstrap 未调用。重启后 MySQL 持久化任务定义不恢复，全部任务实时注册。

## 目标

- 兑现 MQ 跨服务发布：MQ 模式下启动 WorkerPool 消费队列，outbox 事件反序列化后桥接回本地事件总线，与本服务内通知/审计复用现有订阅。
- 兑现 Scheduler 持久化：启动时 RestoreFromStore 恢复 MySQL 中持久化任务定义，内置任务实时注册与恢复去重。

## 关键决策

| 决策 | 选择 | 理由 |
|------|------|------|
| outbox 事件消费后做什么 | 桥接回事件总线 | 与 event_bus 模式行为一致，复用现有 Subscribe，外部服务可消费 MQ |
| 事件投递路径 | 统一走 outbox（删除 service 直接 PublishAsync，Update/Delete 补写） | 避免 MQ 模式下同一事件双路触发（service 直发 + outbox bridge）导致重复通知 |
| 消费走 WorkerPool 还是独立循环 | 复用 WorkerPool | 现成消费底座，RegisterWorker 机制已就绪，避免平行机制 |
| WorkerPool 何时启动 | 仅 `outbox.publisher=mq` 且 `queue.type∈{kafka,rabbitmq}` | Redis 队列单机无跨服务语义，与 outbox_process 定时器重复 |
| Restore 恢复范围 | 全部内置任务（outbox_process/metrics_collect/cleanup） | 兑现重启恢复核心价值 |
| Restore 签名 | 返回 `([]string, error)`（已恢复 id 列表） | 与实时注册去重 |

## 数据流

### MQ 消费端

```
outbox.Process → MQPublisher.Publish → queue.Submit(JobData{Type:"outbox:user.created", Payload:json(EventPayload)})
  → WorkerPool.Consume → GetWorker("outbox:user.created") → 桥接 worker
  → 反序列化 EventPayload → 按 EventType 转强类型 → globalBus.Publish(EventType, payload)
  → user module RegisterEvents（订阅全局总线裸主题）处理（发通知/审计等）
```

event_bus 模式（非 MQ）走同一条全局裸主题：bootstrap 注册 `outbox:*` → 全局总线桥接器，`EventBusPublisher` 发布 `outbox:user.created` 后由该桥接器转强类型发全局裸主题。两模式订阅方一致。

### Scheduler 恢复

```
bootstrap 启动：
  1. RestoreFromStore(ctx, cmdFactory)  // 恢复 MySQL 中 enabled 任务，返回已恢复 id
  2. 实时注册 jobFns，跳过已恢复 id（防双注册）
```

## Task 1：MQ 消费者

### 新增文件

**`internal/modules/admin/infrastructure/job_history_repository.go`**
- `mysqlJobHistoryRepository` 实现 `domain.JobHistoryRepository`
- 方法：`Create`、`ListByJobID`
- gorm 操作表 `job_history`（迁移 010 已建）

**`internal/modules/admin/infrastructure/dead_letter_repository.go`**
- `mysqlDeadLetterRepository` 实现 `domain.DeadLetterRepository`
- 方法：`Create`、`List`、`MarkResolved`
- gorm 操作表 `dead_letters`（迁移 010 已建）

### 修改文件

**`internal/app/container.go`**
- Container 加字段 `WorkerPool *queue.WorkerPool`（MQ 模式非 nil，否则 nil）
- 重构 outbox 分支：queue 实例 `q` 提到分支作用域，outbox 发布与 WorkerPool 复用同一 `q`
- 仅 MQ 模式：构造 3 仓储（`admininfra.NewMysqlJobRepository`、`NewMysqlJobHistoryRepository`、`NewMysqlDeadLetterRepository`）→ `queue.NewMySQLStore` → 断言 `q.(queue.Consumer)` → `queue.NewWorkerPool` → 存 Container

**`internal/app/bootstrap.go`**
- 新增 `registerOutboxWorkers(container)`：
  - 遍历已知业务事件类型（contract 常量），`queue.RegisterWorker("outbox:"+t, bridgeFn)`
  - bridgeFn：反序列化 `outbox.EventPayload` → 按 EventType 转强类型 → `container.EventBus.Publish(evt.EventType, payload)`（裸业务主题）
- event_bus 模式：新增全局总线桥接器——订阅 `outbox:*` 主题，转强类型后发裸业务主题（复用同一转换表）
- WorkerPool 启动：若 `container.WorkerPool != nil`，`go WorkerPool.Start()`（纳入 Application 生命周期，Stop 需关闭）

**`internal/modules/user/application/service.go`**
- `Create`/`Update`/`Delete` 删除 `PublishAsync` 直发
- `Update`/`Delete` 补写 `outbox.Add`（Update 传 changes 字段，Delete 传 user_id）
- Create 保留已有 outbox.Add

**`internal/modules/user/module.go`**
- `RegisterEvents` 由订阅私有总线 `m.eventBus` 改为订阅全局总线 `e`（裸业务主题）；转发 `notification.user.created`/`event.user.deleted` 逻辑保留

**`internal/contract/events.go`**（如需）
- 已有 `EventUserCreated/Updated/Deleted` 常量，直接复用

### 事件桥接细节

**类型转换表**：outbox.Add 存的内层 Payload 是强类型事件 JSON（如 `contract.UserCreatedEvent`）。桥接 worker 反序列化 `EventPayload` 后，按 EventType 查转换表还原强类型，再 Publish 到全局总线**裸业务主题**——保证订阅方类型断言成功。

**发布主题**：发布到全局总线裸业务主题（`user.created`/`user.updated`/`user.deleted`），与 service 原 `PublishAsync` 同主题。user module `RegisterEvents` 由订阅私有总线改为订阅全局总线（global bus 参数直用），转发逻辑保留。

```go
// bootstrap.go
var outboxTypeConverters = map[string]func(json.RawMessage) interface{}{
    contract.EventUserCreated: func(p json.RawMessage) interface{} {
        var e contract.UserCreatedEvent
        _ = json.Unmarshal(p, &e)
        return e
    },
    contract.EventUserUpdated: func(p json.RawMessage) interface{} {
        var e contract.UserUpdatedEvent
        _ = json.Unmarshal(p, &e)
        return e
    },
    contract.EventUserDeleted: func(p json.RawMessage) interface{} {
        var e contract.UserDeletedEvent
        _ = json.Unmarshal(p, &e)
        return e
    },
}

// bridgeFn 反序列化 outbox 载荷并发布强类型事件到全局业务主题
func bridgeFn(c *Container) queue.WorkerFunc {
    return func(ctx context.Context, payload string) error {
        var evt outbox.EventPayload
        if err := json.Unmarshal([]byte(payload), &evt); err != nil {
            return fmt.Errorf("unmarshal outbox event: %w", err)
        }
        conv, ok := outboxTypeConverters[evt.EventType]
        if !ok {
            return fmt.Errorf("no converter for outbox event type: %s", evt.EventType)
        }
        c.EventBus.Publish(evt.EventType, conv(evt.Payload)) // 裸业务主题
        return nil
    }
}

func registerOutboxWorkers(c *Container) {
    for eventType := range outboxTypeConverters {
        eventType := eventType
        queue.RegisterWorker("outbox:"+eventType, bridgeFn(c))
    }
}
```

**架构边界**：platform/queue 层不硬编码业务事件名。worker 注册（业务契约绑定）放装配层 bootstrap，platform 只提供通用 RegisterWorker 机制。

**两种模式行为一致**：事件统一走 outbox。event_bus 模式 `EventBusPublisher` 发布 `outbox:user.created` → 全局总线 `outbox:*` 桥接器转强类型发裸主题；MQ 模式经 outbox → MQ → WorkerPool → 桥接 worker 发裸主题。两路最终都从 `outbox.EventPayload` 转强类型发全局裸业务主题，订阅方收同类型，邮件/审计均触发且不重复。

### 条件启动

```go
// container.go outbox 分支
if cfg.Outbox.Publisher == config.OutboxPublisherMQ {
    q, err := queue.New(...)
    ...
    outboxPublisher = outbox.NewMQPublisher(q)
    if cfg.Queue.Type == "kafka" || cfg.Queue.Type == "rabbitmq" {
        consumer, ok := q.(queue.Consumer)
        if !ok { return nil, fmt.Errorf("queue %s does not implement consumer", cfg.Queue.Type) }
        store := queue.NewMySQLStore(
            admininfra.NewMysqlJobRepository(dbConn),
            admininfra.NewMysqlJobHistoryRepository(dbConn),
            admininfra.NewMysqlDeadLetterRepository(dbConn),
        )
        c.WorkerPool = queue.NewWorkerPool(queue.DefaultWorkerConfig, consumer, store)
    }
}
```

### WorkerPool 生命周期

- WorkerPool 现状非 `contract.Component`。bootstrap 启动：`go container.WorkerPool.Start()`。
- 优雅停机：Application components 追加匿名 component 包装 `Stop()`（内部 `p.cancel()` + `p.wg.Wait()`）。
- 未知事件类型 worker 缺失：`executeJob` 中 `GetWorker` 找不到 → store.MarkFailed + 记录 dead letter，不崩溃。

### 已知限制

- outbox 事件 job ID 是 outbox event ID，不在 jobs 表。WorkerPool 的 store 跟踪（MarkRunning/MarkSuccess）FindByID 失败，错误被 `_ =` 吞掉，**不影响事件执行**，仅 jobs 表无 outbox 跟踪记录。

## Task 2：Scheduler RestoreFromStore 接线

### 修改文件

**`internal/platform/scheduler/scheduler.go`**
- `RestoreFromStore` 签名改为 `([]string, error)`，返回已恢复 id 列表

**`internal/platform/scheduler/scheduler_store_test.go`**
- 同步更新 `TestRestoreFromStore`：断言返回值含 r1

**`internal/app/bootstrap.go`**
- 3 个静态任务闭包抽成 `map[string]func()`（jobFns）
- 启动顺序：
  ```go
  restored, err := container.Scheduler.RestoreFromStore(ctx, func(id string) func() {
      return jobFns[id]
  })
  if err != nil { logger.Error(...) }
  restoredSet := make(map[string]struct{}, len(restored))
  for _, id := range restored { restoredSet[id] = struct{}{} }
  for id, def := range jobFns {
      if _, ok := restoredSet[id]; ok { continue }  // 跳过已恢复
      Scheduler.AddNamedFunc(id, def.name, def.spec, def.fn)
  }
  ```
- name/spec：jobFns 需含 name/spec 元数据。改为 `map[string]jobDef{id: {name, spec, fn}}` 结构。

### 去重逻辑

RestoreFromStore 内部 `AddNamedFunc` 会重复 Save（幂等无害）+ 重复 AddFunc（双 entry）。返回已恢复 id，实时注册跳过 → 不双注册。

### 条件

- store=memory：Restore 返回空，全部实时注册（现状不变）
- store=mysql：Restore 恢复 MySQL 中任务，内置任务若已在 MySQL 则跳过实时注册

## 测试计划

| 测试 | 内容 |
|------|------|
| `job_history_repository_test.go` | Create/ListByJobID（sqlite 内存库，对齐 job_repository 测试模式） |
| `dead_letter_repository_test.go` | Create/List/MarkResolved（sqlite） |
| `scheduler_store_test.go` 更新 | TestRestoreFromStore 断言返回值 |
| 桥接 worker 单测 | 注册 outbox worker → 反序列化 EventPayload → 断言 eventBus 收到强类型事件（裸业务主题） |
| 全局总线 `outbox:*` 桥接器单测 | EventBusPublisher 发 `outbox:user.created` → 桥接器转强类型 → 断言裸主题收到 |
| user service 事件单测 | Create/Update/Delete 不再 PublishAsync，Update/Delete 写 outbox |

## 风险

- **桥接类型转换**（Task 1 关键风险）：MQ 与 event_bus 行为需一致，桥接需按 EventType 转换强类型。范围比预期大。
- **事件投递路径改造**：删除 service 直接 PublishAsync、Update/Delete 补写 outbox 属于行为变更，需确保 event_bus 模式（默认）下通知链路不回归（全局总线 `outbox:*` 桥接器补齐）。
- WorkerPool 生命周期：非 contract.Component，bootstrap 需包装管理优雅停机。
- Task 2 去重：Restore 返回 id 列表是核心新增逻辑，需测试覆盖。
- 仓储补全：history/dead_letter 无实现是历史欠账，补全后 WorkerPool 才可构造。
