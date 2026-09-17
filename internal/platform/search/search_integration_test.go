package search

import (
	"context"
	"testing"

	"jimu/internal/shared/testutil"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMysqlSearcherIntegration 针对真实 MySQL/MariaDB 的检索集成测试。
// CI 通过 services.mariadb 提供；本地无数据库时自动跳过。
func TestMysqlSearcherIntegration(t *testing.T) {
	tdb := testutil.SkipUnlessMysql(t)
	defer tdb.Close()

	require.NoError(t, tdb.Migrate(), "goose 迁移应成功")
	require.NoError(t, tdb.Truncate("search_documents"), "清空 search_documents")

	s, err := New(tdb.DB)
	require.NoError(t, err)

	ctx := context.Background()
	require.NoError(t, s.Index(ctx,
		Document{TenantID: 1, Type: "article", DocID: 1, Title: "golang backend", Body: "structured logging"},
		Document{TenantID: 1, Type: "article", DocID: 2, Title: "database tuning", Body: "indexes and queries"},
		Document{TenantID: 2, Type: "article", DocID: 3, Title: "golang elsewhere", Body: "other tenant"},
	))

	// 租户内检索：只命中本租户文档
	res, err := s.Search(ctx, 1, "golang", 10)
	require.NoError(t, err)
	require.Len(t, res, 1)
	assert.Equal(t, uint64(1), res[0].DocID)

	// 平台级检索（tenantID=0）：跨租户
	res, err = s.Search(ctx, 0, "golang", 10)
	require.NoError(t, err)
	assert.Len(t, res, 2)

	// 幂等覆盖：同 租户+类型+文档ID 再次索引不产生重复行
	require.NoError(t, s.Index(ctx, Document{TenantID: 1, Type: "article", DocID: 1, Title: "golang backend v2", Body: "updated"}))
	res, err = s.Search(ctx, 1, "golang", 10)
	require.NoError(t, err)
	require.Len(t, res, 1)
	assert.Contains(t, res[0].Title, "v2")

	// 删除索引
	require.NoError(t, s.Delete(ctx, 1, "article", 1))
	res, err = s.Search(ctx, 1, "golang", 10)
	require.NoError(t, err)
	assert.Empty(t, res)
}

// TestMysqlSearcherCJKFallbackIntegration 验证默认分词器下中文查询也能命中（LIKE 回退路径）
func TestMysqlSearcherCJKFallbackIntegration(t *testing.T) {
	tdb := testutil.SkipUnlessMysql(t)
	defer tdb.Close()

	require.NoError(t, tdb.Migrate(), "goose 迁移应成功")
	require.NoError(t, tdb.Truncate("search_documents"))

	s, err := New(tdb.DB)
	require.NoError(t, err)

	ctx := context.Background()
	require.NoError(t, s.Index(ctx,
		Document{TenantID: 1, Type: "article", DocID: 11, Title: "数据库调优", Body: "索引与查询计划"},
		Document{TenantID: 1, Type: "article", DocID: 12, Title: "golang backend", Body: "结构化日志"},
		Document{TenantID: 2, Type: "article", DocID: 13, Title: "数据库分片", Body: "其他租户"},
	))

	// 中文子串命中且标题优先
	res, err := s.Search(ctx, 1, "数据库", 10)
	require.NoError(t, err)
	require.Len(t, res, 1)
	assert.Equal(t, uint64(11), res[0].DocID)
	assert.Greater(t, res[0].Score, float64(0))

	// 租户隔离
	res, err = s.Search(ctx, 0, "数据库", 10)
	require.NoError(t, err)
	assert.Len(t, res, 2)

	// 正文中的中文也能命中
	res, err = s.Search(ctx, 1, "结构化", 10)
	require.NoError(t, err)
	require.Len(t, res, 1)
	assert.Equal(t, uint64(12), res[0].DocID)

	// 通配符按字面量处理，不应扩大匹配范围
	res, err = s.Search(ctx, 1, "%", 10)
	require.NoError(t, err)
	assert.Empty(t, res)
}
