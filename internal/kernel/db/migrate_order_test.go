package db_test

import (
	"testing"
	"testing/fstest"

	"jimu/internal/config"
	"jimu/internal/contract"
	"jimu/internal/kernel/db"

	"github.com/stretchr/testify/require"
)

// capNames 取能力名列表（断言辅助）
func capNames(caps []contract.Descriptor) []string {
	out := make([]string, len(caps))
	for i, c := range caps {
		out[i] = c.Name
	}
	return out
}

// TestCapabilitiesInRunOrder 钉死迭代序选择：up/status 按清单正向序，
// down/redo 恰好是清单切片的反转（slices.Reverse），不是按名排序。
// 回归 F2：旧实现按 Name 降序 sort——名字序 ≠ 清单序（真实清单 user, role,
// permission, tenant, ... 按名降序会把 user 排到 tenant 前），叠加各能力迁移
// 深度不齐时 down 中途撞 "Table doesn't exist"。
func TestCapabilitiesInRunOrder(t *testing.T) {
	// 清单序仿真实 catalog 前缀（user, role, permission, tenant），名字序与反转序不同
	caps := []contract.Descriptor{
		{Name: "user"}, {Name: "role"}, {Name: "permission"}, {Name: "tenant"},
	}

	require.Equal(t, []string{"user", "role", "permission", "tenant"},
		capNames(db.CapabilitiesInRunOrder(caps, "up")), "up 应按清单正向序")
	require.Equal(t, []string{"user", "role", "permission", "tenant"},
		capNames(db.CapabilitiesInRunOrder(caps, "status")), "status 应按清单正向序")
	require.Equal(t, []string{"tenant", "permission", "role", "user"},
		capNames(db.CapabilitiesInRunOrder(caps, "down")), "down 应恰为清单切片反转")
	require.Equal(t, []string{"tenant", "permission", "role", "user"},
		capNames(db.CapabilitiesInRunOrder(caps, "redo")), "redo 应恰为清单切片反转")

	// 不改动调用方切片
	require.Equal(t, []string{"user", "role", "permission", "tenant"}, capNames(caps),
		"迭代序选择不得改动调用方切片")
}

// TestMigrateEnabled_DownReversesCapabilityOrder 验证 MigrateEnabled 实际按
// capabilitiesInRunOrder 的选择迭代：声明了 Migrations 却缺 migrations/ 根的
// 能力会报开发期错误，错误信息带能力名，借此观测迭代顺序。不触库（首个
// fs.Stat 即报错）。钉住"运行器确实使用了该辅助函数"这一接线。
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
