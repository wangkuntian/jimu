package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"jimu/internal/profiles/registry"
	"jimu/tools/internal/heavydeps"
	"jimu/tools/internal/profileoverlay"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// profileShardEnv 让 CI 的 race 分片只度量指定形态（逗号分隔，见 scripts/test_shards.sh）。
//
// 动机（实测）：`-race` 下每个形态都要做一次全依赖图 packages.Load + 全闭包行数统计，整包在 4 vCPU
// runner 上要 257s，而它只是 tools 分片里的一个包 —— 那一片因此成了整个 Race job 的长杆。按形态切开
// 后每片只度量自己那部分。**这不是减少 race 覆盖**：每个形态仍然在某个分片里被 `-race` 跑过，只是
// 不再挤在同一个进程/同一台 runner 上；跨形态关系（minimal ≤ 85% full 之类）仍由未设该变量时的
// 完整度量断言（默认路径的 Test job，非 race，整包 23s）。
const profileShardEnv = "JIMU_METRICS_PROFILES"

// 本文件只覆盖 report 本体的渲染与 overlay 口径；度量原语（路由/迁移/表/闭包/行数/直接依赖）
// 的单测已随实现搬到 tools/internal/projectmetrics（P2.7 抽取，两处共用一份口径）。

// TestRenderReportPinsTheCommittedShape 报告是入库产物：用手写 Metrics 钉住表头、归一化列、
// 验收断言与「go.mod 逐形态相同」的结论，渲染逻辑变化会让本用例失败。
func TestRenderReportPinsTheCommittedShape(t *testing.T) {
	ms := []Metrics{
		{Profile: "full", BinaryBytes: 200_000_000, Routes: 100, Migrations: 20, Tables: 20, Files: 300, Lines: 30_000, HeavyDeps: []string{"aws-sdk-go-v2", "excelize"}, Capabilities: []string{"auth", "user"}},
		{Profile: "minimal", BinaryBytes: 100_000_000, Routes: 30, Migrations: 5, Tables: 5, Files: 150, Lines: 10_000, Capabilities: []string{"auth"}},
	}
	out := renderReport(ms, 64)

	assert.Contains(t, out, "| 形态 | 二进制 (MB) | 相对 full | 路由数 | 迁移数 | 表数 | 本仓 Go 文件 | 本仓代码行 | 重型依赖 |")
	assert.Contains(t, out, "| `full` | 200.0 | 100.0% | 100 | 20 | 20 | 300 | 30000 | aws-sdk-go-v2, excelize |")
	assert.Contains(t, out, "| `minimal` | 100.0 | 50.0% | 30 | 5 | 5 | 150 | 10000 | - |")
	assert.Contains(t, out, "| 重型依赖 | 同一闭包（含第三方包）命中 `tools/internal/heavydeps` 前缀表的展示名，`-` 表示零 |")
	assert.Contains(t, out, "| 二进制 | `go build -overlay=<该形态> -o <tmp> ./cmd/server` 的产物大小 |")
	assert.Contains(t, out, "| 本仓 Go 文件 / 代码行 | `golang.org/x/tools/go/packages` 载入 `./cmd/server` 在该形态 overlay 下的 import 闭包，只统计本模块（`jimu/...`）的非 `_test.go` 文件 |")
	assert.Contains(t, out, "形态由 `internal/profiles/active` 的**构建期 overlay** 决定")
	assert.Contains(t, out, "- 二进制：`minimal` 是 `full` 的 50.0%（要求 ≤ 85%）")
	assert.Contains(t, out, "- 路由数：`minimal` 30 < `full` 100")
	assert.Contains(t, out, "五个形态的 go.mod 直接依赖数**逐形态完全相同**（各 64 个）")
	assert.Contains(t, out, "| `minimal` | auth |")
}

// TestRenderReportWithoutMinimal 无 minimal 时不渲染验收段（只按数据渲染，不对形态名做隐藏假设）；
// 无 full 基准时同样跳过。用两个形态走多形态分支（单形态分支由下一条测试钉住）。
func TestRenderReportWithoutMinimal(t *testing.T) {
	out := renderReport([]Metrics{{Profile: "full", BinaryBytes: 200}, {Profile: "saas", BinaryBytes: 2}}, 64)
	assert.NotContains(t, out, "要求 ≤ 85%")
	assert.Contains(t, out, "| `full` | 0.0 | 100.0% |")
}

