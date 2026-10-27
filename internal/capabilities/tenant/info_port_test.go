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

func TestTenantInfoSourceTenantActive(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&domain.Tenant{}))
	require.NoError(t, db.Create(&domain.Tenant{ID: 1, Code: "active", Name: "Active", Status: 1}).Error)
	require.NoError(t, db.Create(&domain.Tenant{ID: 2, Code: "disabled", Name: "Disabled", Status: 0}).Error)
	require.NoError(t, db.Model(&domain.Tenant{}).Where("id = ?", 2).Update("status", 0).Error)
	require.NoError(t, db.Create(&domain.Tenant{ID: 3, Code: "deleted", Name: "Deleted", Status: 1}).Error)
	require.NoError(t, db.Delete(&domain.Tenant{}, uint64(3)).Error)

	source := newTenantInfoSource(db)
	for _, tc := range []struct {
		name     string
		tenantID uint64
		want     bool
	}{
		{name: "active", tenantID: 1, want: true},
		{name: "disabled", tenantID: 2, want: false},
		{name: "deleted", tenantID: 3, want: false},
		{name: "missing", tenantID: 4, want: false},
		{name: "platform context", tenantID: 0, want: false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := source.TenantActive(context.Background(), tc.tenantID)
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}
