package infrastructure

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"jimu/internal/capabilities/audit/domain"
	"jimu/internal/contract"
	"jimu/internal/shared/testutil"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	// 审计迁移自洽（001 建表、002 加 tenant_id、010 建 audit_chain_head），不依赖其他能力：
	// 只注入 audit，避免把无关能力的迁移拖进本测试二进制。
	repoRoot := testutil.MustRepoRoot()
	testutil.SetMigrateCaps([]contract.Descriptor{
		{Name: "audit", Migrations: os.DirFS(filepath.Join(repoRoot, "internal", "capabilities", "audit"))},
	})
}

// T1.7 审计日志写入必须带归属租户；且查询按 tenant_id 过滤（T3.4 的审计面：tenantID=0 不过滤）。
func TestT1_7_AuditLogPersistsTenantAndFiltersByTenant(t *testing.T) {
	tdb := testutil.SkipUnlessMysql(t)
	defer tdb.Close()

	require.NoError(t, tdb.Migrate(), "goose 迁移应成功")
	// 表内断言要求精确计数：先清空日志与链头（两者必须一起清，链头残留会让链式哈希接力到已删条目）。
	require.NoError(t, tdb.Truncate("audit_logs"), "清空 audit_logs")
	require.NoError(t, tdb.Truncate("audit_chain_head"), "清空 audit_chain_head")

	repo := NewMysqlAuditRepository(tdb.DB, "test-secret")
	tenantA := uint64(testutil.RandomInt(100000, 400000))
	tenantB := tenantA + 1
	ctx := context.Background()

	log := &domain.AuditLog{
		TenantID: tenantA,
		Username: "alice",
		Action:   "GET",
		Resource: "/api/v1/probe",
		Path:     "/api/v1/probe",
		Status:   200,
	}
	require.NoError(t, repo.Create(ctx, log))

	stored, err := repo.FindByID(ctx, log.ID)
	require.NoError(t, err)
	assert.Equal(t, tenantA, stored.TenantID, "T1.7 审计写入必须带归属租户")

	_, totalA, err := repo.List(ctx, tenantA, 0, 100, "id", "desc")
	require.NoError(t, err)
	assert.GreaterOrEqual(t, totalA, int64(1), "T1.7 本租户必须能看到该日志")

	_, totalB, err := repo.List(ctx, tenantB, 0, 100, "id", "desc")
	require.NoError(t, err)
	assert.Equal(t, int64(0), totalB, "T1.7 其他租户不得看到该日志")

	_, totalAll, err := repo.List(ctx, 0, 0, 100, "id", "desc")
	require.NoError(t, err)
	assert.GreaterOrEqual(t, totalAll, int64(1), "T3.4 tenantID=0（平台级视角）必须不过滤")
}
