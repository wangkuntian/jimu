package infrastructure

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"jimu/internal/capabilities/access/application"
	"jimu/internal/contract"
	"jimu/internal/kernel/tenant"
	"jimu/internal/shared/errors"
	"jimu/internal/shared/testutil"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	// roles.tenant_id 列与 idx_roles_tenant_name 唯一索引都来自 tenant 能力的 005 迁移，
	// 而 005 的 ALTER 依赖 access 建的角色基表 ⇒ 三个能力的迁移必须一起注入（与
	// internal/capabilities/tenant 与 user 的集成测试同一闭包、同一顺序）。
	repoRoot := testutil.MustRepoRoot()
	testutil.SetMigrateCaps([]contract.Descriptor{
		{Name: "user", Migrations: os.DirFS(filepath.Join(repoRoot, "internal", "capabilities", "user"))},
		{Name: "access", Migrations: os.DirFS(filepath.Join(repoRoot, "internal", "capabilities", "access"))},
		{Name: "tenant", Migrations: os.DirFS(filepath.Join(repoRoot, "internal", "capabilities", "tenant"))},
	})
}

// T1.3 同租户内角色名重复 → 拒绝（唯一索引 idx_roles_tenant_name(tenant_id, name)）。
// T1.4 不同租户使用同名角色 → 允许。
func TestT1_3_T1_4_RoleNameIsUniqueWithinTenantOnly(t *testing.T) {
	tdb := testutil.SkipUnlessMysql(t)
	defer tdb.Close()

	require.NoError(t, tdb.Migrate(), "goose 迁移应成功")

	svc := application.NewRoleService(NewMysqlRepository(tdb.DB))
	tenantA := uint64(testutil.RandomInt(100000, 400000))
	tenantB := tenantA + 1
	name := "role_" + testutil.RandomString(8)

	_, err := svc.Create(tenant.WithTenant(context.Background(), tenantA), application.CreateRoleRequest{Name: name, Description: "d"})
	require.NoError(t, err)

	_, err = svc.Create(tenant.WithTenant(context.Background(), tenantA), application.CreateRoleRequest{Name: name})
	require.Error(t, err, "T1.3 同租户内角色名必须唯一")
	assert.True(t, errors.IsCode(err, errors.CodeConflict), "重复角色名应映射为冲突码，实际: %v", err)

	_, err = svc.Create(tenant.WithTenant(context.Background(), tenantB), application.CreateRoleRequest{Name: name})
	require.NoError(t, err, "T1.4 不同租户允许同名角色")
}

// T3.3 tid=0 时创建角色 → 归属默认租户 id=1（与 T1.2 同源、此处断言角色侧语义）。
func TestT3_3_RoleWithoutTenantContextFallsBackToDefault(t *testing.T) {
	tdb := testutil.SkipUnlessMysql(t)
	defer tdb.Close()

	require.NoError(t, tdb.Migrate(), "goose 迁移应成功")

	repo := NewMysqlRepository(tdb.DB)
	created, err := application.NewRoleService(repo).Create(context.Background(), application.CreateRoleRequest{Name: "def_" + testutil.RandomString(8)})
	require.NoError(t, err)
	assert.Equal(t, tenant.DefaultTenantID, created.TenantID, "T3.3 tid=0 时角色必须归属默认租户")

	row, err := repo.FindByID(context.Background(), created.ID)
	require.NoError(t, err)
	assert.Equal(t, tenant.DefaultTenantID, row.TenantID, "T3.3 落库 tenant_id 必须为默认租户")
}
