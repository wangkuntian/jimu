package id

import (
	"testing"
)

// FuzzSnowflakeWorkerID 保证任意 workerID 下构造与生成不 panic。
// 越界返回 ErrWorkerIDOutOfRange，合法范围内 NextID 持续可用。
func FuzzSnowflakeWorkerID(f *testing.F) {
	f.Add(int64(0))
	f.Add(int64(1023))
	f.Add(int64(1024))
	f.Add(int64(-1))
	f.Fuzz(func(t *testing.T, workerID int64) {
		g, err := NewSnowflake(workerID)
		if err != nil {
			return
		}
		if _, err := g.NextID(); err != nil && err != ErrClockBackwards {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}
