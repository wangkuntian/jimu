package infrastructure

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"jimu/internal/contract"
	"jimu/internal/shared/testutil"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func init() {
	// 本包集成测试依赖完整闭包：user GORM 模型含 TenantID 字段，
	// users.tenant_id 列来自 tenant 能力的 005 迁移（该迁移还会 ALTER roles），
	// 而 tenant 005 的 ALTER 又依赖 user/role 建的基表 —— 三个能力的迁移必须
	// 一起注入，顺序与 catalog 拓扑序一致（user, role 在前）。
	// 迁移 embed 在能力根包，infrastructure 子包测试引用根包会构成 import cycle，
	// 故这里按源码路径直接定位各能力根目录（本文件向上 4 级 = 仓库根）。
	_, thisFile, _, _ := runtime.Caller(0)
	repoRoot := filepath.Join(filepath.Dir(thisFile), "..", "..", "..", "..")
	testutil.SetMigrateCaps([]contract.Descriptor{
		{Name: "user", Migrations: os.DirFS(filepath.Join(repoRoot, "internal", "capabilities", "user"))},
		{Name: "role", Migrations: os.DirFS(filepath.Join(repoRoot, "internal", "capabilities", "role"))},
		{Name: "tenant", Migrations: os.DirFS(filepath.Join(repoRoot, "internal", "capabilities", "tenant"))},
	})
}

// TestMysqlRepositoryMySQLIntegration 针对真实 MariaDB/MySQL 的集成测试。
// CI 通过 services.mariadb 提供；本地无数据库时由 SkipUnlessMysql 自动跳过。
func TestMysqlRepositoryMySQLIntegration(t *testing.T) {
	tdb := testutil.SkipUnlessMysql(t)
	defer tdb.Close()

	require.NoError(t, tdb.Migrate(), "goose 迁移应成功")
	require.NoError(t, tdb.Truncate("users"), "清空 users 表")

	repo := NewMysqlRepository(tdb.DB)
	ctx := context.Background()

	// Create
	u := newTestUser()
	require.NoError(t, repo.Create(ctx, u))
	require.NotZero(t, u.ID)

	// FindByID
	got, err := repo.FindByID(ctx, u.ID)
	require.NoError(t, err)
	assert.Equal(t, u.Username, got.Username)

	// List + Count
	_, total, err := repo.List(ctx, 0, 0, 10, "id", "desc")
	require.NoError(t, err)
	assert.GreaterOrEqual(t, total, int64(1))

	// Update
	status := int8(0)
	u.Status = status
	require.NoError(t, repo.Update(ctx, u))
	got, err = repo.FindByID(ctx, u.ID)
	require.NoError(t, err)
	assert.Equal(t, status, got.Status)

	// Delete（软删除）
	require.NoError(t, repo.Delete(ctx, u.ID))
	_, err = repo.FindByID(ctx, u.ID)
	assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
}
