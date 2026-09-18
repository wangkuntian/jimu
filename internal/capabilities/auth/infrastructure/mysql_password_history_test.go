package infrastructure

import (
	"context"
	"testing"

	authdomain "jimu/internal/capabilities/auth/domain"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func newPasswordHistoryTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&authdomain.PasswordHistory{}))
	return db
}

func TestPasswordHistoryRepositoryAddAndList(t *testing.T) {
	db := newPasswordHistoryTestDB(t)
	repo := NewMysqlPasswordHistoryRepository(db)
	ctx := context.Background()

	for _, hash := range []string{"h1", "h2", "h3"} {
		require.NoError(t, repo.Add(ctx, 1, 42, hash))
	}
	require.NoError(t, repo.Add(ctx, 1, 99, "other-user"))

	hashes, err := repo.ListRecentHashes(ctx, 42, 2)
	require.NoError(t, err)
	assert.Equal(t, []string{"h3", "h2"}, hashes, "应按新→旧返回且只取 limit 条")

	all, err := repo.ListRecentHashes(ctx, 42, 10)
	require.NoError(t, err)
	assert.Len(t, all, 3)

	// limit<=0 直接返回空
	none, err := repo.ListRecentHashes(ctx, 42, 0)
	require.NoError(t, err)
	assert.Empty(t, none)
}

func TestPasswordHistoryRepositoryTrim(t *testing.T) {
	db := newPasswordHistoryTestDB(t)
	repo := NewMysqlPasswordHistoryRepository(db)
	ctx := context.Background()

	for _, hash := range []string{"h1", "h2", "h3", "h4"} {
		require.NoError(t, repo.Add(ctx, 1, 42, hash))
	}

	require.NoError(t, repo.Trim(ctx, 42, 2))

	hashes, err := repo.ListRecentHashes(ctx, 42, 10)
	require.NoError(t, err)
	assert.Equal(t, []string{"h4", "h3"}, hashes, "只保留最近 2 条")

	// keep<=0 不做任何清理
	require.NoError(t, repo.Trim(ctx, 42, 0))
	hashes, err = repo.ListRecentHashes(ctx, 42, 10)
	require.NoError(t, err)
	assert.Len(t, hashes, 2)

	// 历史不足 keep 条时不删除
	require.NoError(t, repo.Trim(ctx, 42, 10))
	hashes, err = repo.ListRecentHashes(ctx, 42, 10)
	require.NoError(t, err)
	assert.Len(t, hashes, 2)
}
