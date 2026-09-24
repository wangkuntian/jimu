package main

import (
	"testing"

	"jimu/internal/capabilities/catalog"
	"jimu/internal/profiles/registry"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 提交态默认 full：声明集必须与 catalog.All() **同序同名，且关键字段逐值一致**
// （这是「full 行为零变化」的钉子）。
//
// 不能整结构 assert.Equal(catalog.All(), got)：Descriptor.Configs 里的 ConfigSpec.New 是
// 函数值，reflect.DeepEqual 对非 nil 函数恒 false（与 embed.FS 无关，实测确认）。
// 故逐条比对可比较的字段：Name（顺序）、Requires、Owns、Permissions。
func TestActiveDescriptorsForFullEqualsCatalogOrder(t *testing.T) {
	got, err := activeDescriptors()
	require.NoError(t, err)

	want := catalog.All()
	require.Len(t, got, len(want))
	for i := range want {
		assert.Equal(t, want[i].Name, got[i].Name, "第 %d 项能力名/顺序", i)
		assert.Equal(t, want[i].Requires, got[i].Requires, "能力 %s 的 Requires", want[i].Name)
		assert.Equal(t, want[i].Owns, got[i].Owns, "能力 %s 的 Owns", want[i].Name)
		assert.Equal(t, want[i].Permissions, got[i].Permissions, "能力 %s 的 Permissions", want[i].Name)
	}
}

// 非 full 形态是 catalog 的**保序子集**：只留该形态声明的能力，顺序仍按 catalog。
func TestResolveActiveKeepsCatalogOrderForSubset(t *testing.T) {
	minimal, err := registry.Lookup("minimal")
	require.NoError(t, err)

	declared := map[string]bool{}
	for _, c := range minimal.Capabilities {
		declared[c.Descriptor.Name] = true
	}
	got, err := resolveActive("minimal", declared)
	require.NoError(t, err)

	// 期望**人工写死**（不用与实现相同的循环，否则断言自指）：
	// minimal 声明 user/access/auth（另两个 encryption/notification 非 catalog，不参与）；
	// 迁移集再补 schema 依赖 tenant（users/roles.tenant_id 由它的迁移添加，见 catalog.MigrationSchemaDeps）。
	// catalog 序 = user, access, tenant, auth；而 minimal 的**装配序**是 encryption,
	// notification, access, user, auth —— 两者不同，正可判别是否走错顺序。
	want := []string{"user", "access", "tenant", "auth"}
	gotNames := make([]string, 0, len(got))
	for _, d := range got {
		gotNames = append(gotNames, d.Name)
	}
	assert.Equal(t, want, gotNames)
	assert.Contains(t, gotNames, "tenant", "含 user/access 的迁移集必须带 schema 依赖 tenant")
	assert.Contains(t, gotNames, "auth")

	// 真子集：确实比 catalog 少（否则这条用例是恒真的）。
	assert.Less(t, len(gotNames), len(catalog.Names()))
}

// 迁移集必须带上 schema 依赖：minimal 声明 user/access/auth（不含 tenant），
// 但 tenant 的迁移给 users/roles 加了 tenant_id 列（user/access 的 ORM 模型始终写它），
// 所以迁移集必须把 tenant 补进来 —— 否则裁剪形态会建出自己写不进去的 schema。
func TestMigrationSetCarriesSchemaDependencies(t *testing.T) {
	minimal, err := registry.Lookup("minimal")
	require.NoError(t, err)
	declared := map[string]bool{}
	for _, c := range minimal.Capabilities {
		declared[c.Descriptor.Name] = true
	}
	require.False(t, declared["tenant"], "minimal 本身不含 tenant")

	names := make([]string, 0, len(declared))
	for _, d := range catalog.MigrationSet(declared) {
		names = append(names, d.Name)
	}
	assert.Contains(t, names, "tenant", "含 user/access 的形态必须带上 tenant 的迁移")
	// 顺序仍是 catalog 序（tenant 在 user/access 之后）。
	assert.Less(t, indexOf(names, "user"), indexOf(names, "tenant"))
	assert.Less(t, indexOf(names, "access"), indexOf(names, "tenant"))

	// machine 只有 user/access/apikey/grpc/encryption：同样补 tenant。
	machine, err := registry.Lookup("machine")
	require.NoError(t, err)
	mDeclared := map[string]bool{}
	for _, c := range machine.Capabilities {
		mDeclared[c.Descriptor.Name] = true
	}
	mNames := make([]string, 0, len(mDeclared))
	for _, d := range catalog.MigrationSet(mDeclared) {
		mNames = append(mNames, d.Name)
	}
	assert.Contains(t, mNames, "tenant")
}

func indexOf(xs []string, want string) int {
	for i, x := range xs {
		if x == want {
			return i
		}
	}
	return -1
}

// 有 catalog 能力但一个迁移都没有 → 同样是「迁移零个能力」，必须 fail-closed
// （catalog 里 console/captcha/feature/uploadsec/breach 的 Migrations 为 nil）。
func TestResolveActiveRejectsProfileWithoutMigrations(t *testing.T) {
	_, err := resolveActive("consoleonly", map[string]bool{"console": true})
	require.ErrorIs(t, err, errNoCatalogCapabilities)
	require.ErrorContains(t, err, "has no capability with migrations")
}

// 空声明集必须 fail-closed（静默「迁移零个能力」比报错危险）。
func TestResolveActiveRejectsEmptyDeclared(t *testing.T) {
	_, err := resolveActive("ghost", map[string]bool{})
	require.ErrorContains(t, err, "declares no catalog capabilities")
	require.ErrorContains(t, err, `"ghost"`)
}
