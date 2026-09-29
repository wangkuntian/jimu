package scheduler

import (
	"context"
	"time"
)

// JobDef 持久化任务定义
type JobDef struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Cron      string    `json:"cron"`
	Enabled   bool      `json:"enabled"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Store 任务定义存储接口
type Store interface {
	// List 列出所有任务定义
	List(ctx context.Context) ([]JobDef, error)
	// Save 保存任务定义（新增或更新）
	Save(ctx context.Context, job JobDef) error
	// Delete 删除任务定义
	Delete(ctx context.Context, id string) error
}
