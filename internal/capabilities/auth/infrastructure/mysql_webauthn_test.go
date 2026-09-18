package infrastructure

import (
	"context"
	"testing"
	"time"

	authdomain "jimu/internal/capabilities/auth/domain"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func newWebAuthnTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&authdomain.WebAuthnCredential{}))
	return db
}

func newCredential(id uint64, tenantID, userID uint64, credentialID string) *authdomain.WebAuthnCredential {
	return &authdomain.WebAuthnCredential{
		ID:           id,
		TenantID:     tenantID,
		UserID:       userID,
		CredentialID: credentialID,
		PublicKey:    []byte{1, 2, 3},
		SignCount:    5,
	}
}

func TestWebAuthnRepositoryCreateAndFind(t *testing.T) {
	db := newWebAuthnTestDB(t)
	repo := NewMysqlWebAuthnCredentialRepository(db)
	ctx := context.Background()

	require.NoError(t, repo.Create(ctx, newCredential(0, 1, 42, "cred-a")))

	found, err := repo.FindByCredentialID(ctx, "cred-a")
	require.NoError(t, err)
	assert.Equal(t, uint64(42), found.UserID)
	assert.Equal(t, []byte{1, 2, 3}, found.PublicKey)
	assert.Equal(t, uint32(5), found.SignCount)

	_, err = repo.FindByCredentialID(ctx, "missing")
	assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
}

func TestWebAuthnRepositoryListIsolatesTenants(t *testing.T) {
	db := newWebAuthnTestDB(t)
	repo := NewMysqlWebAuthnCredentialRepository(db)
	ctx := context.Background()

	require.NoError(t, repo.Create(ctx, newCredential(0, 1, 42, "cred-1")))
	require.NoError(t, repo.Create(ctx, newCredential(0, 2, 42, "cred-2")))
	require.NoError(t, repo.Create(ctx, newCredential(0, 1, 99, "cred-3")))

	list, err := repo.ListByUser(ctx, 1, 42)
	require.NoError(t, err)
	assert.Len(t, list, 1, "只返回该租户该用户的凭证")

	// tenantID=0 表示平台视角，不再按租户过滤
	all, err := repo.ListByUser(ctx, 0, 42)
	require.NoError(t, err)
	assert.Len(t, all, 2)

	count, err := repo.CountByUser(ctx, 42)
	require.NoError(t, err)
	assert.Equal(t, int64(2), count)
}

func TestWebAuthnRepositoryTouchAndRename(t *testing.T) {
	db := newWebAuthnTestDB(t)
	repo := NewMysqlWebAuthnCredentialRepository(db)
	ctx := context.Background()

	require.NoError(t, repo.Create(ctx, newCredential(0, 1, 42, "cred-a")))
	stored, err := repo.FindByCredentialID(ctx, "cred-a")
	require.NoError(t, err)
	require.Nil(t, stored.LastUsedAt)

	usedAt := time.Now().UTC().Truncate(time.Second)
	require.NoError(t, repo.Touch(ctx, stored.ID, 9, true, usedAt))

	again, err := repo.FindByCredentialID(ctx, "cred-a")
	require.NoError(t, err)
	assert.Equal(t, uint32(9), again.SignCount, "签名计数器应回写")
	assert.True(t, again.BackupState)
	require.NotNil(t, again.LastUsedAt)
	assert.WithinDuration(t, usedAt, again.LastUsedAt.UTC(), time.Second)

	require.NoError(t, repo.UpdateName(ctx, 1, 42, stored.ID, "MacBook"))
	renamed, err := repo.FindByCredentialID(ctx, "cred-a")
	require.NoError(t, err)
	assert.Equal(t, "MacBook", renamed.Name)
}

func TestWebAuthnRepositoryDeleteRequiresOwner(t *testing.T) {
	db := newWebAuthnTestDB(t)
	repo := NewMysqlWebAuthnCredentialRepository(db)
	ctx := context.Background()

	require.NoError(t, repo.Create(ctx, newCredential(0, 1, 42, "cred-a")))
	stored, err := repo.FindByCredentialID(ctx, "cred-a")
	require.NoError(t, err)

	// 他人 / 跨租户不可删除
	require.NoError(t, repo.Delete(ctx, 1, 99, stored.ID))
	require.NoError(t, repo.Delete(ctx, 2, 42, stored.ID))
	_, err = repo.FindByCredentialID(ctx, "cred-a")
	assert.NoError(t, err)

	// 归属人可删除
	require.NoError(t, repo.Delete(ctx, 1, 42, stored.ID))
	_, err = repo.FindByCredentialID(ctx, "cred-a")
	assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
}
