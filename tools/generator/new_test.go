package generator

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewProjectWritesNothingOnDryRun(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "proj")
	_, err := newProjectForTest(t, NewOptions{Dir: dir, Profile: "minimal", Module: "example.com/proj", DryRun: true})
	require.NoError(t, err)
	_, statErr := os.Stat(dir)
	assert.True(t, os.IsNotExist(statErr))
}

func TestNewProjectLeavesNoPartialTreeOnFailure(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "proj")
	// 目标目录已存在且非空 → 报错，且不得留下 proj.tmp-* 残留。
	require.NoError(t, os.MkdirAll(dir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "keep.txt"), []byte("x"), 0o644))
	_, err := newProjectForTest(t, NewOptions{Dir: dir, Profile: "minimal", Module: "example.com/proj", NoTidy: true})
	require.ErrorContains(t, err, "not empty")
	entries, rerr := os.ReadDir(filepath.Dir(dir))
	require.NoError(t, rerr)
	for _, e := range entries {
		assert.NotContains(t, e.Name(), ".tmp-")
	}
}

// TestNewProjectAcceptsExistingEmptyDirectory 裁定 8：只拒绝「存在且**非空**」的目标 ——
// 已存在的空目录必须放行（`mkdir proj && jimu new proj` 是常见用法，也不需要 --force）。
// Fix round 2 之前这里会误报 "is not empty"（代码与注释/裁定矛盾）。
func TestNewProjectAcceptsExistingEmptyDirectory(t *testing.T) {
	for _, force := range []bool{false, true} {
		t.Run(fmt.Sprintf("force=%v", force), func(t *testing.T) {
			dir := filepath.Join(t.TempDir(), "proj")
			require.NoError(t, os.MkdirAll(dir, 0o755))
			res, err := newProjectForTest(t, NewOptions{Dir: dir, Profile: "minimal", Module: "example.com/proj", Force: force, NoTidy: true})
			require.NoError(t, err, "已存在的空目录必须放行")
			assert.FileExists(t, filepath.Join(dir, "go.mod"))
			assert.Positive(t, res.FileCount)
			entries, rerr := os.ReadDir(filepath.Dir(dir))
			require.NoError(t, rerr)
			for _, e := range entries {
				assert.NotContains(t, e.Name(), ".tmp-")
				assert.NotContains(t, e.Name(), ".old-")
			}
		})
	}
}

// TestNewProjectForceOnlyOverwritesGeneratorProducts --force 的语义是「覆盖**生成器产物**」：
// 非空且没有 .jimu-generated 标记的目录即使带 --force 也拒绝（用户数据不被碰）。
func TestNewProjectForceOnlyOverwritesGeneratorProducts(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "proj")
	require.NoError(t, os.MkdirAll(dir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "keep.txt"), []byte("user data"), 0o644))
	_, err := newProjectForTest(t, NewOptions{Dir: dir, Profile: "minimal", Module: "example.com/proj", Force: true, NoTidy: true})
	// 非空 + 无 .jimu-generated 标记 → 不是生成器产物，--force 不生效。
	require.ErrorContains(t, err, ".jimu-generated")
	content, rerr := os.ReadFile(filepath.Join(dir, "keep.txt"))
	require.NoError(t, rerr)
	assert.Equal(t, "user data", string(content), "拒绝时不得动目标目录")
}