// TestRenderSingleShapeReportHasNoMultiShapeClaims 是 T8 裁定 15 的收口：`jimu new` 的生成项目
// 里 `make compose-report` 只度量一个形态，报告必须单形态正确 —— 没有「相对 full」列、没有构建期
// overlay 叙述、没有「五个形态」的结论，也不引用生成项目里不存在的 README 章节与本仓测试名；
// 主模块名取 modulePath（生成时被受控重写改成 --module）。
func TestRenderSingleShapeReportHasNoMultiShapeClaims(t *testing.T) {
	out := renderReport([]Metrics{{
		Profile: "app", BinaryBytes: 12_345_678, Routes: 32, Migrations: 10, Tables: 9,
		Files: 181, Lines: 17_522, Capabilities: []string{"access", "queue", "user"},
	}}, 64)

	assert.Contains(t, out, "# 项目编译面报告")
	assert.Contains(t, out, "本项目只有一个形态、在生成期固定")
	assert.Contains(t, out, "| 形态 | 二进制 (MB) | 路由数 | 迁移数 | 表数 | 本模块 Go 文件 | 本模块代码行 | 重型依赖 |")
	assert.Contains(t, out, "| `app` | 12.3 | 32 | 10 | 9 | 181 | 17522 | - |")
	assert.Contains(t, out, "| 装配的能力（按装配顺序） |")
	assert.Contains(t, out, "| access queue user |")
	assert.Contains(t, out, "本项目的 go.mod 直接依赖数为 **64** 个")
	assert.Contains(t, out, "（`"+modulePath+"/...`）")
	assert.Contains(t, out, "不含主模块 `"+modulePath+"` 自身")
	assert.Contains(t, out, "形态在**生成期**固定")

	// 多形态口吻与失效引用一律不得出现。
	for _, gone := range []string{
		"相对 full", "overlay", "五个形态", "README", "TestMinimalCompiledSurfaceIsMateriallySmaller",
		"各形态", "full",
	} {
		assert.NotContains(t, out, gone, "单形态报告不得出现 %q", gone)
	}
}

// TestPercentAndMB 归一化列与 MB 呈现的边界：base 为 0 时不得除零，末位按一位小数四舍五入。
func TestPercentAndMB(t *testing.T) {
	assert.Equal(t, "—", percent(1, 0))
	assert.Equal(t, "50.0%", percent(1, 2))
	assert.Equal(t, "0.0", mb(0))
	assert.Equal(t, "1.5", mb(1_500_000))
}

// TestOverlayForProfileMatchesTheSharedPackage 报告统计闭包用的内存 overlay 必须与共享包
// tools/internal/profileoverlay 的输出逐字节相同：报告若再长出模板副本，度量的就不再是
// 出货二进制。
func TestOverlayForProfileMatchesTheSharedPackage(t *testing.T) {
	root := t.TempDir()
	shared, err := profileoverlay.ReplaceMap(root, "minimal")
	require.NoError(t, err)
	got, err := overlayForProfile(root, "minimal")
	require.NoError(t, err)
	assert.Equal(t, shared, got)

	_, err = overlayForProfile(root, "ghost")
	require.ErrorContains(t, err, `unknown profile "ghost"`)
}

// TestWriteOverlayMatchesTheSharedPackage 报告构建二进制用的 overlay JSON 也由共享包按形态
// 隔离写出（`.overlay/<profile>/`）：路径、active.go 内容与 Replace 映射逐字节一致。
func TestWriteOverlayMatchesTheSharedPackage(t *testing.T) {
	root := t.TempDir()
	path, err := writeOverlay(root, "enterprise")
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(profileoverlay.Dir(root, "enterprise"), "overlay.json"), path)

	src, err := profileoverlay.Source("enterprise")
	require.NoError(t, err)
	active, err := os.ReadFile(filepath.Join(profileoverlay.Dir(root, "enterprise"), "active.go"))
	require.NoError(t, err)
	assert.Equal(t, src, string(active))

	content, err := os.ReadFile(path)
	require.NoError(t, err)
	var cfg struct{ Replace map[string]string }
	require.NoError(t, json.Unmarshal(content, &cfg))
	assert.Equal(t,
		filepath.Join(profileoverlay.Dir(root, "enterprise"), "active.go"),
		cfg.Replace[filepath.Join(root, "internal", "profiles", "active", "assembly.go")])

	_, err = writeOverlay(root, "ghost")
	require.ErrorContains(t, err, `unknown profile "ghost"`)
}

