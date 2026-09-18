package scheduler

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

func TestTriggerJobRunsCommand(t *testing.T) {
	s := NewWithStore(newTestLogger(), NewMemoryStore(), nil)
	var ran int64
	if err := s.AddNamedFunc("j1", "Job One", "@every 10s", func() {
		atomic.AddInt64(&ran, 1)
	}); err != nil {
		t.Fatalf("AddNamedFunc() error: %v", err)
	}
	if err := s.TriggerJob(context.Background(), "j1"); err != nil {
		t.Fatalf("TriggerJob() error: %v", err)
	}
	// recordRun 在 goroutine 中执行，轮询等待
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) && atomic.LoadInt64(&ran) == 0 {
		time.Sleep(10 * time.Millisecond)
	}
	if atomic.LoadInt64(&ran) != 1 {
		t.Fatalf("expected job to run once, ran %d times", atomic.LoadInt64(&ran))
	}
}

func TestTriggerJobNotFound(t *testing.T) {
	s := NewWithStore(newTestLogger(), NewMemoryStore(), nil)
	if err := s.TriggerJob(context.Background(), "nope"); err == nil {
		t.Fatal("expected error for unknown job, got nil")
	}
}

func TestSetEnabledToggle(t *testing.T) {
	s := NewWithStore(newTestLogger(), NewMemoryStore(), nil)
	if err := s.AddNamedFunc("j1", "Job One", "@every 10s", func() {}); err != nil {
		t.Fatalf("AddNamedFunc() error: %v", err)
	}

	// 初始启用
	if err := s.SetEnabled(context.Background(), "j1", false); err != nil {
		t.Fatalf("SetEnabled(false) error: %v", err)
	}
	if got := s.EntryCount(); got != 0 {
		t.Fatalf("EntryCount() after disable = %d, want 0", got)
	}
	// 手动触发被禁用任务应报错
	if err := s.TriggerJob(context.Background(), "j1"); err == nil {
		t.Fatal("expected error triggering disabled job, got nil")
	}

	// 恢复
	if err := s.SetEnabled(context.Background(), "j1", true); err != nil {
		t.Fatalf("SetEnabled(true) error: %v", err)
	}
	if got := s.EntryCount(); got != 1 {
		t.Fatalf("EntryCount() after enable = %d, want 1", got)
	}
}

func TestSetEnabledNotFound(t *testing.T) {
	s := NewWithStore(newTestLogger(), NewMemoryStore(), nil)
	if err := s.SetEnabled(context.Background(), "nope", false); err == nil {
		t.Fatal("expected error for unknown job, got nil")
	}
}
