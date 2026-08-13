package scheduler

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestObserveEmitsSuccessMetrics(t *testing.T) {
	s := NewWithStore(newTestLogger(), NewMemoryStore(), nil)
	var ran atomic.Bool
	require.NoError(t, s.AddNamedFunc("obs-1", "obs", "@every 1m", func() { ran.Store(true) }))
	require.NoError(t, s.TriggerJob(context.Background(), "obs-1"))

	require.Eventually(t, func() bool {
		for _, j := range s.Jobs() {
			if j.ID == "obs-1" && j.LastStatus == "success" {
				return true
			}
		}
		return false
	}, time.Second, 10*time.Millisecond)
	assert.True(t, ran.Load())

	// 指标：成功计数 ≥ 1
	assert.GreaterOrEqual(t, testutil.ToFloat64(schedulerJobsTotal.WithLabelValues("obs", "success")), float64(1))
}

func TestObserveCountsPanicAsFailure(t *testing.T) {
	s := NewWithStore(newTestLogger(), NewMemoryStore(), nil)
	require.NoError(t, s.AddNamedFunc("obs-2", "panicjob", "@every 1m", func() { panic("boom") }))
	require.NoError(t, s.TriggerJob(context.Background(), "obs-2"))

	require.Eventually(t, func() bool {
		for _, j := range s.Jobs() {
			if j.ID == "obs-2" && j.LastStatus == "failed" {
				return true
			}
		}
		return false
	}, time.Second, 10*time.Millisecond)

	// 失败路径：panic 恢复后状态为 failed，指标记 failure 且不记 success
	assert.GreaterOrEqual(t, testutil.ToFloat64(schedulerJobsTotal.WithLabelValues("panicjob", "failed")), float64(1))
	assert.Zero(t, testutil.ToFloat64(schedulerJobsTotal.WithLabelValues("panicjob", "success")))
}
