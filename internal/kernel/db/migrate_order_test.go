package db_test

import (
	"testing"
	"testing/fstest"

	"jimu/internal/config"
	"jimu/internal/contract"
	"jimu/internal/kernel/db"

	"github.com/stretchr/testify/require"
)

// TestMigrateEnabled_DownReversesCapabilityOrder 验证 down/redo 按反向能力序迭代：
// 声明了 Migrations 却缺 migrations/ 根的能力会报开发期错误，错误信息带能力名，
// 借此观测迭代顺序——up 按清单正向序、down/redo 按反向序。不触库（首个 fs.Stat 即报错）。
// 回归 F2：旧实现 down 也按正向迭代，第二轮起访问已被回滚的表报 "Table doesn't exist"。
func TestMigrateEnabled_DownReversesCapabilityOrder(t *testing.T) {
	// 两个声明了 Migrations 但缺 migrations/ 根的夹具能力
	caps := []contract.Descriptor{
		{Name: "aaa", Migrations: fstest.MapFS{}},
		{Name: "zzz", Migrations: fstest.MapFS{}},
	}
	// Host 不可达：一旦真正触库即失败，可佐证本测试全程未执行迁移
	cfg := config.DBConfig{Host: "127.0.0.1", Port: 1, User: "u", Password: "p", Database: "app"}

	err := db.MigrateEnabled(cfg, caps, "up")
	require.Error(t, err)
	require.Contains(t, err.Error(), "capability aaa", "up 应按清单正向序：aaa 先报错")

	err = db.MigrateEnabled(cfg, caps, "down")
	require.Error(t, err)
	require.Contains(t, err.Error(), "capability zzz", "down 应按反向序：zzz 先报错")

	err = db.MigrateEnabled(cfg, caps, "redo")
	require.Error(t, err)
	require.Contains(t, err.Error(), "capability zzz", "redo 应按反向序：zzz 先报错")
}
