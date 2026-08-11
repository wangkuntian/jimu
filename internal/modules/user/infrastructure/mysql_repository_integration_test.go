package infrastructure

import (
	"context"
	"testing"

	"jimu/internal/shared/testutil"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

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
	_, total, err := repo.List(ctx, 0, 10, "id", "desc")
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
