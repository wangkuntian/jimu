package id

import (
	"sync"
	"testing"
	"time"
)

func TestNewSnowflakeValidatesWorkerID(t *testing.T) {
	for _, wid := range []int64{-1, 1024, 2048} {
		if _, err := NewSnowflake(wid); err != ErrWorkerIDOutOfRange {
			t.Fatalf("NewSnowflake(%d) err = %v, want ErrWorkerIDOutOfRange", wid, err)
		}
	}
	if _, err := NewSnowflake(0); err != nil {
		t.Fatalf("NewSnowflake(0) err = %v", err)
	}
	if _, err := NewSnowflake(1023); err != nil {
		t.Fatalf("NewSnowflake(1023) err = %v", err)
	}
}

func TestSnowflakeConcurrentUniqueness(t *testing.T) {
	g, err := NewSnowflake(1)
	if err != nil {
		t.Fatal(err)
	}
	const n = 1000
	ids := make([]uint64, n)
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			id, err := g.NextID()
			if err != nil {
				t.Errorf("NextID err = %v", err)
				return
			}
			ids[i] = id
		}(i)
	}
	wg.Wait()

	seen := make(map[uint64]bool, n)
	for _, id := range ids {
		if id == 0 {
			t.Fatal("zero id")
		}
		if seen[id] {
			t.Fatalf("duplicate id %d", id)
		}
		seen[id] = true
	}
}

func TestSnowflakeMonotonicSingleWorker(t *testing.T) {
	g, err := NewSnowflake(1)
	if err != nil {
		t.Fatal(err)
	}
	var prev uint64
	for i := 0; i < 1000; i++ {
		id, err := g.NextID()
		if err != nil {
			t.Fatal(err)
		}
		if i > 0 && id <= prev {
			t.Fatalf("non-monotonic: %d <= %d", id, prev)
		}
		prev = id
	}
}

func TestSnowflakeClockBackwards(t *testing.T) {
	s := &snowflake{workerID: 1, lastStamp: time.Now().UnixMilli() - epoch + 1000}
	_, err := s.NextID()
	if err != ErrClockBackwards {
		t.Fatalf("err = %v, want ErrClockBackwards", err)
	}
}

func TestUUIDGeneratorUnique(t *testing.T) {
	g := NewUUIDGenerator()
	seen := make(map[uint64]bool, 1000)
	for i := 0; i < 1000; i++ {
		id, err := g.NextID()
		if err != nil {
			t.Fatal(err)
		}
		if seen[id] {
			t.Fatalf("duplicate uuid id %d", id)
		}
		seen[id] = true
	}
}
