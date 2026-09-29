package generator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 本文件是 Step 2 的落地验收：`jimu new --report` 的度量与生成的
// <dir>/docs/profiles/generated-report.md。度量原语（路由/迁移/表/闭包）的单测在
// tools/internal/projectmetrics，这里只钉「报告 = 本项目实测」与「重跑逐字节相同」。

// TestReportMeasuresTheGeneratedProject 报告必须能在已生成项目上独立重跑（从 marker 读回声明集），
// 且每一项都来自这棵树：闭包文件/代码行、解析集、迁移/表/路由都是正数。
func TestReportMeasuresTheGeneratedProject(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "proj")
	_, err := newProjectForTest(t, NewOptions{Dir: dir, With: "queue", Module: "example.com/proj", NoTidy: true})
	require.NoError(t, err)

	m, err := Report(dir)
	require.NoError(t, err)
	assert.Equal(t, "app", m.Profile, "--with 的默认形态名")
	assert.Equal(t, []string{"queue"}, m.Capabilities, "解析集只含声明集（outbox 依赖 queue，反向不成立）")
	assert.Positive(t, m.Files, "闭包文件数")
	assert.Positive(t, m.Lines, "闭包代码行")
	assert.Positive(t, m.Routes, "路由数")
	assert.Positive(t, m.Migrations, "迁移数")
	assert.Positive(t, m.Tables, "表数")
	assert.GreaterOrEqual(t, m.Lines, m.Files, "每个文件至少一行")
	assert.Empty(t, m.HeavyDeps, "queue 默认驱动是 redis，不含重型依赖")
}

// TestWriteReportWritesAndIsIdempotent 报告落点是 <dir>/docs/profiles/generated-report.md，
// 内容只由实测值与 marker 决定：同一项目重写两次必须逐字节相同（入库产物的幂等前提）。
func TestWriteReportWritesAndIsIdempotent(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "proj")
	_, err := newProjectForTest(t, NewOptions{Dir: dir, Profile: "minimal", Module: "example.com/proj", NoTidy: true})
	require.NoError(t, err)
	set, err := ParseCapabilitySet("minimal", "", "")
	require.NoError(t, err)
	m, err := Report(dir)
	require.NoError(t, err)

	require.NoError(t, WriteReport(dir, *m, set))
	path := filepath.Join(dir, filepath.FromSlash(reportRelPath))
	first, err := os.ReadFile(path)
	require.NoError(t, err)
	src := string(first)

	// 报告只在 --report 时落盘：先写一次再断言内容，然后重写一次验幂等。
	assert.Contains(t, src, "# 生成项目编译面报告")
	assert.Contains(t, src, "| 模块 | `example.com/proj` |")
	assert.Contains(t, src, "| 形态 | `minimal` |")
	assert.Contains(t, src, "| 迁移携带能力（schema 依赖） | tenant |")
	assert.Contains(t, src, "| 生成文件数 |")
	assert.Contains(t, src, "| 路由数 |")
	// 主模块名必须是项目自己：报告里不得出现「相对 full」列或构建期 overlay 叙述（裁定 15）。
	assert.NotContains(t, src, "相对 full")
	assert.NotContains(t, src, "构建期 overlay")
	assert.NotContains(t, src, "`jimu`")

	require.NoError(t, WriteReport(dir, *m, set))
	second, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, first, second, "报告必须幂等")
}

// TestNewProjectReportWritesFileAndKeepsItOutOfTheFileList --report 是 `jimu new` 的一部分：
// 文件落盘、Result.Files 不含报告自身（报告是「关于产物的产物」，不是产物）。
func TestNewProjectReportWritesFileAndKeepsItOutOfTheFileList(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "proj")
	res, err := newProjectForTest(t, NewOptions{Dir: dir, Profile: "minimal", Module: "example.com/proj", NoTidy: true, Report: true})
	require.NoError(t, err)
	assert.FileExists(t, filepath.Join(dir, filepath.FromSlash(reportRelPath)))
	assert.NotContains(t, res.Files, reportRelPath)
	content, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(reportRelPath)))
	require.NoError(t, err)
	assert.True(t, strings.HasSuffix(string(content), "\n"), "报告以换行收尾")
}

