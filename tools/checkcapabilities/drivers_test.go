package main

import (
	"maps"
	"path/filepath"
	"slices"
	"testing"

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

// TestEntryImportViolation 断言⑤的入口包半边：profiles/<name> 入口包只允许 import 能力
// 根包，任何 capabilities 子包（含已声明的驱动包）都违规 —— 驱动选中只允许发生在
// internal/profiles/<name>/drivers.go。
func TestEntryImportViolation(t *testing.T) {
	const entry = "jimu/profiles/full"

	// 合法：非 capabilities 依赖（internal/assembly 与形态库包）。
	require.False(t, entryImportViolation(entry, "jimu/internal/assembly"))
	require.False(t, entryImportViolation(entry, "jimu/internal/profiles/full"))
	// 合法：能力根包（取 Descriptor/Wire）。
	require.False(t, entryImportViolation(entry, "jimu/internal/capabilities/queue"))
	// 违规：已声明的驱动包也不许由入口包选中（选中点只在 internal/profiles/<name>/drivers.go）。
	require.True(t, entryImportViolation(entry, "jimu/internal/capabilities/queue/redis"))
	// 违规：未声明的 capabilities 子包（漏声明 + 从入口 blank import 的经典形态）。
	require.True(t, entryImportViolation(entry, "jimu/internal/capabilities/dataops/importer"))
	// 不在作用域：import 方不是形态入口包（归 internal/profiles/* 那半边）。
	require.False(t, entryImportViolation("jimu/internal/profiles/full", "jimu/internal/capabilities/queue/redis"))
	require.False(t, entryImportViolation(entry, "github.com/gin-gonic/gin"))
}

// TestIsCapabilityRootPackage 根包判定 = 前缀之后不含 "/"。
func TestIsCapabilityRootPackage(t *testing.T) {
	require.True(t, isCapabilityRootPackage("jimu/internal/capabilities/queue"))
	require.True(t, isCapabilityRootPackage("jimu/internal/capabilities/catalog"))
	require.False(t, isCapabilityRootPackage("jimu/internal/capabilities/queue/redis"))
	require.False(t, isCapabilityRootPackage("jimu/internal/capabilities"))
	require.False(t, isCapabilityRootPackage("jimu/internal/profiles/full"))
}

// TestCheckProfileDriverImportsAcceptsDeclaredDrivers 真仓 GREEN：形态生产代码的
// capabilities import 只有能力根包与已声明驱动包。
func TestCheckProfileDriverImportsAcceptsDeclaredDrivers(t *testing.T) {
	asms := profileAssemblies()
	require.NoError(t, checkProfileDriverImports(
		repoRoot(t), slices.Sorted(maps.Keys(asms)), availableDrivers(asms)))
}

// TestCheckProfileDriverImportsRejectsUndeclaredDriverImport 真仓 RED 路径：available 集
// 故意漏掉 full 实际 import 的驱动（dataops/csv）时，必须在文案里点出「哪个形态包 → import
// 了哪个包 → 未声明为驱动」。
func TestCheckProfileDriverImportsRejectsUndeclaredDriverImport(t *testing.T) {
	err := checkProfileDriverImports(repoRoot(t), []string{"full"}, map[string]bool{})
	require.Error(t, err)
	require.Contains(t, err.Error(), "jimu/internal/profiles/full")
	require.Contains(t, err.Error(), "jimu/internal/capabilities/dataops/csv")
	require.Contains(t, err.Error(), "no capability declares as a driver")
}

// repoRoot 返回仓库根（本文件位于 tools/checkcapabilities/）。
func repoRoot(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("..", ".."))
	require.NoError(t, err)
	return root
}