// TestNewProjectCopiesKernelAndSelectedCapabilities 是「复制与落盘」这一层的落地验收：
// 内核必需目录齐全、能力目录按 S1 复制集落地、驱动目录按选中集过滤、迁移携带目录只带
// migrations/ + 生成的 module.go、module 前缀已重写。
func TestNewProjectCopiesKernelAndSelectedCapabilities(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "proj")
	res, err := newProjectForTest(t, NewOptions{Dir: dir, Profile: "minimal", Module: "example.com/proj", NoTidy: true})
	require.NoError(t, err)
	require.Equal(t, "example.com/proj", res.Module)
	assert.Positive(t, res.FileCount)

	for _, p := range []string{
		"go.mod", ".golangci.yml", ".gitignore", "conf/rbac_model.conf",
		"internal/kernel/tenant/tenant.go", "internal/contract/module.go",
		"internal/capability/resolve.go", "internal/assembly/assembly.go",
		"internal/profiles/profiles.go", "cmd/server/main.go", "cmd/cli/main.go",
		"internal/profiles/registry/registry.go",
		"internal/profiles/minimal/assembly.go", "internal/profiles/active/assembly.go",
		"internal/capabilities/catalog/catalog.go",
	} {
		assert.FileExists(t, filepath.Join(dir, p), "缺少必需产物 %s", p)
	}

	// S1 复制集：只包含已声明的能力和必要的 schema 依赖。
	for _, capName := range []string{"user", "access", "auth", "encryption", "notification"} {
		assert.FileExists(t, filepath.Join(dir, "internal/capabilities", capName, "wire.go"), "能力 %s 未复制", capName)
	}
	// 未选中能力零出现。
	for _, absent := range []string{"mfa", "passkey", "console", "audit", "storage", "dataops"} {
		_, statErr := os.Stat(filepath.Join(dir, "internal/capabilities", absent))
		assert.True(t, os.IsNotExist(statErr), "未选中能力 %s 不应出现", absent)
	}
	// 驱动级过滤：minimal 未给 queue 选任何驱动 → 无 redis/kafka/rabbitmq 子目录。
	for _, drv := range []string{"redis", "kafka", "rabbitmq"} {
		_, statErr := os.Stat(filepath.Join(dir, "internal/capabilities/queue", drv))
		assert.True(t, os.IsNotExist(statErr), "未选中驱动 queue/%s 不应出现", drv)
	}

	// S2 迁移携带：tenant 带 migrations/ + domain/（app/seed 的编译期依赖）+ 生成的 module.go；
	// application/infrastructure/interfaces/wire/cli 仍不复制。
	assert.DirExists(t, filepath.Join(dir, "internal/capabilities/tenant/migrations"))
	assert.DirExists(t, filepath.Join(dir, "internal/capabilities/tenant/domain"))
	assert.FileExists(t, filepath.Join(dir, "internal/capabilities/tenant/module.go"))
	for _, absent := range []string{"wire.go", "cli", "application", "infrastructure", "interfaces"} {
		_, statErr := os.Stat(filepath.Join(dir, "internal/capabilities/tenant", absent))
		assert.True(t, os.IsNotExist(statErr), "迁移携带目录不应有 %s", absent)
	}

	// go.mod 的 module 指令已重写，且生成的 tenant/module.go 自带 Descriptor。
	gomod, err := os.ReadFile(filepath.Join(dir, "go.mod"))
	require.NoError(t, err)
	assert.Contains(t, string(gomod), "module example.com/proj")
	tenantModule, err := os.ReadFile(filepath.Join(dir, "internal/capabilities/tenant/module.go"))
	require.NoError(t, err)
	assert.Contains(t, string(tenantModule), `Name:       "tenant"`)
	assert.Contains(t, string(tenantModule), "Owns:")
	assert.Contains(t, string(tenantModule), "//go:embed migrations")

	// 全树不得残留框架 module path（精确判据：protobuf rawDesc/Metadata 里的 proto 文件名显式放行）。
	assertNoStaleModulePath(t, dir)

	// .jimu-generated 标记（--force 的识别依据）。
	assert.FileExists(t, filepath.Join(dir, markerFile))
}

