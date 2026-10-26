package application

import (
	"context"
	stderrors "errors"
	"testing"

	"jimu/internal/capabilities/tenant/domain"
	"jimu/internal/capabilities/tenant/infrastructure"
	"jimu/internal/contract"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

type fakeProvisioningRoleStore struct {
	cfg  contract.ProvisioningRoles
	err  error
	role uint64
}

func (f *fakeProvisioningRoleStore) ProvisionRoles(_ context.Context, _ uint64, cfg contract.ProvisioningRoles) (uint64, error) {
	f.cfg = cfg
	return f.role, f.err
}

func (f *fakeProvisioningRoleStore) AssignRole(context.Context, uint64, uint64) error { return nil }

func newTenantServiceTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:tenant_service_"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&domain.Tenant{}))
	return db
}

func TestTenantServiceCreateProvisionsRoles(t *testing.T) {
	db := newTenantServiceTestDB(t)
	store := &fakeProvisioningRoleStore{role: 99}
	service := NewTenantService(infrastructure.NewMysqlRepository(db)).WithDB(db).WithRoles(store)

	resp, err := service.Create(context.Background(), CreateTenantRequest{Code: "roles", Name: "Roles Inc"})
	require.NoError(t, err)
	require.NotEmpty(t, resp.ID)
	require.Len(t, store.cfg.Roles, 2)
	assert.Equal(t, contract.RoleCodeTenantAdmin, store.cfg.OwnerRole)
	assert.Equal(t, contract.RoleCodeTenantAdmin, store.cfg.Roles[0].Code)
	assert.Equal(t, contract.RoleCodeTenantMember, store.cfg.Roles[1].Code)

	var count int64
	require.NoError(t, db.Model(&domain.Tenant{}).Where("code = ?", "roles").Count(&count).Error)
	assert.Equal(t, int64(1), count)
}

func TestTenantServiceCreateRollsBackWhenRolesFail(t *testing.T) {
	db := newTenantServiceTestDB(t)
	store := &fakeProvisioningRoleStore{err: stderrors.New("role provisioning failed")}
	service := NewTenantService(infrastructure.NewMysqlRepository(db)).WithDB(db).WithRoles(store)

	_, err := service.Create(context.Background(), CreateTenantRequest{Code: "rollback", Name: "Rollback Inc"})
	require.ErrorContains(t, err, "provision tenant roles")

	var count int64
	require.NoError(t, db.Model(&domain.Tenant{}).Where("code = ?", "rollback").Count(&count).Error)
	assert.Zero(t, count)
}

func TestTenantServiceCreateRequiresRoleStore(t *testing.T) {
	db := newTenantServiceTestDB(t)
	service := NewTenantService(infrastructure.NewMysqlRepository(db)).WithDB(db)

	_, err := service.Create(context.Background(), CreateTenantRequest{Code: "no-store", Name: "No Store"})
	require.ErrorContains(t, err, "role provisioning is not configured")

	var count int64
	require.NoError(t, db.Model(&domain.Tenant{}).Where("code = ?", "no-store").Count(&count).Error)
	assert.Zero(t, count)
}
