package tenant

import (
	"context"
	"testing"

	"jimu/internal/capabilities/tenant/domain"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestTenantInfoSourceGetByIDs(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&domain.Tenant{}))
	require.NoError(t, db.Create(&domain.Tenant{ID: 1, Code: "default", Name: "默认租户", Status: 1}).Error)
	require.NoError(t, db.Create(&domain.Tenant{ID: 2, Code: "acme", Name: "Acme", Status: 1}).Error)

	got, err := newTenantInfoSource(db).GetByIDs(context.Background(), []uint64{1, 2, 3})
	require.NoError(t, err)
	assert.Equal(t, "默认租户", got[1].Name)
	assert.Equal(t, "Acme", got[2].Name)
	assert.NotContains(t, got, uint64(3))
}