// TestNewProjectWithSelectsFirstDriverAndFiltersDirectories 钉住 S4 + 驱动目录过滤：
// --with=queue 默认 redis → redis 目录在、kafka/rabbitmq 不在，且 Descriptor.Drivers 被收窄，
// 否则生成项目的 check-capabilities 断言①（声明的驱动目录必须存在）必红。
func TestNewProjectWithSelectsFirstDriverAndFiltersDirectories(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "proj")
	_, err := newProjectForTest(t, NewOptions{Dir: dir, With: "queue", Module: "example.com/proj", NoTidy: true})
	require.NoError(t, err)
	assert.FileExists(t, filepath.Join(dir, "internal/capabilities/queue/redis/redis_queue.go"))
	for _, drv := range []string{"kafka", "rabbitmq"} {
		_, statErr := os.Stat(filepath.Join(dir, "internal/capabilities/queue", drv))
		assert.True(t, os.IsNotExist(statErr), "未选中驱动 queue/%s 不应出现", drv)
	}
	src, err := os.ReadFile(filepath.Join(dir, "internal/capabilities/queue/migrations.go"))
	require.NoError(t, err)
	assert.Contains(t, string(src), `Drivers:    []string{"redis"}`)

	// 未被选中的能力（storage 声明 local/s3，但既不声明也未选驱动）不复制任何驱动目录。
	_, statErr := os.Stat(filepath.Join(dir, "internal/capabilities/storage/local"))
	assert.True(t, os.IsNotExist(statErr))
}

// TestNewProjectForceOverwritesMarkedProduct 覆盖 --force 的正路径：带标记的目录被整体替换。
func TestNewProjectForceOverwritesMarkedProduct(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "proj")
	_, err := newProjectForTest(t, NewOptions{Dir: dir, Profile: "minimal", Module: "example.com/proj", NoTidy: true})
	require.NoError(t, err)
	stale := filepath.Join(dir, "stale.txt")
	require.NoError(t, os.WriteFile(stale, []byte("old"), 0o644))

	_, err = newProjectForTest(t, NewOptions{Dir: dir, Profile: "minimal", Module: "example.com/proj", Force: true, NoTidy: true})
	require.NoError(t, err)
	_, statErr := os.Stat(stale)
	assert.True(t, os.IsNotExist(statErr), "--force 必须整体替换旧产物")

	entries, rerr := os.ReadDir(filepath.Dir(dir))
	require.NoError(t, rerr)
	for _, e := range entries {
		assert.NotContains(t, e.Name(), ".tmp-")
		assert.NotContains(t, e.Name(), ".old-")
	}
}

// TestNewProjectDryRunCountMatchesRealTree 把 --dry-run 的「将复制/渲染的文件数」钉成**精确上界**：
// 它是用户判断「这个选择会产出多少文件」的唯一依据，也是复制集口径（S1/S2/裁定④）的一条端到端
// 回归网 —— 任一处的统计口径漂移（例如把只带 domain/ 的能力当成整目录复制）都会让两侧不等。
//
// 精确关系：planned = real + len(discardedTests)。差额**只**来自 Important 4 的测试裁剪
// （dry-run 不落盘、无法预知哪些测试 import 得不到满足），所以差值必须与 marker 里的
// discardedTests 数量严格相等 —— 这比「相等」约束更强：多算/少算一个文件都会红。
func TestNewProjectDryRunCountMatchesRealTree(t *testing.T) {
	cases := []struct {
		name string
		opts NewOptions
	}{
		{"with queue（只带 domain 的内核依赖）", NewOptions{With: "queue"}},
		{"with user,access（schema 依赖 tenant）", NewOptions{With: "user,access"}},
		{"profile minimal", NewOptions{Profile: "minimal"}},
		// apidocs/full 会额外携带 docs/openapi（T6 的资产复制）：曾被 dry-run 漏计，使 planned < real。
		{"with apidocs", NewOptions{With: "apidocs"}},
		{"profile full", NewOptions{Profile: "full"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			base := t.TempDir()
			dry := filepath.Join(base, "dry")
			opts := tc.opts
			opts.Dir = dry
			opts.Module = "example.com/proj"
			opts.NoTidy = true
			opts.DryRun = true
			planned, err := newProjectForTest(t, opts)
			require.NoError(t, err)
			_, statErr := os.Stat(dry)
			require.True(t, os.IsNotExist(statErr), "--dry-run 不得落盘")

			opts.DryRun = false
			opts.Dir = filepath.Join(base, "real")
			real, err := newProjectForTest(t, opts)
			require.NoError(t, err)
			marker := readMarkerForTest(t, opts.Dir)
			assert.Equal(t, real.FileCount+len(marker.DiscardedTests), planned.FileCount,
				"planned = real + 被裁剪的测试文件数（dry-run 是裁剪前的上界）")
			assert.Len(t, real.Files, real.FileCount)
			assert.NotContains(t, real.Files, markerFile)
			assert.Equal(t, real.Files, marker.Files, "marker.files 必须是本次落盘的完整清单")
		})
	}
}

