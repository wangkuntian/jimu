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

func newLoginHistoryTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&authdomain.LoginHistory{}))
	return db
}

func TestLoginHistoryRepositoryCreateAndList(t *testing.T) {
	db := newLoginHistoryTestDB(t)
	repo := NewMysqlLoginHistoryRepository(db)
	ctx := context.Background()

	require.NoError(t, repo.Create(ctx, &authdomain.LoginHistory{
		TenantID: 1, UserID: 42, Username: "alice", Status: authdomain.LoginStatusSuccess, IP: "1.1.1.1",
	}))
	require.NoError(t, repo.Create(ctx, &authdomain.LoginHistory{
		TenantID: 1, UserID: 42, Username: "alice", Status: authdomain.LoginStatusFailed, Reason: "invalid password", IP: "2.2.2.2",
	}))
	require.NoError(t, repo.Create(ctx, &authdomain.LoginHistory{
		TenantID: 2, UserID: 43, Username: "bob", Status: authdomain.LoginStatusSuccess,
	}))

	records, total, err := repo.ListByUser(ctx, 1, 42, 0, 10)
	require.NoError(t, err)
	assert.Equal(t, int64(2), total)
	require.Len(t, records, 2)
	assert.Equal(t, authdomain.LoginStatusFailed, records[0].Status, "应按 id 倒序，最新在前")
	assert.Equal(t, "2.2.2.2", records[0].IP)

	// 分页
	page, total, err := repo.ListByUser(ctx, 1, 42, 1, 1)
	require.NoError(t, err)
	assert.Equal(t, int64(2), total)
	require.Len(t, page, 1)
	assert.Equal(t, authdomain.LoginStatusSuccess, page[0].Status)

	// 平台级视角（tenantID=0）不过滤租户，但按 user_id 隔离
	all, _, err := repo.ListByUser(ctx, 0, 42, 0, 10)
	require.NoError(t, err)
	assert.Len(t, all, 2)

	// 其他用户不可见
	other, total, err := repo.ListByUser(ctx, 1, 999, 0, 10)
	require.NoError(t, err)
	assert.Zero(t, total)
	assert.Empty(t, other)
}
