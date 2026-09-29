package scheduler

import (
	"context"
	"sync"
	"time"
)

// MemoryStore 内存任务定义存储（不持久化，重启丢失）
type MemoryStore struct {
	mu   sync.RWMutex
	jobs map[string]JobDef
}

// NewMemoryStore 创建内存存储
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{jobs: make(map[string]JobDef)}
}

// List 列出所有任务定义
func (s *MemoryStore) List(ctx context.Context) ([]JobDef, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]JobDef, 0, len(s.jobs))
	for _, job := range s.jobs {
		result = append(result, job)
	}
	return result, nil
}

// Save 保存任务定义
func (s *MemoryStore) Save(ctx context.Context, job JobDef) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	if _, ok := s.jobs[job.ID]; ok {
		job.UpdatedAt = now
	} else {
		job.CreatedAt = now
		job.UpdatedAt = now
	}
	s.jobs[job.ID] = job
	return nil
}

// Delete 删除任务定义
func (s *MemoryStore) Delete(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.jobs, id)
	return nil
}

var _ Store = (*MemoryStore)(nil)
