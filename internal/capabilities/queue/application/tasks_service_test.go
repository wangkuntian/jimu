package application

import (
	"context"
	"testing"

	"jimu/internal/config"
	"jimu/internal/platform/logger"
	"jimu/internal/platform/scheduler"

	"github.com/stretchr/testify/assert"
)

func newTestScheduler(t *testing.T) *scheduler.CronScheduler {
	t.Helper()
	s := scheduler.NewWithStore(logger.New(config.LogConfig{Level: "warn", Format: "console", Output: "stdout"}), scheduler.NewMemoryStore(), nil)
	if err := s.AddNamedFunc("j1", "Job One", "@every 10s", func() {}); err != nil {
		t.Fatalf("AddNamedFunc error: %v", err)
	}
	return s
}

func TestAdminTaskServiceListTasks(t *testing.T) {
	// 未配置调度器
	svc := NewAdminTaskService(nil)
	assert.Len(t, svc.ListTasks(), 0)

	svc = NewAdminTaskService(newTestScheduler(t))
	tasks := svc.ListTasks()
	assert.Len(t, tasks, 1)
	assert.Equal(t, "j1", tasks[0].ID)
	assert.Equal(t, "Job One", tasks[0].Name)
	assert.True(t, tasks[0].Enabled)
}

func TestAdminTaskServiceTriggerTask(t *testing.T) {
	ctx := context.Background()

	// 未配置调度器
	svc := NewAdminTaskService(nil)
	assert.Error(t, svc.TriggerTask(ctx, "j1"))

	svc = NewAdminTaskService(newTestScheduler(t))
	assert.NoError(t, svc.TriggerTask(ctx, "j1"))

	// 任务不存在
	assert.Error(t, svc.TriggerTask(ctx, "nope"))
}

func TestAdminTaskServiceToggleTask(t *testing.T) {
	ctx := context.Background()

	// 未配置调度器
	svc := NewAdminTaskService(nil)
	assert.Error(t, svc.ToggleTask(ctx, "j1"))

	svc = NewAdminTaskService(newTestScheduler(t))
	// 任务存在且已启用 -> 切换为禁用
	assert.NoError(t, svc.ToggleTask(ctx, "j1"))

	// 任务不存在
	assert.Error(t, svc.ToggleTask(ctx, "nope"))
}

func TestAdminTaskServiceGetHistory(t *testing.T) {
	// 未配置调度器
	svc := NewAdminTaskService(nil)
	assert.Len(t, svc.GetHistory("j1"), 0)

	svc = NewAdminTaskService(newTestScheduler(t))
	history := svc.GetHistory("j1")
	assert.Len(t, history, 1)
	assert.Equal(t, "j1", history[0].TaskID)

	// 任务不存在
	assert.Len(t, svc.GetHistory("nope"), 0)
}
