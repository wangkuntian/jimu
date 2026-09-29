package generator

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"jimu/internal/capabilities/catalog"
)

// catalogRel / migrationRel 是 RenderCatalog 的两个产物键。
const (
	catalogRel   = "internal/capabilities/catalog/catalog.go"
	migrationRel = "internal/capabilities/catalog/migration.go"
)

// TestRenderCatalogMatchesGoldenForMinimal 是生成版 catalog 的逐字节回归网。黄金文件
// testdata/golden/catalog/*.txt 由**本仓真实文件** internal/capabilities/catalog/{catalog.go,
// migration.go} 收窄得来（函数体逐字同源，只替换 entries/import 与固化 knownNames；来源说明见
// task-3-report.md），因此本测试同时钉住「生成项目 catalog 与框架蓝本同构」。
func TestRenderCatalogMatchesGoldenForMinimal(t *testing.T) {
	set, err := ParseCapabilitySet("minimal", "", "")
	require.NoError(t, err)
	files, err := RenderCatalog(set)
	require.NoError(t, err)
	for rel, golden := range map[string]string{
		catalogRel:   "testdata/golden/catalog/minimal_catalog.go.txt",
		migrationRel: "testdata/golden/catalog/minimal_migration.go.txt",
	} {
		want, err := os.ReadFile(golden)
		require.NoError(t, err)
		assert.Equal(t, string(want), string(files[rel]), rel)
	}
}

// TestRenderCatalogMatchesGoldenForWithSelection 钉住 `--with` 子集形态：entries 收窄为
// user/access/queue ∪ 迁移携带的 tenant，migration.go 也逐字节钉住（当前与 minimal 同形 ——
// queue 无 schema 依赖；这条 golden 防的是「entries 泄漏进 migration.go」）。
func TestRenderCatalogMatchesGoldenForWithSelection(t *testing.T) {
	set, err := ParseCapabilitySet("", "user,access,queue", "")
	require.NoError(t, err)
	files, err := RenderCatalog(set)
	require.NoError(t, err)
	for rel, golden := range map[string]string{
		catalogRel:   "testdata/golden/catalog/with_user_access_queue_catalog.go.txt",
		migrationRel: "testdata/golden/catalog/with_user_access_queue_migration.go.txt",
	} {
		want, err := os.ReadFile(golden)
		require.NoError(t, err)
		assert.Equal(t, string(want), string(files[rel]), rel)
	}
}

// TestRenderCatalogKeepsTopologicalOrderAndCarriesMigrationDeps 钉住裁定修订第 1 条：entries 是
// **唯一** descriptor 来源，必须含迁移携带能力（tenant）—— 否则 filterAll/MigrationSet 取不到，
// `jimu migrate` 会漏它的建表/加列（P2.6 的 C1 在生成项目里复活）；且**不得**再有 migrationExtras
// 这条「只进迁移集」的第二路径。
func TestRenderCatalogKeepsTopologicalOrderAndCarriesMigrationDeps(t *testing.T) {
	set, err := ParseCapabilitySet("minimal", "", "")
	require.NoError(t, err)
	files, err := RenderCatalog(set)
	require.NoError(t, err)

	src := string(files[catalogRel])
	// 先 require.Contains 再比位置：符号缺失时 strings.Index 返回 -1，直接 assert.Less 会「误过」。
	for _, marker := range []string{"usermodule.Descriptor,", "accessmodule.Descriptor,", "tenantmodule.Descriptor,", "authmodule.Descriptor,"} {
		require.Contains(t, src, marker)
	}
	// catalog 拓扑序：user → access → tenant → auth（auth 的软依赖 tenant 排在它之前）。
	assert.Less(t, strings.Index(src, "usermodule.Descriptor,"), strings.Index(src, "accessmodule.Descriptor,"))
	assert.Less(t, strings.Index(src, "accessmodule.Descriptor,"), strings.Index(src, "tenantmodule.Descriptor,"))
	assert.Less(t, strings.Index(src, "tenantmodule.Descriptor,"), strings.Index(src, "authmodule.Descriptor,"))
	assert.NotContains(t, src, "migrationExtras")

	migration := string(files[migrationRel])
	// 迁移携带能力不再由 migration.go 持有（它已在 entries 里），migration.go 没有第二个清单。
	assert.NotContains(t, migration, "tenantmodule")
	assert.NotContains(t, migration, "migrationExtras")
	assert.Contains(t, migration, `"user":   {"tenant"},`)
	assert.Contains(t, migration, `"access": {"tenant"},`)
}

// TestRenderCatalogPinsKnownNamesToFullUniverse 钉住 S5：known 必须是框架全量能力名，否则子集
// 清单上 SoftRequires 指向缺席能力会被判 unknown。**不写死总数**（框架加第 26 个能力时不误伤）：
// 断言 known 覆盖 catalog 全量清单 + 若干 Ungated 条目，且产物字面量与 set.Known 等长。
func TestRenderCatalogPinsKnownNamesToFullUniverse(t *testing.T) {
	set, err := ParseCapabilitySet("minimal", "", "")
	require.NoError(t, err)
	files, err := RenderCatalog(set)
	require.NoError(t, err)
	src := string(files[catalogRel])
	assert.Subset(t, set.Known, catalog.Names(), "known 必须覆盖框架 catalog 全量清单")
	for _, n := range []string{"tenant", "mfa", "captcha", "breach", "apidocs", "grpc"} {
		assert.Contains(t, set.Known, n)
		assert.Contains(t, src, `"`+n+`"`, "known names must include %s", n)
	}
	assert.Contains(t, src, "capability.ValidateDeclarations(entries, knownNames)")
	assert.Equal(t, len(set.Known), countKnownNames(t, src), "knownNames 字面量必须与 set.Known 逐项同长")
}

// TestRenderCatalogRejectsEmptyKnownNames 是 Fix round 1 / Minor 1 的 fail-closed：空 knownNames
// 会让生成项目的 ValidateDeclarations 把所有软依赖判成 unknown，必须渲染期直接报错。
func TestRenderCatalogRejectsEmptyKnownNames(t *testing.T) {
	set, err := ParseCapabilitySet("minimal", "", "")
	require.NoError(t, err)
	set.Known = nil
	_, err = RenderCatalog(set)
	require.ErrorContains(t, err, "knownNames")
}

// TestRenderCatalogRejectsEntryOutsideKnownNames 是 Fix round 1 / Minor 1 的自洽断言：entries 里
// 出现白名单外的能力名（CapabilitySet 构造错误）同样 fail-closed。断言不写死总数。
func TestRenderCatalogRejectsEntryOutsideKnownNames(t *testing.T) {
	set := CapabilitySet{Shape: "app", Declared: []string{"ghost"}, Known: []string{"user", "access"}}
	_, err := RenderCatalog(set)
	require.ErrorContains(t, err, `"ghost"`)
}

// countKnownNames 数 knownNames 字面量里的元素个数（与黄金文件相互独立的计数断言）。
func countKnownNames(t *testing.T, src string) int {
	t.Helper()
	_, rest, ok := strings.Cut(src, "var knownNames = []string{")
	require.True(t, ok, "knownNames literal missing")
	block, _, ok := strings.Cut(rest, "\n}")
	require.True(t, ok, "knownNames literal not terminated")
	count := 0
	for _, line := range strings.Split(block, "\n") {
		if strings.TrimSpace(line) != "" {
			count++
		}
	}
	return count
}
