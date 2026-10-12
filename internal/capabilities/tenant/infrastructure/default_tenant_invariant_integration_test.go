// 文件名以 `default_` 开头是**有意**的，不要改名：Go 的 `go test` 按**文件名字典序**
// 执行测试函数（实测：同包内 `aa_test.go` 先于 `zz_test.go`）。`default_...` <
// `mysql_repository_integration_test.go` ⇒ 本测试先于同包的 `TRUNCATE tenants`
// （mysql_repository_integration_test.go:40）执行，因而能在种子行被抹掉**之前**观测
// 005_tenants.sql:27-28 自己写入的默认租户行。
//
// 这是 T3.1b 的**唯一**鉴别力来源，本测试**不得自播种**被断言的那一行：若本文件改名排到
// truncate 之后（或同包出现更早的 truncate），迁移种子就观测不到了 —— 此时要么断言变红
// （若删掉 005 的种子则必然红，见下），要么退化。**改名时务必同步核对本注释与报告 F1。**
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

// T3.1b 默认租户编码：005_tenants.sql:27-28 的种子行为 id=1 / code=default。
// T3.2 删除默认租户 → 拒绝，且行仍然存在。
// T4.3 更新已存在租户的名称不得改动编码（哨兵；真实守卫见 T4.3 处注释）。
func TestT3_1b_T3_2_T4_3_DefaultTenantIsSeededProtectedAndImmutable(t *testing.T) {
	tdb := testutil.SkipUnlessMysql(t)
	// 关连接必须用 `t.Cleanup` 而**不是** `defer tdb.Close()`：测试体的 defer 早于**所有**
	// `t.Cleanup` 执行（实测顺序 `body → defer → t.Cleanup`），而下面的还原也是 t.Cleanup。
	// t.Cleanup 是 **LIFO**，故本行先注册 ⇒ 还原先跑（连接仍可用）、Close 最后跑。
	// 若改回 `defer`，还原会先 `FindByID` 拿到 `sql: database is closed`，cleanup 里的
	// assert 失败并让本测试在**有库时必然 FAIL**（fix round 2 的 Critical）。
	t.Cleanup(func() { _ = tdb.Close() })

	require.NoError(t, tdb.Migrate(), "goose 迁移应成功")

	svc := application.NewTenantService(NewMysqlRepository(tdb.DB))
	ctx := context.Background()

	// T3.1b：断言的对象是**迁移自己**写入的行。破坏播种来源（删掉 005_tenants.sql:27-28 的
	// `INSERT IGNORE INTO tenants ... VALUES (1, 'default', '默认租户', 1)`）⇒ 本 Get 取不到
	// 行 ⇒ 下面的 require.NoError 变红。这里不插入任何行。
	got, err := svc.Get(ctx, tenant.DefaultTenantID)
	require.NoError(t, err, "T3.1b 迁移种子必须写入默认租户 id=1；取不到说明 005 的种子被破坏，"+
		"或本测试已不再早于同包的 TRUNCATE tenants 执行（重跑请重建测试库）")
	assert.Equal(t, "default", got.Code, "T3.1b 默认租户编码必须为 default")

	// 本测试随后会把 name 改掉：在**确认种子行存在之后**注册还原，避免把共享测试库里默认租户的
	// 展示名永久改写（消除「未来新增断言默认租户 name」或本地共享库多包连跑的顺序依赖）。
	// 注册点放在 T3.1b 之后：种子缺失时上面的 require 已 FailNow，不会注册出「还原不存在的行」
	// 的二次噪音失败——那种情况下的失败必须是单纯、可诊断的种子缺失。
	// 本 cleanup 晚于函数开头的 Close cleanup 注册 ⇒ LIFO 下**先于** Close 执行（连接仍可用）。
	originalName := got.Name
	t.Cleanup(func() {
		restore := originalName
		assert.NoError(t, svc.Update(context.Background(), tenant.DefaultTenantID,
			application.UpdateTenantRequest{Name: &restore}), "还原默认租户 name")
	})

	// T4.3：改名称不得改动编码。
	// **哨兵语义**：本次 Update 只带 Name，落库不得冲掉编码。它**不是**「编码变更被拒绝」的
	// 证明——本测试从未尝试改编码。编码不可变的真实守卫在别处：
	//   ① 类型层：UpdateTenantRequest 只有 Name/Status，调用方在编译期无法表达改编码
	//      （tenant/application/dto.go:14-17）；
	//   ② 仓储列白名单：mysql_repository.go:58-67 的 SaveOptimistic 只写 name/status。
	// 因此本断言的**可鉴别变异必须同时**让 (i) 写入集合包含 code、(ii) 写入时 t.Code 偏离
	// "default"。单独改任何一处都**仍绿**（brief 的 `t.Code = "mutated"` 被 ② 挡住；
	// 只把 code 加进写入集合则写入的仍是 "default"）——详见报告 F2。
	name := "默认租户-改名"
	require.NoError(t, svc.Update(ctx, tenant.DefaultTenantID, application.UpdateTenantRequest{Name: &name}))
	after, err := svc.Get(ctx, tenant.DefaultTenantID)
	require.NoError(t, err)
	assert.Equal(t, "default", after.Code, "T4.3 改名不得冲掉编码（哨兵）")
	assert.Equal(t, name, after.Name, "T4.3 名称应已更新（证明 Update 确实生效）")

	// T3.2
	err = svc.Delete(ctx, tenant.DefaultTenantID)
	require.Error(t, err, "T3.2 默认租户不可删除")
	assert.True(t, errors.IsCode(err, errors.CodeTenantProtected), "应映射为受保护错误码，实际: %v", err)
	_, err = svc.Get(ctx, tenant.DefaultTenantID)
	require.NoError(t, err, "T3.2 默认租户行必须仍然存在")
}