// TestSwapIntoPlaceRestoresTargetOnRenameFailure 钉住 --force 的「失败时换回原目录」分支：
// 旧产物已被挪成 <dir>.old-<rand>，若新产物没能就位（rename 失败），必须把旧目录换回来，
// 绝不留下「目标不存在」的窗口（T8 复用本语义）。
func TestSwapIntoPlaceRestoresTargetOnRenameFailure(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "proj")
	require.NoError(t, os.MkdirAll(target, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(target, "keep.txt"), []byte("old"), 0o644))
	// tmp 不存在 → os.Rename(tmp, target) 必然失败，正好走 restore 分支。
	missingTmp := filepath.Join(dir, "proj.tmp-missing")

	err := swapIntoPlace(target, missingTmp)
	require.ErrorContains(t, err, "install new product")

	content, rerr := os.ReadFile(filepath.Join(target, "keep.txt"))
	require.NoError(t, rerr, "旧产物必须被换回原路径")
	assert.Equal(t, "old", string(content))

	entries, rerr := os.ReadDir(dir)
	require.NoError(t, rerr)
	for _, e := range entries {
		assert.NotContains(t, e.Name(), ".old-", "不得留下被挪走的旧目录")
		assert.NotContains(t, e.Name(), ".tmp-")
	}
}

// TestNewProjectRejectsCollidingModule 把 RewriteModule 的 fail-closed 转成面向用户的 --module 提示。
func TestNewProjectRejectsCollidingModule(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "proj")
	_, err := newProjectForTest(t, NewOptions{Dir: dir, Profile: "minimal", Module: "github.com/foo/jimu", NoTidy: true})
	require.ErrorContains(t, err, "--module")
	_, err = newProjectForTest(t, NewOptions{Dir: dir, Profile: "minimal", Module: "github.com/foo/jimu/v2", NoTidy: true})
	require.ErrorContains(t, err, "--module")
	_, statErr := os.Stat(dir)
	assert.True(t, os.IsNotExist(statErr), "参数校验失败时不得留下产物")
}

// TestParseCapabilitySetClosureDoesNotDragMigrationOnlyIntoCode 反证 S2：tenant 的编译闭包含
// auth 链，若把迁移携带目录当声明集展开就会把 auth 拖进 --with=user,access 的项目。
func TestParseCapabilitySetClosureDoesNotDragMigrationOnlyIntoCode(t *testing.T) {
	set, err := ParseCapabilitySet("", "user,access", "app")
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"user", "access", "tenant"}, set.Copy)
	assert.Equal(t, []string{"tenant"}, set.MigrationOnly)
	assert.Equal(t, set.Copy, slices.Sorted(slices.Values(set.Copy)), "Copy 必须排序且无重复")
	assert.NotContains(t, set.Copy, "auth")
}

// readMarkerForTest 读生成项目的 .jimu-generated。
func readMarkerForTest(t *testing.T, dir string) Marker {
	t.Helper()
	content, err := os.ReadFile(filepath.Join(dir, markerFile))
	require.NoError(t, err)
	var m Marker
	require.NoError(t, json.Unmarshal(content, &m))
	return m
}
