package infrastructure

import (
	"context"
	"testing"
	"time"

	authdomain "jimu/internal/capabilities/mfa/domain"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func newTrustedDeviceTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&authdomain.TrustedDevice{}))
	return db
}

func newDevice(id, tenantID, userID uint64, hash string, expiresAt time.Time) *authdomain.TrustedDevice {
	return &authdomain.TrustedDevice{
		ID:        id,
		TenantID:  tenantID,
		UserID:    userID,
		TokenHash: hash,
		ExpiresAt: expiresAt,
	}
}

func TestTrustedDeviceRepositoryCreateAndFind(t *testing.T) {
	db := newTrustedDeviceTestDB(t)
	repo := NewMysqlTrustedDeviceRepository(db)
	ctx := context.Background()

	future := time.Now().Add(time.Hour)
	require.NoError(t, repo.Create(ctx, newDevice(0, 1, 42, "hash-a", future)))

	found, err := repo.FindByTokenHash(ctx, "hash-a")
	require.NoError(t, err)
	assert.Equal(t, uint64(42), found.UserID)
	assert.Equal(t, uint64(1), found.TenantID)

	_, err = repo.FindByTokenHash(ctx, "missing")
	assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
}

func TestTrustedDeviceRepositoryTouch(t *testing.T) {
	db := newTrustedDeviceTestDB(t)
	repo := NewMysqlTrustedDeviceRepository(db)
	ctx := context.Background()

	require.NoError(t, repo.Create(ctx, newDevice(0, 1, 42, "hash-a", time.Now().Add(time.Hour))))
	found, err := repo.FindByTokenHash(ctx, "hash-a")
	require.NoError(t, err)
	require.Nil(t, found.LastUsedAt)

	usedAt := time.Now().UTC().Truncate(time.Second)
	require.NoError(t, repo.Touch(ctx, found.ID, usedAt))

	again, err := repo.FindByTokenHash(ctx, "hash-a")
	require.NoError(t, err)
	require.NotNil(t, again.LastUsedAt)
	assert.WithinDuration(t, usedAt, again.LastUsedAt.UTC(), time.Second)
}

func TestTrustedDeviceRepositoryListByUserIsolatesTenants(t *testing.T) {
	db := newTrustedDeviceTestDB(t)
	repo := NewMysqlTrustedDeviceRepository(db)
	ctx := context.Background()
	future := time.Now().Add(time.Hour)

	require.NoError(t, repo.Create(ctx, newDevice(0, 1, 42, "hash-1", future)))
	require.NoError(t, repo.Create(ctx, newDevice(0, 2, 42, "hash-2", future)))
	require.NoError(t, repo.Create(ctx, newDevice(0, 1, 99, "hash-3", future)))

	devices, err := repo.ListByUser(ctx, 1, 42)
	require.NoError(t, err)
	assert.Len(t, devices, 1, "只返回该租户该用户的设备")

	// tenantID=0 表示平台视角，不再按租户过滤
	all, err := repo.ListByUser(ctx, 0, 42)
	require.NoError(t, err)
	assert.Len(t, all, 2)
}

func TestTrustedDeviceRepositoryDeleteRequiresOwner(t *testing.T) {
	db := newTrustedDeviceTestDB(t)
	repo := NewMysqlTrustedDeviceRepository(db)
	ctx := context.Background()

	require.NoError(t, repo.Create(ctx, newDevice(0, 1, 42, "hash-a", time.Now().Add(time.Hour))))
	found, err := repo.FindByTokenHash(ctx, "hash-a")
	require.NoError(t, err)

	// 他人不可删除
	require.NoError(t, repo.Delete(ctx, 1, 99, found.ID))
	_, err = repo.FindByTokenHash(ctx, "hash-a")
	assert.NoError(t, err, "越权删除应为无效操作")

	// 跨租户不可删除
	require.NoError(t, repo.Delete(ctx, 2, 42, found.ID))
	_, err = repo.FindByTokenHash(ctx, "hash-a")
	assert.NoError(t, err, "跨租户删除应为无效操作")

	// 归属人可删除
	require.NoError(t, repo.Delete(ctx, 1, 42, found.ID))
	_, err = repo.FindByTokenHash(ctx, "hash-a")
	assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
}

func TestTrustedDeviceRepositoryDeleteAllAndExpired(t *testing.T) {
	db := newTrustedDeviceTestDB(t)
	repo := NewMysqlTrustedDeviceRepository(db)
	ctx := context.Background()
	now := time.Now()

	require.NoError(t, repo.Create(ctx, newDevice(0, 1, 42, "expired", now.Add(-time.Hour))))
	require.NoError(t, repo.Create(ctx, newDevice(0, 1, 42, "alive", now.Add(time.Hour))))
	require.NoError(t, repo.Create(ctx, newDevice(0, 1, 99, "other", now.Add(time.Hour))))

	removed, err := repo.DeleteExpired(ctx, now)
	require.NoError(t, err)
	assert.Equal(t, int64(1), removed)

	alive, err := repo.ListByUser(ctx, 0, 42)
	require.NoError(t, err)
	require.Len(t, alive, 1)
	assert.Equal(t, "alive", alive[0].TokenHash)

	require.NoError(t, repo.DeleteAllByUser(ctx, 42))
	alive, err = repo.ListByUser(ctx, 0, 42)
	require.NoError(t, err)
	assert.Empty(t, alive)

	other, err := repo.ListByUser(ctx, 0, 99)
	require.NoError(t, err)
	assert.Len(t, other, 1, "只删除目标用户的设备")
}
