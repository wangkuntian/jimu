package main

import (
	"path/filepath"
	"testing"

	"jimu/internal/profiles/registry"

	"github.com/stretchr/testify/require"
)

// TestCompareDriverSets 集合逐值比较（不比顺序）：imported 多出的是 "not declared"，
// declared 多出的是 "not imported"。
func TestCompareDriverSets(t *testing.T) {
	require.NoError(t, compareDriverSets(
		map[string]bool{"a": true, "b": true}, map[string]bool{"a": true, "b": true}))
	require.ErrorContains(t, compareDriverSets(
		map[string]bool{"a": true}, map[string]bool{"a": true, "b": true}), "not declared")
	require.ErrorContains(t, compareDriverSets(
		map[string]bool{"a": true, "b": true}, map[string]bool{"a": true}), "not imported")
}

// TestDriverImportViolation 断言⑤的判定口径：形态包 import 能力根包合法，import 已声明
// 驱动合法，import 其它 capabilities 子包（典型：漏声明的驱动）违规；非形态 import 方与
// 非 capabilities 依赖不在本断言作用域内（前者归断言④）。
func TestDriverImportViolation(t *testing.T) {
	declared := map[string]bool{"jimu/internal/capabilities/queue/redis": true}

	// 合法：能力根包（取 Descriptor/Wire）。
	require.False(t, driverImportViolation("jimu/internal/profiles/full", "jimu/internal/capabilities/queue", declared))
	// 合法：已声明的驱动包。
	require.False(t, driverImportViolation("jimu/internal/profiles/full", "jimu/internal/capabilities/queue/redis", declared))
	// 违规：capabilities 子包但没有任何能力声明它是驱动（漏声明的经典形态）。
	require.True(t, driverImportViolation("jimu/internal/profiles/full", "jimu/internal/capabilities/dataops/importer", declared))
	// 违规：驱动包存在于其它形态声明，但本 available 集不含它。
	require.True(t, driverImportViolation("jimu/internal/profiles/full", "jimu/internal/capabilities/queue/kafka", declared))
	// 不在作用域：import 方不是形态包（归断言④）或依赖不是 capabilities 子包。
	require.False(t, driverImportViolation("jimu/internal/capabilities/dataops", "jimu/internal/capabilities/queue/kafka", nil))
	require.False(t, driverImportViolation("jimu/internal/profiles/full", "github.com/gin-gonic/gin", nil))
}

// TestIsCapabilityRootPackage 根包判定 = 前缀之后不含 "/"。
func TestIsCapabilityRootPackage(t *testing.T) {
	require.True(t, isCapabilityRootPackage("jimu/internal/capabilities/queue"))
	require.True(t, isCapabilityRootPackage("jimu/internal/capabilities/catalog"))
	require.False(t, isCapabilityRootPackage("jimu/internal/capabilities/queue/redis"))
	require.False(t, isCapabilityRootPackage("jimu/internal/capabilities"))
	require.False(t, isCapabilityRootPackage("jimu/internal/profiles/full"))
}

// TestCheckProfileDriverImportsAcceptsDeclaredDrivers 真仓 GREEN：internal/profiles/* 生产
// 代码的 capabilities import 只有能力根包与已声明驱动包。
func TestCheckProfileDriverImportsAcceptsDeclaredDrivers(t *testing.T) {
	require.NoError(t, checkProfileDriverImports(repoRoot(t), availableDrivers(registry.All())))
}

// TestCheckProfileDriverImportsRejectsUndeclaredDriverImport 真仓 RED 路径：available 集
// 故意漏掉形态实际 import 的驱动（dataops/csv，full 与 enterprise 都选中）时，必须在文案里
// 点出「哪个形态包 → import 了哪个包 → 未声明为驱动」。
func TestCheckProfileDriverImportsRejectsUndeclaredDriverImport(t *testing.T) {
	available := availableDrivers(registry.All())
	delete(available, "jimu/internal/capabilities/dataops/csv")

	err := checkProfileDriverImports(repoRoot(t), available)
	require.Error(t, err)
	require.Contains(t, err.Error(), "jimu/internal/profiles/")
	require.Contains(t, err.Error(), "jimu/internal/capabilities/dataops/csv")
	require.Contains(t, err.Error(), "no capability declares as a driver")
}

// TestActiveSelectionMustImportExactlyOneProfile 选点包恰好 import 一个形态包（且是
// registry 认识的形态）；选点包 import registry 会把 5 个形态全拉回二进制，必须拦下。
func TestActiveSelectionMustImportExactlyOneProfile(t *testing.T) {
	require.NoError(t, checkActiveImports([]string{"jimu/internal/assembly", "jimu/internal/profiles/full"}))
	require.ErrorContains(t, checkActiveImports([]string{"jimu/internal/profiles/full", "jimu/internal/profiles/minimal"}), "exactly one")
	require.ErrorContains(t, checkActiveImports([]string{"jimu/internal/assembly"}), "exactly one")
	require.ErrorContains(t, checkActiveImports([]string{"jimu/internal/profiles/ghost"}), "unknown profile")
	require.ErrorContains(t, checkActiveImports([]string{"jimu/internal/profiles/registry"}), "registry")
}

// TestEntryPackageMustNotImportRegistryOrOtherProfiles 唯一入口只 import assembly 与选点包
// （+ 标准库）；形态包与 registry 都由选点包间接决定，入口不得直接引入。
func TestEntryPackageMustNotImportRegistryOrOtherProfiles(t *testing.T) {
	require.NoError(t, checkEntryImports([]string{"fmt", "os", "jimu/internal/assembly", "jimu/internal/profiles/active"}))
	require.ErrorContains(t, checkEntryImports([]string{"jimu/internal/profiles/minimal"}), "must not import")
	require.ErrorContains(t, checkEntryImports([]string{"jimu/internal/profiles/registry"}), "registry")
}

// TestProfileClosureLoadsTheUniqueEntry 形态闭包的根是唯一入口 ./cmd/server（`-overlay` 下），
// 不再是 profiles/<name>：T5 删除 5 个入口后这条口径不变，闭包里必须出现 cmd/server 包。
func TestProfileClosureLoadsTheUniqueEntry(t *testing.T) {
	for _, name := range registry.Names() {
		closure, err := profileClosure(repoRoot(t), name)
		require.NoError(t, err, "profile %s", name)
		require.Contains(t, closure, "jimu/cmd/server", "profile %s 的闭包根应是唯一入口", name)
		require.NotContains(t, closure, "jimu/profiles/"+name, "profile %s 不应再载入 profiles/<name>", name)
	}

	_, err := profileClosure(repoRoot(t), "ghost")
	require.ErrorContains(t, err, `unknown profile "ghost"`)
}

// repoRoot 返回仓库根（本文件位于 tools/checkcapabilities/）。
func repoRoot(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("..", ".."))
	require.NoError(t, err)
	return root
}
