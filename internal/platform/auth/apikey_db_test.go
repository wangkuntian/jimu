package auth

import (
	"context"
	"testing"
	"time"

	adminapi "jimu/internal/modules/admin/domain"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func newTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&adminapi.APIKey{}))
	return db
}

// createKey 插入 API Key。用 map 避免 gorm 零值省略（sqlite default:true 会掩盖显式 false）
func createKey(t *testing.T, db *gorm.DB, key *adminapi.APIKey) *adminapi.APIKey {
	err := db.Model(&adminapi.APIKey{}).Create(map[string]interface{}{
		"name":       key.Name,
		"key_prefix": key.KeyPrefix,
		"key_hash":   key.KeyHash,
		"scopes":     key.Scopes,
		"enabled":    key.Enabled,
		"expires_at": key.ExpiresAt,
	}).Error
	require.NoError(t, err)
	var saved adminapi.APIKey
	require.NoError(t, db.Where("key_hash = ?", key.KeyHash).First(&saved).Error)
	return &saved
}

func TestDBAPIKeyStore_GetByKeyHash(t *testing.T) {
	db := newTestDB(t)
	createKey(t, db, &adminapi.APIKey{
		Name:      "test",
		KeyPrefix: "jimu_abc",
		KeyHash:   "deadbeef",
		Scopes:    `["read","write"]`,
		Enabled:   true,
	})

	store := NewDBAPIKeyStore(db)
	got, err := store.GetByKeyHash(context.Background(), "deadbeef")
	require.NoError(t, err)
	assert.Equal(t, "test", got.Name)
	assert.Equal(t, []string{"read", "write"}, got.Scopes)
	assert.True(t, got.Enabled)
}

func TestDBAPIKeyStore_GetByKeyHashNotFound(t *testing.T) {
	store := NewDBAPIKeyStore(newTestDB(t))
	_, err := store.GetByKeyHash(context.Background(), "nope")
	assert.Error(t, err)
}

func TestDBAPIKeyStore_UpdateLastUsed(t *testing.T) {
	db := newTestDB(t)
	row := createKey(t, db, &adminapi.APIKey{Name: "t", KeyHash: "h", Enabled: true})

	store := NewDBAPIKeyStore(db)
	now := time.Now()
	require.NoError(t, store.UpdateLastUsed(context.Background(), row.ID, now))

	var got adminapi.APIKey
	require.NoError(t, db.First(&got, row.ID).Error)
	assert.WithinDuration(t, now, got.LastUsed, time.Second)
}

func TestVerifyWithDBStore(t *testing.T) {
	db := newTestDB(t)
	fullKey := "jimu_" + "abcdef0123456789abcdef0123456789"
	require.NoError(t, db.Create(&adminapi.APIKey{
		Name:      "t",
		KeyPrefix: fullKey[:min(16, len(fullKey))],
		KeyHash:   HashKey(fullKey),
		Enabled:   true,
	}).Error)

	v := NewAPIKeyVerifier(NewDBAPIKeyStore(db))
	got, err := v.Verify(context.Background(), fullKey)
	require.NoError(t, err)
	assert.Equal(t, "t", got.Name)
	assert.Equal(t, fullKey[:min(16, len(fullKey))], got.KeyPrefix)
}

func TestVerifyWithDBStoreDisabled(t *testing.T) {
	db := newTestDB(t)
	fullKey := "jimu_" + "abcdef0123456789abcdef0123456789"
	createKey(t, db, &adminapi.APIKey{
		Name:    "t",
		KeyHash: HashKey(fullKey),
		Enabled: false,
	})

	v := NewAPIKeyVerifier(NewDBAPIKeyStore(db))
	got, err := v.Verify(context.Background(), fullKey)
	assert.Nil(t, got)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "disabled")
}
