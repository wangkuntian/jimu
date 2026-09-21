package infrastructure

import (
	"context"
	"testing"

	"jimu/internal/capabilities/apikey/domain"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

func newAPIKeyRepoTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	assert.NoError(t, err)
	assert.NoError(t, db.AutoMigrate(&domain.APIKey{}))
	return db
}

func TestMysqlAPIKeyRepository(t *testing.T) {
	db := newAPIKeyRepoTestDB(t)
	repo := NewMysqlAPIKeyRepository(db)
	ctx := context.Background()

	// Create + FindByID（含租户字段）
	key := &domain.APIKey{ID: 1, TenantID: 1, Name: "web", KeyPrefix: "jimu_ab", KeyHash: "h1", Scopes: "[\"read\"]", Enabled: true, CreatedBy: 3}
	assert.NoError(t, repo.Create(ctx, key))
	got, err := repo.FindByID(ctx, 1)
	assert.NoError(t, err)
	assert.Equal(t, "web", got.Name)
	assert.Equal(t, uint64(1), got.TenantID)

	// FindByKeyHash
	byHash, err := repo.FindByKeyHash(ctx, "h1")
	assert.NoError(t, err)
	assert.Equal(t, uint64(1), byHash.ID)

	// List：平台级视角（tenantID=0）不过滤
	keys, total, err := repo.List(ctx, 0, 0, 10)
	assert.NoError(t, err)
	assert.Len(t, keys, 1)
	assert.Equal(t, int64(1), total)

	// List：按租户过滤（其他租户的 Key 不可见）
	other := &domain.APIKey{ID: 2, TenantID: 2, Name: "other", KeyPrefix: "jimu_cd", KeyHash: "h2", Enabled: true}
	assert.NoError(t, repo.Create(ctx, other))
	keys, total, err = repo.List(ctx, 1, 0, 10)
	assert.NoError(t, err)
	assert.Len(t, keys, 1)
	assert.Equal(t, int64(1), total)
	assert.Equal(t, "web", keys[0].Name)
	keys, total, err = repo.List(ctx, 2, 0, 10)
	assert.NoError(t, err)
	assert.Len(t, keys, 1)
	assert.Equal(t, int64(1), total)
	assert.Equal(t, "other", keys[0].Name)

	// Update
	key.Name = "web2"
	assert.NoError(t, repo.Update(ctx, key))
	got, _ = repo.FindByID(ctx, 1)
	assert.Equal(t, "web2", got.Name)

	// IncrementUseCount
	assert.NoError(t, repo.IncrementUseCount(ctx, 1))
	got, _ = repo.FindByID(ctx, 1)
	assert.Equal(t, int64(1), got.UseCount)

	// Delete
	assert.NoError(t, repo.Delete(ctx, 1))
	_, err = repo.FindByID(ctx, 1)
	assert.Error(t, err)
}