// TestMinimalCompiledSurfaceIsMateriallySmaller 是 Task 7 的验收断言：minimal 的二进制、
// 路由、表与本仓代码量必须显著低于 full。断言的是实测关系（相对比例），不写死任何数字，
// 因此内核膨胀或能力增减都不会让用例误报 —— 只会让真正的裁剪失效暴露出来。
func TestMinimalCompiledSurfaceIsMateriallySmaller(t *testing.T) {
	if testing.Short() {
		t.Skip("构建 5 个形态二进制，-short 下跳过")
	}
	if os.Getenv(profileShardEnv) != "" {
		t.Skip("按形态切片：" + profileShardEnv + " 已设，本次由 TestProfileCompiledSurface 度量；跨形态关系由未设该变量时的完整度量承担")
	}
	ms, err := measureAll(repoRoot(t), nil)
	require.NoError(t, err)
	require.Len(t, ms, len(registry.Names()))

	byName := make(map[string]Metrics, len(ms))
	for _, m := range ms {
		byName[m.Profile] = m
	}
	fullM, ok := byName["full"]
	require.True(t, ok)
	minM, ok := byName["minimal"]
	require.True(t, ok)

	require.Positive(t, fullM.BinaryBytes)
	require.Positive(t, fullM.Routes)
	require.Positive(t, fullM.Tables)
	require.Positive(t, fullM.Files)

	// 二进制至少小 15%（实测约 -30%）；路由、表、本仓文件数必须严格更小。
	assert.LessOrEqual(t, float64(minM.BinaryBytes), 0.85*float64(fullM.BinaryBytes),
		"minimal 二进制应至少比 full 小 15%%（full %d, minimal %d）", fullM.BinaryBytes, minM.BinaryBytes)
	assert.Less(t, minM.Routes, fullM.Routes, "minimal 路由数应少于 full")
	assert.Less(t, minM.Tables, fullM.Tables, "minimal 表数应少于 full")
	assert.Less(t, minM.Files, fullM.Files, "minimal 本仓 Go 文件数应少于 full")
	assert.Less(t, minM.Lines, fullM.Lines, "minimal 本仓代码行数应少于 full")

	// 驱动拆包后的重型依赖列：full 编进全部四类驱动，其余形态的编译面为零
	//（可插拔的实际效果；enterprise 已收敛为 local + csv）。
	assert.ElementsMatch(t, heavydeps.Names(), fullM.HeavyDeps, "full 应含全部四类重型依赖")
	for _, name := range []string{"minimal", "saas", "enterprise", "machine"} {
		m, ok := byName[name]
		require.True(t, ok, "报告缺少形态 %s", name)
		assert.Empty(t, m.HeavyDeps, "%s 不应把重型依赖编进编译面", name)
	}
}