// TestReportSucceedsForEverySelection 是 `--report` 的系统网：25 个 `--with=<cap>` 选择逐个生成项目
// 后，Report 必须能算出每一项（它经 `assembly.ProbeAssembly` 求路由/迁移/表 —— 装配失败的选区会让
// `jimu new --report` 整体失败，不能只在 minimal/mfa 等少数选区上验证）。
//
// 不构建产物：只走 packages.Load 的 import 图，因此每个选区的成本 ≈ 一次项目复制 + 一次闭包度量。
func TestReportSucceedsForEverySelection(t *testing.T) {
	for _, name := range allCapabilityNames(t) {
		t.Run(name, func(t *testing.T) {
			dir := filepath.Join(t.TempDir(), "proj")
			_, err := newProjectForTest(t, NewOptions{Dir: dir, With: name, Module: "example.com/proj", NoTidy: true})
			require.NoError(t, err)
			m, err := Report(dir)
			require.NoError(t, err, "选区 %s 的 --report 必须可用", name)
			assert.Contains(t, m.Capabilities, name)
			assert.Positive(t, m.Files)
			assert.Positive(t, m.Lines)
			// 路由数可以为 0（如 storage/ws/retention 这类不注册 HTTP 路由的选区），但不得为负。
			assert.GreaterOrEqual(t, m.Routes, 0)
		})
	}
}

// TestReportRerunIsByteIdentical 独立重跑（`Report` + `WriteReport` 对已生成项目）必须与生成时写出的
// 报告**逐字节相同**：markerSet 经 CapabilityRoots 重算 Copy/MigrationOnly/DomainOnly（否则
// 「迁移携带能力 / 只带 domain 的能力」两行会退化成 `-`），且生成文件数不含报告自身。
func TestReportRerunIsByteIdentical(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "proj")
	_, err := newProjectForTest(t, NewOptions{Dir: dir, Profile: "minimal", Module: "example.com/proj", NoTidy: true, Report: true})
	require.NoError(t, err)
	path := filepath.Join(dir, filepath.FromSlash(reportRelPath))
	first, err := os.ReadFile(path)
	require.NoError(t, err)

	rerun, err := Report(dir)
	require.NoError(t, err)
	rerunSet, err := markerSet(FrameworkRoot(), mustMarker(t, dir))
	require.NoError(t, err)
	require.NoError(t, WriteReport(dir, *rerun, rerunSet))
	second, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, string(first), string(second), "独立重跑必须逐字节幂等")
	// --with=queue 会把 access/tenant/user 只作 domain 携带，重跑后该行不得退化成 `-`。
	qdir := filepath.Join(t.TempDir(), "projq")
	_, err = newProjectForTest(t, NewOptions{Dir: qdir, With: "queue", Module: "example.com/proj", NoTidy: true, Report: true})
	require.NoError(t, err)
	m, err := Report(qdir)
	require.NoError(t, err)
	set, err := markerSet(FrameworkRoot(), mustMarker(t, qdir))
	require.NoError(t, err)
	assert.Equal(t, []string{"access", "tenant", "user"}, set.DomainOnly)
	require.NoError(t, WriteReport(qdir, *m, set))
	content, err := os.ReadFile(filepath.Join(qdir, filepath.FromSlash(reportRelPath)))
	require.NoError(t, err)
	assert.Contains(t, string(content), "| 只带 domain 的能力（内核编译期依赖） | access, tenant, user |")
}

// mustMarker 读生成项目的 marker（测试辅助）。
func mustMarker(t *testing.T, dir string) *Marker {
	t.Helper()
	m, err := LoadMarker(dir)
	require.NoError(t, err)
	return m
}

// TestReportRejectsForeignDirectory 未生成的目录没有 .jimu-generated → 明确报错（fail-closed），
// 绝不按猜测度量任意目录。
func TestReportRejectsForeignDirectory(t *testing.T) {
	_, err := Report(t.TempDir())
	require.ErrorContains(t, err, markerFile)
}
