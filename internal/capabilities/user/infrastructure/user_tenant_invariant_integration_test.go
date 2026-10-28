package infrastructure

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"jimu/internal/capabilities/user/application"
	"jimu/internal/capabilities/user/domain"
	"jimu/internal/contract"
	"jimu/internal/kernel/tenant"
	"jimu/internal/shared/testutil"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	// 与 mysql_repository_integration_test.go 的 init 保持**逐字一致**：同一测试二进制里
	// testutil.migrateCaps 是包级变量，两个文件都设同一份闭包才是幂等的（顺序无关）。
	repoRoot := testutil.MustRepoRoot()
	testutil.SetMigrateCaps([]contract.Descriptor{
		{Name: "user", Migrations: os.DirFS(filepath.Join(repoRoot, "internal", "capabilities", "user"))},
		{Name: "access", Migrations: os.DirFS(filepath.Join(repoRoot, "internal", "capabilities", "access"))},
		{Name: "tenant", Migrations: os.DirFS(filepath.Join(repoRoot, "internal", "capabilities", "tenant"))},
	})
}

// T1.1 有租户上下文时创建用户 → 落库 tenant_id == 上下文租户。
// T1.2 无租户上下文（tid=0）时创建 → 落默认租户 id=1。
func TestT1_1_T1_2_UserTenantAssignment(t *testing.T) {
	tdb := testutil.SkipUnlessMysql(t)
	defer tdb.Close()

	require.NoError(t, tdb.Migrate(), "goose 迁移应成功")
	require.NoError(t, tdb.Truncate("users"), "清空 users 表")

	repo := NewMysqlRepository(tdb.DB)
	svc := application.NewUserService(repo, staticTenantInfoSource{})

	tenantID := uint64(testutil.RandomInt(100000, 400000))
	ctx := tenant.WithTenant(context.Background(), tenantID)
	created, err := svc.Create(ctx, application.CreateUserRequest{
		Username: "t1_1_" + testutil.RandomString(8),
		Password: "password1",
	})
	require.NoError(t, err)
	createdID, err := strconv.ParseUint(created.ID, 10, 64)
	require.NoError(t, err)
	row, err := repo.FindByID(context.Background(), createdID)
	require.NoError(t, err)
	assert.Equal(t, tenantID, row.TenantID, "T1.1 有租户上下文时必须落该租户")

	fallback, err := svc.Create(context.Background(), application.CreateUserRequest{
		TenantID: strconv.FormatUint(tenant.DefaultTenantID, 10),
		Username: "t1_2_" + testutil.RandomString(8),
		Password: "password1",
	})
	require.NoError(t, err)
	fallbackID, err := strconv.ParseUint(fallback.ID, 10, 64)
	require.NoError(t, err)
	row, err = repo.FindByID(context.Background(), fallbackID)
	require.NoError(t, err)
	assert.Equal(t, tenant.DefaultTenantID, row.TenantID, "平台视角显式指定默认租户时必须落该租户")
}

type staticTenantInfoSource struct{}

func (staticTenantInfoSource) GetByIDs(_ context.Context, ids []uint64) (map[uint64]contract.TenantInfo, error) {
	out := make(map[uint64]contract.TenantInfo, len(ids))
	for _, id := range ids {
		out[id] = contract.TenantInfo{ID: id}
	}
	return out, nil
}

// T1.5 用户名与邮箱盲索引必须**跨租户全局唯一**（服务层预检之外的 DB 兜底）。
func TestT1_5_UsernameAndEmailHashAreGloballyUnique(t *testing.T) {
	tdb := testutil.SkipUnlessMysql(t)
	defer tdb.Close()

	require.NoError(t, tdb.Migrate(), "goose 迁移应成功")
	require.NoError(t, tdb.Truncate("users"), "清空 users 表")

	repo := NewMysqlRepository(tdb.DB)
	tenantA := uint64(testutil.RandomInt(100000, 400000))
	tenantB := tenantA + 1
	ctxA := tenant.WithTenant(context.Background(), tenantA)
	ctxB := tenant.WithTenant(context.Background(), tenantB)

	username := "dup_" + testutil.RandomString(8)
	require.NoError(t, repo.Create(ctxA, &domain.User{Username: username, Password: "x", Status: 1, TenantID: tenantA}))
	err := repo.Create(ctxB, &domain.User{Username: username, Password: "x", Status: 1, TenantID: tenantB})
	require.Error(t, err, "T1.5 用户名必须跨租户全局唯一")

	emailHash := testutil.RandomString(48)
	require.NoError(t, repo.Create(ctxA, &domain.User{Username: "e1_" + testutil.RandomString(8), Password: "x", Status: 1, TenantID: tenantA, EmailHash: &emailHash}))
	err = repo.Create(ctxB, &domain.User{Username: "e2_" + testutil.RandomString(8), Password: "x", Status: 1, TenantID: tenantB, EmailHash: &emailHash})
	require.Error(t, err, "T1.5 邮箱盲索引必须跨租户全局唯一")
}

func containsTenant(users []domain.User, tenantID uint64) bool {
	for _, u := range users {
		if u.TenantID == tenantID {
			return true
		}
	}
	return false
}

// T1.6 以租户 A 的上下文查询 → 取不到租户 B 的记录（按 tenant_id 过滤）。
// T3.4 tid=0 时查询 → 不过滤，跨租户结果都能看到（平台级视角）。
func TestT1_6_T3_4_UserListIsTenantScopedUnlessPlatformView(t *testing.T) {
	tdb := testutil.SkipUnlessMysql(t)
	defer tdb.Close()

	require.NoError(t, tdb.Migrate(), "goose 迁移应成功")
	require.NoError(t, tdb.Truncate("users"), "清空 users 表")

	repo := NewMysqlRepository(tdb.DB)
	tenantA := uint64(testutil.RandomInt(100000, 400000))
	tenantB := tenantA + 1
	require.NoError(t, repo.Create(context.Background(), &domain.User{Username: "a_" + testutil.RandomString(8), Password: "x", Status: 1, TenantID: tenantA}))
	require.NoError(t, repo.Create(context.Background(), &domain.User{Username: "b_" + testutil.RandomString(8), Password: "x", Status: 1, TenantID: tenantB}))

	ctxA := tenant.WithTenant(context.Background(), tenantA)
	scoped, _, err := repo.List(ctxA, domain.UserListFilter{TenantID: tenant.FromContext(ctxA)}, 0, 500, "id", "desc")
	require.NoError(t, err)
	assert.True(t, containsTenant(scoped, tenantA), "T1.6 必须能看到本租户记录")
	for _, u := range scoped {
		assert.NotEqual(t, tenantB, u.TenantID, "T1.6 不得看到其他租户的记录")
	}

	platform, _, err := repo.List(context.Background(), domain.UserListFilter{TenantID: tenant.FromContext(context.Background())}, 0, 500, "id", "desc")
	require.NoError(t, err)
	assert.True(t, containsTenant(platform, tenantA) && containsTenant(platform, tenantB), "T3.4 平台级视角必须不过滤")
}
