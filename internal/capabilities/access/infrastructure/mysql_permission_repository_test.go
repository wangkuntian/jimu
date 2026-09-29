package infrastructure

import (
	"context"
	"testing"

	"jimu/internal/capabilities/access/domain"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func newPermissionTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&domain.Permission{}))
	return db
}

func TestPermissionRepositoryCRUD(t *testing.T) {
	db := newPermissionTestDB(t)
	repo := NewMysqlPermissionRepository(db)
	ctx := context.Background()

	// Create + FindByID
	perm := &domain.Permission{Name: "读用户", Resource: "/api/v1/users", Action: "GET"}
	require.NoError(t, repo.Create(ctx, perm))
	require.NotZero(t, perm.ID)

	got, err := repo.FindByID(ctx, perm.ID)
	require.NoError(t, err)
	assert.Equal(t, "/api/v1/users", got.Resource)

	// List 分页与排序
	require.NoError(t, repo.Create(ctx, &domain.Permission{Name: "写用户", Resource: "/api/v1/users", Action: "POST"}))
	list, total, err := repo.List(ctx, 0, 10, "id", "asc")
	require.NoError(t, err)
	assert.Equal(t, int64(2), total)
	assert.Len(t, list, 2)

	// 分页偏移
	list, total, err = repo.List(ctx, 1, 10, "id", "asc")
	require.NoError(t, err)
	assert.Equal(t, int64(2), total)
	assert.Len(t, list, 1)

	// Update
	perm.Name = "读用户v2"
	require.NoError(t, repo.Update(ctx, perm))
	got, err = repo.FindByID(ctx, perm.ID)
	require.NoError(t, err)
	assert.Equal(t, "读用户v2", got.Name)

	// Delete
	require.NoError(t, repo.Delete(ctx, perm.ID))
	_, err = repo.FindByID(ctx, perm.ID)
	assert.Error(t, err)
}
