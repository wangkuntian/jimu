package generator

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"jimu/internal/config"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestFrameworkRootRespectsTheConfigSearchDepth 钉住 P2.8 的口径统一：frameworkRoot 的向上搜索层数
// 与 internal/config 找 `configs/` 的层数**是同一个常量**（`config.SearchDepthUp`，含 cwd 本身）。
//
// 统一前两处不一致：frameworkRoot 一路找到文件系统根（无上限），而配置加载只向上 5 层 —— 于是从
// 更深的目录跑 `jimu new --report` 会先成功发现源根、复制整棵树，再在 ProbeAssembly 加载能力配置段
// 时半路失败。现在超出层数即在源根发现处 fail-closed 报错，且错误里带层数与起点。
func TestFrameworkRootRespectsTheConfigSearchDepth(t *testing.T) {
	base := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(base, "go.mod"), []byte("module "+frameworkModule+"\n\ngo 1.26\n"), 0o644))

	// 深度 1..SearchDepthUp-1（相对框架根）仍在限制内：最深处应能找到源根
	deepest := base
	for i := 1; i < config.SearchDepthUp; i++ {
		deepest = filepath.Join(deepest, fmt.Sprintf("d%d", i))
	}
	require.NoError(t, os.MkdirAll(deepest, 0o755))
	t.Chdir(deepest)
	root, err := frameworkRoot()
	require.NoError(t, err, "限制内的深度（%d 层）应能找到框架根", config.SearchDepthUp-1)
	assert.Equal(t, base, root)

	// 再多一层即超出限制：fail-closed，而不是一路向上找到文件系统根
	tooDeep := filepath.Join(deepest, "extra")
	require.NoError(t, os.MkdirAll(tooDeep, 0o755))
	t.Chdir(tooDeep)
	_, err = frameworkRoot()
	require.Error(t, err, "超出 %d 层应报错", config.SearchDepthUp)
	assert.Contains(t, err.Error(), "no framework source root within")
	assert.Contains(t, err.Error(), "extra", "错误里应带上搜索起点便于定位")
}

func TestParseCapabilitySetRejectsNeitherFlag(t *testing.T) {
	_, err := ParseCapabilitySet("", "", "app")
	require.ErrorContains(t, err, "--profile")
	require.ErrorContains(t, err, "--with")
}

func TestParseCapabilitySetRejectsBothFlags(t *testing.T) {
	_, err := ParseCapabilitySet("minimal", "user", "app")
	require.ErrorContains(t, err, "mutually exclusive")
}

func TestParseCapabilitySetProfileAddsCompileAndSchemaDeps(t *testing.T) {
	set, err := ParseCapabilitySet("minimal", "", "")
	require.NoError(t, err)
	assert.Equal(t, "minimal", set.Shape)
	// 声明集 = catalog 条目（拓扑序）+ 非 catalog（Ungated）条目（P2.4 裁定：Ungated 不受门控）。
	assert.Equal(t, []string{"user", "access", "auth", "encryption", "notification"}, set.Declared)
	// S1：user/auth 的类型级残留（outbox/queue/notification/encryption）必须复制，否则 go build 失败；
	// tenant 是 schema 依赖的「迁移携带」目录（只带 migrations/ + 生成的 module.go，故不展开其编译闭包
	// —— 否则 auth 链会被一起拖进来）。
	assert.ElementsMatch(t, []string{"access", "auth", "encryption", "notification", "outbox", "queue", "tenant", "user"}, set.Copy)
	// S1/S2：user/access 的 schema 依赖 tenant 只作「迁移携带」。
	assert.Equal(t, []string{"tenant"}, set.MigrationOnly)
	assert.ElementsMatch(t, []string{"encryption", "notification"}, set.Ungated)
}

