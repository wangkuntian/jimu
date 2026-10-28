package infrastructure

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"jimu/internal/capabilities/tenant/domain"
	"jimu/internal/contract"
	"jimu/internal/shared/testutil"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func init() {
	// 本包集成测试依赖完整闭包：tenant 的 005 迁移会 ALTER users/roles（补 tenant_id 列），
	// 这两张基表来自 user/role 的迁移 —— 单独注入 tenant 在全新库上会报
	// `Table 'jimu_test.users' doesn't exist`。三个能力的迁移必须一起注入，
	// 顺序与 catalog 拓扑序一致（user, role 在前）。
	// 迁移 embed 在能力根包，infrastructure 子包测试引用根包会构成 import cycle，
	// 故这里经 testutil.RepoRoot 定位模块根（不用 runtime.Caller：生成项目的重型矩阵构建带 -trimpath，编译期路径会被写成模块相对路径 —— 见 testutil.RepoRoot 的注释）。
	repoRoot := testutil.MustRepoRoot()
	testutil.SetMigrateCaps([]contract.Descriptor{
		{Name: "user", Migrations: os.DirFS(filepath.Join(repoRoot, "internal", "capabilities", "user"))},
		{Name: "access", Migrations: os.DirFS(filepath.Join(repoRoot, "internal", "capabilities", "access"))},
		{Name: "tenant", Migrations: os.DirFS(filepath.Join(repoRoot, "internal", "capabilities", "tenant"))},
	})
}

// TestMysqlRepositoryMySQLIntegration 针对真实 MariaDB/MySQL 的租户仓储集成测试。
// CI 通过 services.mariadb 提供；本地无数据库时由 SkipUnlessMysql 自动跳过。
func TestTenantMysqlRepositoryMySQLIntegration(t *testing.T) {
	tdb := testutil.SkipUnlessMysql(t)
	defer tdb.Close()

	require.NoError(t, tdb.Migrate(), "goose 迁移应成功")
	require.NoError(t, tdb.Truncate("tenants"), "清空 tenants 表")

	repo := NewMysqlRepository(tdb.DB)
	ctx := context.Background()

	// Create
	t1 := &domain.Tenant{Code: "acme", Name: "Acme Inc", Status: 1}
	require.NoError(t, repo.Create(ctx, t1))
	require.NotZero(t, t1.ID)

	// FindByID
	got, err := repo.FindByID(ctx, t1.ID)
	require.NoError(t, err)
	assert.Equal(t, "acme", got.Code)

	// FindByCode
	byCode, err := repo.FindByCode(ctx, "acme")
	require.NoError(t, err)
	assert.Equal(t, t1.ID, byCode.ID)

	// List
	_, total, err := repo.List(ctx, 0, 10, "id", "desc", domain.TenantListFilter{})
	require.NoError(t, err)
	assert.GreaterOrEqual(t, total, int64(1))

	// Update
	got.Name = "Acme Updated"
	require.NoError(t, repo.Update(ctx, got))
	got, err = repo.FindByID(ctx, t1.ID)
	require.NoError(t, err)
	assert.Equal(t, "Acme Updated", got.Name)

	// Delete（软删除）
	require.NoError(t, repo.Delete(ctx, t1.ID))
	_, err = repo.FindByID(ctx, t1.ID)
	assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
}

// TestTenantMysqlRepositoryListFilters 真实库上的筛选与 LIKE 转义。
func TestTenantMysqlRepositoryListFilters(t *testing.T) {
	tdb := testutil.SkipUnlessMysql(t)
	defer tdb.Close()

	require.NoError(t, tdb.Migrate())
	require.NoError(t, tdb.Truncate("tenants"))

	repo := NewMysqlRepository(tdb.DB)
	ctx := context.Background()
	require.NoError(t, repo.Create(ctx, &domain.Tenant{Code: "acme", Name: "Acme Inc", Status: 1}))
	require.NoError(t, repo.Create(ctx, &domain.Tenant{Code: "bravo", Name: "Bravo Ltd", Status: 1}))
	// status 列带 `default:1`，GORM 不会把零值字段写进 INSERT（落库仍是 1），
	// 因此停用态必须像 HTTP 更新路径那样经 Update 显式写回。
	paused := &domain.Tenant{Code: "paused", Name: "100% Paused", Status: 1}
	require.NoError(t, repo.Create(ctx, paused))
	paused.Status = 0
	require.NoError(t, repo.Update(ctx, paused))

	list := func(filter domain.TenantListFilter) ([]domain.Tenant, int64) {
		t.Helper()
		got, total, err := repo.List(ctx, 0, 20, "id", "asc", filter)
		require.NoError(t, err)
		return got, total
	}

	if _, total := list(domain.TenantListFilter{}); total != 3 {
		t.Fatalf("no filter total = %d, want 3", total)
	}
	if _, total := list(domain.TenantListFilter{Keyword: "acme"}); total != 1 {
		t.Fatalf("keyword=acme total = %d, want 1", total)
	}
	if _, total := list(domain.TenantListFilter{Keyword: "Bravo"}); total != 1 {
		t.Fatalf("keyword=Bravo total = %d, want 1", total)
	}
	if _, total := list(domain.TenantListFilter{Keyword: "%"}); total != 1 {
		t.Fatalf("keyword=%% total = %d, want 1（只命中名字里真的有 %% 的 paused）", total)
	}
	if _, total := list(domain.TenantListFilter{Keyword: "_"}); total != 0 {
		t.Fatalf("keyword=_ total = %d, want 0（下划线必须被转义）", total)
	}
	zero := int8(0)
	if _, total := list(domain.TenantListFilter{Status: &zero}); total != 1 {
		t.Fatalf("status=0 total = %d, want 1", total)
	}
	if _, total := list(domain.TenantListFilter{Keyword: "acme", Status: &zero}); total != 0 {
		t.Fatalf("keyword+status total = %d, want 0", total)
	}
}
