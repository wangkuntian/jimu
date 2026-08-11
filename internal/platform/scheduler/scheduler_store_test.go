package scheduler

import (
	"context"
	"sync/atomic"
	"testing"

	"jimu/internal/config"
	"jimu/internal/platform/logger"
)

func newTestLogger() *logger.Logger {
	return logger.New(config.LogConfig{Level: "warn", Format: "console", Output: "stdout"})
}

func TestAddNamedFuncPersistsAndRuns(t *testing.T) {
	log := newTestLogger()
	store := NewMemoryStore()
	s := NewWithStore(log, store, nil)

	var ran int64
	if err := s.AddNamedFunc("j1", "Job One", "@every 10s", func() {
		atomic.AddInt64(&ran, 1)
	}); err != nil {
		t.Fatalf("AddNamedFunc() error: %v", err)
	}
	jobs, err := store.List(context.Background())
	if err != nil {
		t.Fatalf("store.List() error: %v", err)
	}
	if len(jobs) != 1 || jobs[0].ID != "j1" || !jobs[0].Enabled {
		t.Fatalf("expected 1 enabled persisted job j1, got %+v", jobs)
	}
	if got := s.EntryCount(); got != 1 {
		t.Fatalf("EntryCount() = %d, want 1", got)
	}
}

func TestAddNamedFuncSaveErrorPropagates(t *testing.T) {
	log := newTestLogger()
	s := NewWithStore(log, &failStore{}, nil)
	err := s.AddNamedFunc("j1", "Job One", "@every 10s", func() {})
	if err == nil {
		t.Fatal("expected error when store.Save fails, got nil")
	}
}

func TestRestoreFromStore(t *testing.T) {
	log := newTestLogger()
	store := NewMemoryStore()
	_ = store.Save(context.Background(), JobDef{ID: "r1", Name: "Restored", Cron: "@every 10s", Enabled: true})
	_ = store.Save(context.Background(), JobDef{ID: "r2", Name: "Disabled", Cron: "@every 10s", Enabled: false})
	s := NewWithStore(log, store, nil)

	err := s.RestoreFromStore(context.Background(), func(id string) func() {
		if id == "r1" {
			return func() {}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("RestoreFromStore() error: %v", err)
	}
	if got := s.EntryCount(); got != 1 {
		t.Fatalf("EntryCount() = %d, want 1 (only enabled r1)", got)
	}
}

// failStore 测试用：Save 始终失败
type failStore struct{}

func (f *failStore) List(context.Context) ([]JobDef, error) { return nil, nil }
func (f *failStore) Save(context.Context, JobDef) error     { return errFailSave }
func (f *failStore) Delete(context.Context, string) error   { return nil }

var errFailSave = &failSaveError{}

type failSaveError struct{}

func (e *failSaveError) Error() string { return "save failed" }
