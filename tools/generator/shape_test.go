package generator

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRenderShapeWritesSingleShapeRegistryAndAssembly 钉住单形态渲染的确定形态：
// registry 只含一个形态、assembly 的能力别名一律 <name>module、Ungated/Drivers 逐项写出、
// active 只 import 那个唯一形态。
func TestRenderShapeWritesSingleShapeRegistryAndAssembly(t *testing.T) {
	dst := t.TempDir()
	set := CapabilitySet{
		Shape:    "minimal",
		Profile:  "minimal",
		Declared: []string{"user", "access", "auth", "encryption", "notification"},
		Copy:     []string{"access", "auth", "encryption", "notification", "outbox", "queue", "tenant", "user"},
		Ungated:  []string{"encryption", "notification"},
		Drivers:  map[string][]string{"queue": {"redis"}},
	}
	require.NoError(t, RenderShape("", dst, set))

	registry, err := os.ReadFile(filepath.Join(dst, "internal/profiles/registry/registry.go"))
	require.NoError(t, err)
	assert.Contains(t, string(registry), `var names = []string{"minimal"}`)
	assert.Contains(t, string(registry), `func All() map[string]assembly.Assembly`)
	assert.Contains(t, string(registry), `"minimal": minimal.Assembly(),`)

	assembly, err := os.ReadFile(filepath.Join(dst, "internal/profiles/minimal/assembly.go"))
	require.NoError(t, err)
	src := string(assembly)
	assert.Contains(t, src, `encryptionmodule "jimu/internal/capabilities/encryption"`)
	assert.Contains(t, src, "Descriptor: encryptionmodule.Descriptor, Wire: encryptionmodule.Wire, Ungated: true")
	assert.Contains(t, src, "Descriptor: usermodule.Descriptor, Wire: usermodule.Wire")
	assert.Contains(t, src, `Name:    "minimal"`)
	assert.Contains(t, src, "Seed: profiles.StructuralSeed")
	// 迁移携带能力（tenant）不参与装配。
	assert.NotContains(t, src, "tenant")

	drivers, err := os.ReadFile(filepath.Join(dst, "internal/profiles/minimal/drivers.go"))
	require.NoError(t, err)
	assert.Contains(t, string(drivers), "package minimal")

	active, err := os.ReadFile(filepath.Join(dst, "internal/profiles/active/assembly.go"))
	require.NoError(t, err)
	assert.Contains(t, string(active), `"jimu/internal/profiles/minimal"`)
	assert.Contains(t, string(active), "func Assembly() assembly.Assembly { return minimal.Assembly() }")
}

// TestRenderShapeWritesSelectedDrivers 驱动清单逐项进入 drivers.go 的 blank import，
// 且 assembly 里写上对应的 Drivers 子集（S4）。
func TestRenderShapeWritesSelectedDrivers(t *testing.T) {
	dst := t.TempDir()
	set := CapabilitySet{
		Shape:    "app",
		Declared: []string{"queue"},
		Copy:     []string{"queue"},
		Drivers:  map[string][]string{"queue": {"kafka"}},
	}
	require.NoError(t, RenderShape("", dst, set))
	drivers, err := os.ReadFile(filepath.Join(dst, "internal/profiles/app/drivers.go"))
	require.NoError(t, err)
	assert.Contains(t, string(drivers), `_ "jimu/internal/capabilities/queue/kafka"`)
	assert.NotContains(t, string(drivers), "queue/redis")

	assembly, err := os.ReadFile(filepath.Join(dst, "internal/profiles/app/assembly.go"))
	require.NoError(t, err)
	assert.Contains(t, string(assembly), `Drivers: []string{"kafka"}`)
	// 渲染出的 .go 必须已经 gofmt（RenderText 对 .go 名字做格式化）。
	assert.NotContains(t, string(assembly), "\t\t\t\t")
}

// TestParseCapabilitySetValidatesShape 钉住 --shape 的 fail-closed 校验（Fix round 4）：
// 保留名（registry/active/…）会生成自 import 的包或让两份产物撞到同一路径，非标识符同理。
func TestParseCapabilitySetValidatesShape(t *testing.T) {
	for _, bad := range []string{"registry", "active", "catalog", "configs", "profiles", "internal", "cmd"} {
		t.Run("reserved/"+bad, func(t *testing.T) {
			_, err := ParseCapabilitySet("", "queue", bad)
			require.ErrorContains(t, err, "invalid --shape")
			require.ErrorContains(t, err, "保留名")
		})
	}
	for _, bad := range []string{"Bad", "my-app", "1app", "app ", "app/x"} {
		t.Run("identifier/"+bad, func(t *testing.T) {
			_, err := ParseCapabilitySet("", "queue", bad)
			require.ErrorContains(t, err, "invalid --shape")
		})
	}
	// 合法形态名（含默认值）照常放行。
	for _, ok := range []string{"app", "minimal", "machine", "my_app2"} {
		if _, err := ParseCapabilitySet("", "queue", ok); err != nil {
			t.Errorf("--shape=%s 应当放行：%v", ok, err)
		}
	}
	if _, err := ParseCapabilitySet("", "queue", ""); err != nil {
		t.Errorf("缺省 --shape 应当放行：%v", err)
	}
}

// TestNewProjectRejectsReservedShapeWithoutWriting --shape=registry 曾生成自 import 的 registry
// 包（rc=0 但项目不可编译）；现在必须报错且不落盘。
func TestNewProjectRejectsReservedShapeWithoutWriting(t *testing.T) {
	for _, bad := range []string{"registry", "active"} {
		t.Run(bad, func(t *testing.T) {
			dir := filepath.Join(t.TempDir(), "proj")
			_, err := newProjectForTest(t, NewOptions{Dir: dir, With: "queue", Shape: bad, Module: "example.com/proj", NoTidy: true})
			require.ErrorContains(t, err, "invalid --shape")
			_, statErr := os.Stat(dir)
			assert.True(t, os.IsNotExist(statErr), "--shape 非法时不得落盘")
			entries, rerr := os.ReadDir(filepath.Dir(dir))
			require.NoError(t, rerr)
			for _, e := range entries {
				assert.NotContains(t, e.Name(), ".tmp-", "不得留下临时目录")
			}
		})
	}
}

// TestRenderShapeRejectsCollidingArtifacts 结构性地消除「静默丢模板」：即使用户绕过校验直接调用
// RenderShape（--shape=active），产物路径冲突也必须报错，而不是 map 字面量里后一个键覆盖前一个。
func TestRenderShapeRejectsCollidingArtifacts(t *testing.T) {
	dst := t.TempDir()
	err := RenderShape("", dst, CapabilitySet{Shape: "active", Declared: []string{"queue"}, Drivers: map[string][]string{}})
	require.ErrorContains(t, err, "冲突")
	// 已写下的前几个产物可以是部分的 —— RenderShape 只在 NewProject 的临时目录里被调用，
	// 失败由 <dir>.tmp-<rand> + defer RemoveAll 兜底（原子性在 T2 编排层，不在这里）。
	// 这里要保证的是：**碰撞必须报错**，绝不静默丢掉第 4 份产物。
	_, statErr := os.Stat(filepath.Join(dst, "internal/profiles/active/drivers.go"))
	assert.NoError(t, statErr, "碰撞检测发生在写第 4 份产物之前，drivers.go 应当已经写出")
}
