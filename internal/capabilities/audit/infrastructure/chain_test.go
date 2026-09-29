package infrastructure

import (
	"context"
	"testing"
	"time"

	"jimu/internal/capabilities/audit/domain"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func newAuditTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&domain.AuditLog{}, &auditChainHead{}))
	return db
}

func TestCreateBatchChainsEntryHashes(t *testing.T) {
	db := newAuditTestDB(t)
	repo := NewMysqlAuditRepository(db, "secret")
	ctx := context.Background()

	require.NoError(t, repo.CreateBatch(ctx, []domain.AuditLog{
		{TenantID: 1, Username: "alice", Action: "a", Path: "/a"},
		{TenantID: 1, Username: "alice", Action: "b", Path: "/b"},
	}))

	logs, err := repo.ListForVerify(ctx, 1, 0, 0, 100)
	require.NoError(t, err)
	require.Len(t, logs, 2)

	assert.Equal(t, "", logs[0].PrevHash, "链首 prev_hash 为空")
	assert.NotEmpty(t, logs[0].EntryHash)
	assert.Equal(t, logs[0].EntryHash, logs[1].PrevHash, "第二条应链接第一条")
	assert.True(t, logs[0].MatchesEntryHash([]byte("secret")))
	assert.True(t, logs[1].MatchesEntryHash([]byte("secret")))
	assert.False(t, logs[0].MatchesEntryHash(nil), "密钥不一致时校验应失败")

	head, err := repo.ChainHead(ctx, 1)
	require.NoError(t, err)
	assert.Equal(t, logs[1].EntryHash, head, "链头应指向最后一条")
}

func TestCreateBatchKeepsPerTenantChains(t *testing.T) {
	db := newAuditTestDB(t)
	repo := NewMysqlAuditRepository(db, "")
	ctx := context.Background()

	require.NoError(t, repo.CreateBatch(ctx, []domain.AuditLog{
		{TenantID: 1, Action: "a1", Path: "/a"},
		{TenantID: 2, Action: "b1", Path: "/b"},
		{TenantID: 1, Action: "a2", Path: "/a"},
	}))

	tenant1, err := repo.ListForVerify(ctx, 1, 0, 0, 100)
	require.NoError(t, err)
	require.Len(t, tenant1, 2)
	assert.Equal(t, "", tenant1[0].PrevHash)
	assert.Equal(t, tenant1[0].EntryHash, tenant1[1].PrevHash)

	tenant2, err := repo.ListForVerify(ctx, 2, 0, 0, 100)
	require.NoError(t, err)
	require.Len(t, tenant2, 1)
	assert.Equal(t, "", tenant2[0].PrevHash, "各租户链独立")

	head1, err := repo.ChainHead(ctx, 1)
	require.NoError(t, err)
	assert.Equal(t, tenant1[1].EntryHash, head1)

	head2, err := repo.ChainHead(ctx, 2)
	require.NoError(t, err)
	assert.Equal(t, tenant2[0].EntryHash, head2)
}

func TestTaintedRowFailsHashCheck(t *testing.T) {
	db := newAuditTestDB(t)
	repo := NewMysqlAuditRepository(db, "secret")
	ctx := context.Background()

	require.NoError(t, repo.CreateBatch(ctx, []domain.AuditLog{
		{TenantID: 1, Username: "alice", Action: "create", Path: "/users"},
	}))
	logs, err := repo.ListForVerify(ctx, 1, 0, 0, 10)
	require.NoError(t, err)
	require.Len(t, logs, 1)

	// 直接改库（绕过应用层）：校验应发现内容与哈希不一致
	require.NoError(t, db.Model(&domain.AuditLog{}).Where("id = ?", logs[0].ID).
		Update("action", "delete").Error)

	after, err := repo.ListForVerify(ctx, 1, 0, 0, 10)
	require.NoError(t, err)
	require.Len(t, after, 1)
	assert.False(t, after[0].MatchesEntryHash([]byte("secret")), "篡改后哈希校验必须失败")
}

func TestCreateAssignsChainedHash(t *testing.T) {
	db := newAuditTestDB(t)
	repo := NewMysqlAuditRepository(db, "")
	ctx := context.Background()

	entry := &domain.AuditLog{TenantID: 1, Action: "single", Path: "/x", CreatedAt: time.Now()}
	require.NoError(t, repo.Create(ctx, entry))

	assert.NotZero(t, entry.ID)
	assert.NotEmpty(t, entry.EntryHash, "单条写入也应计算哈希")
	assert.Equal(t, "", entry.PrevHash)
	assert.True(t, entry.MatchesEntryHash(nil))
}
