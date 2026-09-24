package catalog

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// full 的迁移集必须与 catalog.All() 逐值同序（full 行为零变化的钉子）。
func TestMigrationSetFullEqualsCatalogOrder(t *testing.T) {
	all := map[string]bool{}
	for _, d := range All() {
		all[d.Name] = true
	}
	got := MigrationSet(all)
	require.Len(t, got, len(All()))
	for i, d := range All() {
		assert.Equal(t, d.Name, got[i].Name, "第 %d 项顺序", i)
	}
}

// 含 user/access 的形态必须带上 tenant 的迁移（schema 依赖：users/roles 的 tenant_id 列
// 由 tenant 的迁移 005 添加，而 user/access 的 ORM 模型始终写该列）。
func TestMigrationSetCarriesSchemaDeps(t *testing.T) {
	for _, declared := range []map[string]bool{
		{"user": true, "access": true, "auth": true},                // minimal
		{"user": true, "access": true, "apikey": true},              // machine
		{"user": true, "access": true, "auth": true, "audit": true}, // saas
	} {
		names := make([]string, 0, len(declared))
		for _, d := range MigrationSet(declared) {
			names = append(names, d.Name)
		}
		assert.Contains(t, names, "tenant", "含 user/access 的迁移集必须带 tenant：%v", declared)
		// 顺序仍是 catalog 序：user/access 在 tenant 之前。
		assert.Less(t, indexOf(names, "user"), indexOf(names, "tenant"))
		assert.Less(t, indexOf(names, "access"), indexOf(names, "tenant"))
	}

	// 不含 user/access 的集合不引入 tenant（依赖按需补齐，不做无差别放大）。
	only := MigrationSet(map[string]bool{"queue": true})
	names := make([]string, 0, len(only))
	for _, d := range only {
		names = append(names, d.Name)
	}
	assert.NotContains(t, names, "tenant")
	assert.Contains(t, names, "queue")
}

func indexOf(xs []string, want string) int {
	for i, x := range xs {
		if x == want {
			return i
		}
	}
	return -1
}