// TestProfileCompiledSurface 是上面那条完整度量的**单形态切片**（见 profileShardEnv）：
// CI 的 race 分片用 JIMU_METRICS_PROFILES=<p1,p2> 指定它负责的形态，只断言**该形态自身可独立判定**
// 的性质；跨形态关系留在完整度量里（默认路径的 Test job 跑它）。形态名拼错会让 measureAll 报错，
// 不会静默少测。
func TestProfileCompiledSurface(t *testing.T) {
	if testing.Short() {
		t.Skip("构建形态二进制，-short 下跳过")
	}
	raw := os.Getenv(profileShardEnv)
	if raw == "" {
		t.Skip("按形态切片需要 " + profileShardEnv + "=<p1,p2>；完整度量见 TestMinimalCompiledSurfaceIsMateriallySmaller")
	}
	profiles := strings.Split(raw, ",")
	ms, err := measureAll(repoRoot(t), profiles)
	require.NoError(t, err)
	require.Len(t, ms, len(profiles))

	for _, m := range ms {
		require.Positive(t, m.BinaryBytes, "%s 的二进制大小必须实测到", m.Profile)
		require.Positive(t, m.Routes, "%s 必须注册到路由", m.Profile)
		require.Positive(t, m.Tables, "%s 必须归属到表", m.Profile)
		require.Positive(t, m.Files, "%s 的闭包必须含本模块文件", m.Profile)
		require.Positive(t, m.Lines, "%s 的闭包行数必须为正", m.Profile)
		if m.Profile == "full" {
			assert.ElementsMatch(t, heavydeps.Names(), m.HeavyDeps, "full 应含全部四类重型依赖")
		} else {
			assert.Empty(t, m.HeavyDeps, "%s 不应把重型依赖编进编译面", m.Profile)
		}
	}
}

// TestMaskVolatileCellsPlatformColumns 钉住门禁的平台中立性：二进制大小与「相对 full」比例是
// **平台相关**列（同一份代码在 darwin/arm64 与 linux/amd64 上不同），掩码后两份「只差这些列」的
// 报告必须相等 —— 否则这个门禁只能在作者本机通过（CI 上必红，实测过）；而平台无关列
// （路由/迁移/表/本仓文件数与代码行/重型依赖）变化必须仍被判为不一致。
func TestMaskVolatileCellsPlatformColumns(t *testing.T) {
	a := "" +
		"| 形态 | 二进制 (MB) | 相对 full | 路由数 | 迁移数 | 表数 | 本仓 Go 文件 | 本仓代码行 | 重型依赖 |\n" +
		"| `full` | 123.1 | 100.0% | 99 | 25 | 23 | 331 | 34543 | aws-sdk-go-v2 |\n" +
		"| `minimal` | 84.7 | 68.8% | 32 | 10 | 9 | 180 | 17532 | - |\n" +
		"- 二进制：`minimal` 是 `full` 的 68.8%（要求 ≤ 85%）\n"
	b := strings.NewReplacer("123.1", "118.4", "84.7", "79.2", "68.8%", "66.9%").Replace(a)
	assert.Equal(t, maskVolatileCells(a), maskVolatileCells(b), "只差二进制列的报告掩码后必须相等（跨平台）")
	assert.NotEqual(t, maskVolatileCells(a), maskVolatileCells(strings.Replace(a, "| 99 |", "| 98 |", 1)),
		"平台无关列（路由）变化必须仍判为不一致")
	assert.NotEqual(t, maskVolatileCells(a), maskVolatileCells(strings.Replace(a, "| 34543 |", "| 34544 |", 1)),
		"平台无关列（本仓代码行）变化必须仍判为不一致")
}

// TestCheckCommittedReportsTheFirstDriftLine 钉住 -check 的报错形态：指出首个差异行与两侧内容；
// 只差二进制列时不报错（那条列本来就是平台相关的归档数据）。
func TestCheckCommittedReportsTheFirstDriftLine(t *testing.T) {
	dir := t.TempDir()
	committed := filepath.Join(dir, "compose-report.md")
	fresh := "| 形态 | 二进制 (MB) | 相对 full | 路由数 |\n| `full` | 1.0 | 100.0% | 99 |\n"

	require.NoError(t, os.WriteFile(committed, []byte(strings.Replace(fresh, "| 99 |", "| 98 |", 1)), 0o644))
	err := checkCommitted(committed, fresh)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "第 2 行")
	assert.Contains(t, err.Error(), "入库:")
	assert.Contains(t, err.Error(), "实测:")
	assert.Contains(t, err.Error(), "make compose-report", "报错里要给修复入口")

	require.NoError(t, os.WriteFile(committed, []byte(strings.Replace(fresh, "1.0", "0.9", 1)), 0o644))
	require.NoError(t, checkCommitted(committed, fresh), "只有二进制列不同时不该报错")
}

// repoRoot 返回仓库根（本文件位于 tools/composereport/）。
func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	require.True(t, ok)
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}
