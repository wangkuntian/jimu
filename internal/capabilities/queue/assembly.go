package queue

import (
	"context"

	queueinfra "jimu/internal/capabilities/queue/infrastructure"

	"gorm.io/gorm"
)

// NewMySQLStoreForDB 用 gorm 句柄构造工作池所需的持久化存储（装配期便利入口，
// 供 outbox 等消费方构造 WorkerPool，无需 import 本能力内部包）。
func NewMySQLStoreForDB(db *gorm.DB) *MySQLStore {
	return NewMySQLStore(
		queueinfra.NewMysqlJobRepository(db),
		queueinfra.NewMysqlJobHistoryRepository(db),
		queueinfra.NewMysqlDeadLetterRepository(db),
	)
}

// WorkerPoolComponent 把 WorkerPool 的启停纳入应用生命周期（contract.Component）。
type WorkerPoolComponent struct {
	pool *WorkerPool
}

// NewWorkerPoolComponent 创建 WorkerPool 生命周期组件。
func NewWorkerPoolComponent(pool *WorkerPool) *WorkerPoolComponent {
	return &WorkerPoolComponent{pool: pool}
}

// Start 启动 Worker 池。
func (c *WorkerPoolComponent) Start(context.Context) error {
	c.pool.Start()
	return nil
}

// Stop 停止 Worker 池并等待在途任务结束。
func (c *WorkerPoolComponent) Stop(context.Context) error {
	c.pool.Stop()
	return nil
}
