package infrastructure

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"jimu/internal/capabilities/tenant/domain"
	"jimu/internal/contract"
	"jimu/internal/shared/testutil"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func init() {
	// 本包集成测试依赖 tenants 表：注入租户能力的迁移。
	// 迁移 embed 在能力根包（tenants/migrations.go），infrastructure 子包测试
	// 引用根包会构成 import cycle，故这里按源码路径直接定位迁移目录。
	_, thisFile, _, _ := runtime.Caller(0)
	capRoot := filepath.Join(filepath.Dir(thisFile), "..")
	testutil.SetMigrateCaps([]contract.Descriptor{
		{Name: "tenant", Migrations: os.DirFS(capRoot)},
	})
}

// TestMysqlRepositoryMySQLIntegration 针对真实 MariaDB/MySQL 的租户仓储集成测试。
// CI 通过 services.mariadb 提供；本地无数据库时由 SkipUnlessMysql 自动跳过。
func TestTenantMysqlRepositoryMySQLIntegration(t *testing.T) {
	tdb := testutil.SkipUnlessMysql(t)
	defer tdb.Close()

	require.NoError(t, tdb.Migrate(), "goose 迁移应成功")
	require.NoError(t, tdb.Truncate("tenants"), "清空 tenants 表")

	repo := NewMysqlRepository(tdb.DB)
	ctx := context.Background()

	// Create
	t1 := &domain.Tenant{Code: "acme", Name: "Acme Inc", Status: 1}
	require.NoError(t, repo.Create(ctx, t1))
	require.NotZero(t, t1.ID)

	// FindByID
	got, err := repo.FindByID(ctx, t1.ID)
	require.NoError(t, err)
	assert.Equal(t, "acme", got.Code)

	// FindByCode
	byCode, err := repo.FindByCode(ctx, "acme")
	require.NoError(t, err)
	assert.Equal(t, t1.ID, byCode.ID)

	// List
	_, total, err := repo.List(ctx, 0, 10, "id", "desc")
	require.NoError(t, err)
	assert.GreaterOrEqual(t, total, int64(1))

	// Update
	got.Name = "Acme Updated"
	require.NoError(t, repo.Update(ctx, got))
	got, err = repo.FindByID(ctx, t1.ID)
	require.NoError(t, err)
	assert.Equal(t, "Acme Updated", got.Name)

	// Delete（软删除）
	require.NoError(t, repo.Delete(ctx, t1.ID))
	_, err = repo.FindByID(ctx, t1.ID)
	assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
}
