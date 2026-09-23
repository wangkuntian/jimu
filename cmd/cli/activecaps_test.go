package main

import (
	"testing"

	"jimu/internal/capabilities/catalog"
	"jimu/internal/contract"
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
	// minimal 声明 user/access/auth（另两个 encryption/notification 非 catalog，不参与）。
	// catalog 序 = user, access, auth；而 minimal 的**装配序**是 encryption, notification,
	// access, user, auth → 过滤成 access, user, auth —— 两者不同，正可判别是否走错顺序。
	want := []string{"user", "access", "auth"}
	gotNames := make([]string, 0, len(got))
	for _, d := range got {
		gotNames = append(gotNames, d.Name)
	}
	assert.Equal(t, want, gotNames)
	assert.NotContains(t, gotNames, "tenant", "minimal 不含 tenant")
	assert.Contains(t, gotNames, "auth")

	// 真子集：确实比 catalog 少（否则这条用例是恒真的）。
	assert.Less(t, len(gotNames), len(catalog.Names()))
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

// 结构种子需要 tenant 能力：不含它的形态必须被显式拒绝（附替代路径），
// 而不是让种子在事务中途抛 "no such table: tenants"（P2.6 迁移裁剪后 minimal/machine/enterprise 就是这种形态）。
func TestCheckSeedCapabilitiesRequiresTenant(t *testing.T) {
	full := []contract.Descriptor{{Name: "user"}, {Name: "access"}, {Name: "tenant"}}
	require.NoError(t, checkSeedCapabilities("full", full))

	enterprise := []contract.Descriptor{{Name: "user"}, {Name: "access"}, {Name: "console"}}
	err := checkSeedCapabilities("enterprise", enterprise)
	require.ErrorIs(t, err, errSeedNeedsTenant)
	require.ErrorContains(t, err, `profile "enterprise"`)
	require.ErrorContains(t, err, "full/saas")
}
