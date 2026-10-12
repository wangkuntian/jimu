package infrastructure

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"jimu/internal/capabilities/tenant/application"
	"jimu/internal/contract"
	"jimu/internal/kernel/tenant"
	"jimu/internal/shared/errors"
	"jimu/internal/shared/testutil"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	// 与 mysql_repository_integration_test.go 的 init 保持**逐字一致**（同一测试二进制、
	// testutil.migrateCaps 是包级变量）。
	repoRoot := testutil.MustRepoRoot()
	testutil.SetMigrateCaps([]contract.Descriptor{
		{Name: "user", Migrations: os.DirFS(filepath.Join(repoRoot, "internal", "capabilities", "user"))},
		{Name: "access", Migrations: os.DirFS(filepath.Join(repoRoot, "internal", "capabilities", "access"))},
		{Name: "tenant", Migrations: os.DirFS(filepath.Join(repoRoot, "internal", "capabilities", "tenant"))},
	})
}

// T3.1b 默认租户编码：005_add_tenants.sql 的种子行为 id=1 / code=default。
// T3.2 删除默认租户 → 拒绝，且行仍然存在。
// T4.3 更新已存在租户的编码 → 拒绝（UpdateTenantRequest 无 Code 字段；真实库确认编码未变）。
func TestT3_1b_T3_2_T4_3_DefaultTenantIsSeededProtectedAndImmutable(t *testing.T) {
	tdb := testutil.SkipUnlessMysql(t)
	defer tdb.Close()

	require.NoError(t, tdb.Migrate(), "goose 迁移应成功")

	// 同包既有集成测试会 TRUNCATE tenants（005 的种子行随之消失）：这里按迁移的同一语义
	// 幂等重建，让本测试不依赖同包其它测试的执行顺序。
	require.NoError(t, tdb.Exec(
		"INSERT IGNORE INTO tenants (id, code, name, status) VALUES (?, 'default', '默认租户', 1)",
		tenant.DefaultTenantID,
	).Error, "幂等重建默认租户")

	svc := application.NewTenantService(NewMysqlRepository(tdb.DB))
	ctx := context.Background()

	// T3.1b
	got, err := svc.Get(ctx, tenant.DefaultTenantID)
	require.NoError(t, err)
	assert.Equal(t, "default", got.Code, "T3.1b 默认租户编码必须为 default")

	// T4.3：改名称不得改动编码
	name := "默认租户-改名"
	require.NoError(t, svc.Update(ctx, tenant.DefaultTenantID, application.UpdateTenantRequest{Name: &name}))
	after, err := svc.Get(ctx, tenant.DefaultTenantID)
	require.NoError(t, err)
	assert.Equal(t, "default", after.Code, "T4.3 编码不可变")
	assert.Equal(t, name, after.Name, "T4.3 名称应已更新（证明 Update 确实生效）")

	// T3.2
	err = svc.Delete(ctx, tenant.DefaultTenantID)
	require.Error(t, err, "T3.2 默认租户不可删除")
	assert.True(t, errors.IsCode(err, errors.CodeTenantProtected), "应映射为受保护错误码，实际: %v", err)
	_, err = svc.Get(ctx, tenant.DefaultTenantID)
	require.NoError(t, err, "T3.2 默认租户行必须仍然存在")
}