func TestParseCapabilitySetWithUsesResolveAndDefaultDrivers(t *testing.T) {
	set, err := ParseCapabilitySet("", "user,access,queue", "app")
	require.NoError(t, err)
	assert.Equal(t, "app", set.Shape)
	// S4：--with 的驱动默认取 Descriptor.Drivers 首项（queue→redis），故 kafka-go 不进依赖图。
	assert.Equal(t, []string{"redis"}, set.Drivers["queue"])
	assert.Contains(t, set.Copy, "outbox")
	assert.NotContains(t, set.Copy, "auth")
	// S1/S2：user/access 的 schema 依赖 tenant 只作迁移携带，不展开其编译闭包（否则拖进 auth）。
	assert.Equal(t, []string{"tenant"}, set.MigrationOnly)
	assert.Empty(t, set.Ungated)
}

func TestParseCapabilitySetWithExplicitDriverAndErrors(t *testing.T) {
	set, err := ParseCapabilitySet("", "queue:kafka", "app")
	require.NoError(t, err)
	assert.Equal(t, []string{"kafka"}, set.Drivers["queue"])

	_, err = ParseCapabilitySet("", "queue:ghost", "app")
	require.ErrorContains(t, err, `unknown driver "ghost"`)
	require.ErrorContains(t, err, "redis, kafka, rabbitmq")

	_, err = ParseCapabilitySet("", "queue:", "app")
	require.ErrorContains(t, err, "empty driver")

	_, err = ParseCapabilitySet("", "user,,queue", "app")
	require.ErrorContains(t, err, "empty capability")
}

func TestParseCapabilitySetRejectsUnknownWith(t *testing.T) {
	_, err := ParseCapabilitySet("", "ghost", "app")
	require.ErrorContains(t, err, `unknown capability "ghost"`)
}

func TestParseCapabilitySetRejectsUnknownProfile(t *testing.T) {
	_, err := ParseCapabilitySet("ghost", "", "app")
	require.ErrorContains(t, err, `unknown profile "ghost"`)
}

// TestCapabilityRootsClosureOfQueueOnly 钉住 S1+C1 的闭包口径：
//   - 能力根包 import 与**非 domain 子包** import 都映射回属主能力根包 → 整目录复制
//     （C1：auth → user/infrastructure 这类子包依赖必须整目录复制）；
//   - `<cap>/domain…` 子包**忽略**（由裁定 ④/S2 决定只带 domain/，否则 tenant/auth 链会被拖进来）；
//   - 完整复制成员再传递补齐 Requires。
//
// queue 自身经 `./queue` 根包进闭包；access/tenant/user 只出现在结果里，是裁定 ④ 的
// **内核编译期 domain 依赖**（internal/app/seed.go 直接 import 它们的 domain 叶子包），
// 且只带 domain/、不带 migrations 与生成的 module。
func TestCapabilityRootsClosureOfQueueOnly(t *testing.T) {
	set, err := ParseCapabilitySet("", "queue", "app")
	require.NoError(t, err)
	assert.Equal(t, []string{"queue"}, set.Declared)
	assert.Equal(t, []string{"access", "queue", "tenant", "user"}, set.Copy)
	assert.Equal(t, []string{"access", "tenant", "user"}, set.DomainOnly)
	assert.Empty(t, set.MigrationOnly, "queue 不触发 schema 依赖：tenant 是 domain 携带而非迁移携带")
	assert.NotContains(t, set.Copy, "auth")
}

// TestCapabilityRootsIsIdempotentOnComputedCopy 闭包已算好后再跑一次必须不变：
// CapabilityRoots 只从 Declared 出发，重入不会把闭包/迁移携带当成声明集再展开。
func TestCapabilityRootsIsIdempotentOnComputedCopy(t *testing.T) {
	first, err := CapabilityRoots(FrameworkRoot(), CapabilitySet{Shape: "app", Declared: []string{"user", "access"}})
	require.NoError(t, err)
	second, err := CapabilityRoots(FrameworkRoot(), first)
	require.NoError(t, err)
	assert.Equal(t, first.Copy, second.Copy)
	assert.Equal(t, first.MigrationOnly, second.MigrationOnly)
}
