package testutil

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestMigrateIsSafeUnderConcurrency 复现「多包并行共用同一测试库」的场景：
// 并发调用 Migrate 时，数据库级咨询锁应让它们串行执行，全部成功。
// 没有该锁时，goose 的 DDL 会互相打架（典型报错：Duplicate column name 'tenant_id'）。
func TestMigrateIsSafeUnderConcurrency(t *testing.T) {
	tdb := SkipUnlessMysql(t)
	defer tdb.Close()

	const workers = 4
	var wg sync.WaitGroup
	errs := make([]error, workers)

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			errs[idx] = tdb.Migrate()
		}(i)
	}
	wg.Wait()

	for i, err := range errs {
		require.NoError(t, err, "并发迁移第 %d 个调用不应失败", i)
	}
}
