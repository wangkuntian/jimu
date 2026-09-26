package generator

import (
	"maps"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"jimu/tools/internal/profileassets"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAssetsForMinimalIncludesCoreGroupsButNotOpenapi 钉住 S6②/设计 §3.8：内核资产组
// （ops/observability）全形态携带，docs/openapi 只在含 apidocs 的选择里出现。
func TestAssetsForMinimalIncludesCoreGroupsButNotOpenapi(t *testing.T) {
	set, err := ParseCapabilitySet("minimal", "", "")
	require.NoError(t, err)
	got, err := AssetsFor(set)
	require.NoError(t, err)
	assert.Contains(t, got, "deploy/k8s")
	assert.Contains(t, got, "deploy/openobserve")
	assert.Contains(t, got, "deploy/backup")
	assert.NotContains(t, got, "docs/openapi")
}

func TestAssetsForFullIncludesOpenapi(t *testing.T) {
	set, err := ParseCapabilitySet("full", "", "")
	require.NoError(t, err)
	got, err := AssetsFor(set)
	require.NoError(t, err)
	assert.Contains(t, got, "docs/openapi")
}

// TestAssetsForWithUsesDeclaredTableNotARegistryProfile 钉住任务口径：`--with` 不查形态清单，
// 只取「选中能力在 profileassets.Declared() 里的资产」∪ 内核资产组。
func TestAssetsForWithUsesDeclaredTableNotARegistryProfile(t *testing.T) {
	set, err := ParseCapabilitySet("", "user,access,queue", "app")
	require.NoError(t, err)
	got, err := AssetsFor(set)
	require.NoError(t, err)
	assert.Contains(t, got, "deploy/k8s")
	assert.NotContains(t, got, "docs/openapi")
}

// TestAssetsForWithAPIdocsCarriesTheUngatedAsset 记录 T2 裁定修订第 3 条对 Task 6 brief 旧文本的
// 覆盖：apidocs 是 Ungated 能力，`--with=apidocs` **可选**（brief 旧的「报错 unknown capability」作废）。
func TestAssetsForWithAPIdocsCarriesTheUngatedAsset(t *testing.T) {
	set, err := ParseCapabilitySet("", "apidocs", "app")
	require.NoError(t, err)
	require.Contains(t, set.Declared, "apidocs")
	got, err := AssetsFor(set)
	require.NoError(t, err)
	assert.Contains(t, got, "docs/openapi")
}

// TestCopyAssetsCopiesEveryOwnedFile 钉住「复制 vs 归属」逐文件同源（P2.6 的 OwnershipIn 口径）：
// 复制了多少文件，就必须有多少个有效所有者 —— 少一个（有所有者却没复制）或多一个（复制了但无
// 所有者）都会让生成项目的第 5 条门禁失真。
func TestCopyAssetsCopiesEveryOwnedFile(t *testing.T) {
	root := frameworkRootForTest(t)
	dst := t.TempDir()
	files, err := CopyAssets(root, dst, []string{"deploy/openobserve"})
	require.NoError(t, err)
	assert.NotEmpty(t, files)
	owned, err := profileassets.OwnershipIn(dst, map[string][]string{"group:observability": {"deploy/openobserve"}})
	require.NoError(t, err)
	assert.Equal(t, slices.Sorted(maps.Keys(owned)), files)
}

// TestCopyAssetsCopiesExactlyTheOwnedTree 用 minimal 的真实资产集做整树核对：复制结果 == 该树上
// 每个文件的有效所有者集合（OwnershipIn 对无主文件直接报错，故这条同时是「无未声明资产」的证明）。
func TestCopyAssetsCopiesExactlyTheOwnedTree(t *testing.T) {
	root := frameworkRootForTest(t)
	set, err := ParseCapabilitySet("minimal", "", "")
	require.NoError(t, err)
	assets, err := AssetsFor(set)
	require.NoError(t, err)

	dst := t.TempDir()
	files, err := CopyAssets(root, dst, assets)
	require.NoError(t, err)
	require.NotEmpty(t, files)

	owned, err := profileassets.OwnershipIn(dst, profileassets.Declared())
	require.NoError(t, err)
	assert.Equal(t, slices.Sorted(maps.Keys(owned)), files)
	// 内核单文件资产（deploy/otel-collector.yaml）必须真的落地，不是只复制目录。
	assert.Contains(t, files, "deploy/otel-collector.yaml")
}

// TestAssetsForMatchesTheAssetTableTheGeneratedProjectDerives 是任务铁律 2 的漂移护栏：
// 生成项目的 profileassets.Declared() 从**生成版 catalog ∪ registry** 派生，而生成版
// catalog 的 entries = Declared ∪ MigrationOnly（RenderCatalog）、registry 只含 Declared，
// 故生成项目算出的有效声明表 == 这些能力的 Descriptor.Assets ∪ 内核资产组。
// AssetsFor 必须与这张表**逐路径相等**：不等就意味着生成项目门禁要么误红（声明了没复制）
// 要么静默放过（复制了没声明）。
func TestAssetsForMatchesTheAssetTableTheGeneratedProjectDerives(t *testing.T) {
	declared := profileassets.Declared()
	for _, tc := range []struct{ name, profile, with, shape string }{
		{"profile minimal", "minimal", "", ""},
		{"profile full", "full", "", ""},
		{"with user,access,queue", "", "user,access,queue", "app"},
		{"with apidocs", "", "apidocs", "app"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			set, err := ParseCapabilitySet(tc.profile, tc.with, tc.shape)
			require.NoError(t, err)
			got, err := AssetsFor(set)
			require.NoError(t, err)

			want := map[string]bool{}
			for _, paths := range profileassets.CoreGroups() {
				for _, p := range paths {
					want[profileassets.Canonical(p)] = true
				}
			}
			for _, name := range append(slices.Clone(set.Declared), set.MigrationOnly...) {
				for _, p := range declared["cap:"+name] {
					want[profileassets.Canonical(p)] = true
				}
			}
			assert.Equal(t, slices.Sorted(maps.Keys(want)), got)
		})
	}
}

// TestAssetsForRejectsUnknownCapability fail-closed：声明集里出现 known 集合之外的能力名时
// 直接报错，绝不用「查不到资产」静默降级成空集。
func TestAssetsForRejectsUnknownCapability(t *testing.T) {
	set := CapabilitySet{Declared: []string{"nope"}, Known: []string{"user"}}
	_, err := AssetsFor(set)
	require.ErrorContains(t, err, "nope")
}

// TestValuesSectionsPrunesUnselectedCapabilityKeys 是 T4 裁定 ⑫ 的接线验收：values.yaml 的
// 键裁剪必须复用 T4 的 ValuesSections/RenderValuesYAML（不新增第二套实现），且实测
// minimal 22 键（24 - audit - storage）、full 24 键。
func TestValuesSectionsPrunesUnselectedCapabilityKeys(t *testing.T) {
	src := readRepoFile(t, "deploy/helm/values.yaml")

	minimal, err := ParseCapabilitySet("minimal", "", "")
	require.NoError(t, err)
	keep, err := ValuesSections(src, minimal)
	require.NoError(t, err)
	assert.Len(t, keep, 22)
	assert.Contains(t, keep, "auth") // 选中 → 与能力同名的键保留
	assert.NotContains(t, keep, "audit")
	assert.NotContains(t, keep, "storage")

	out, err := RenderValuesYAML(src, keep)
	require.NoError(t, err)
	assert.Contains(t, string(out), "\nopenobserve:") // 内核键恒保留
	assert.Contains(t, string(out), "\nauth:")
	assert.NotContains(t, string(out), "\naudit:")
	assert.NotContains(t, string(out), "\nstorage:")

	full, err := ParseCapabilitySet("full", "", "")
	require.NoError(t, err)
	fullKeys, err := ValuesSections(src, full)
	require.NoError(t, err)
	assert.Len(t, fullKeys, 24)
	assert.Equal(t, keep, slices.DeleteFunc(slices.Clone(fullKeys), func(k string) bool { return k == "audit" || k == "storage" }))
}

// TestCopiedAssetsKeepFrameworkNames 钉住资产复制的执行落点：必须在 RewriteModule **之后**，
// 否则字面量重写规则 `jimu/` → `<module>/` 会把 deploy/backup/Dockerfile 的 /opt/jimu/scripts/
// 改成 /opt/<module>/scripts/（容器内不存在的路径）。资产里的 jimu 是框架自己的名字（裁定 3）。
func TestCopiedAssetsKeepFrameworkNames(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "proj")
	_, err := NewProject(NewOptions{Dir: dir, Profile: "minimal", Module: "example.com/proj", NoTidy: true})
	require.NoError(t, err)
	content, err := os.ReadFile(filepath.Join(dir, "deploy", "backup", "Dockerfile"))
	require.NoError(t, err)
	assert.Contains(t, string(content), "/opt/jimu/scripts/")
	assert.NotContains(t, string(content), "example.com/proj")
}

// TestCheckCapabilitiesAssetsCheckIsNotPatchedInGeneratedProject 是 T5 裁定 ⑭ 的护栏：
// 生成项目的资产段必须是蓝本原文（T5 的「资产根整棵缺席」容错是临时桥，T6 落地资产后必须拆除，
// 否则「资产根缺席」会成为永久静默豁免，掩盖未来的复制/声明漂移）。
func TestCheckCapabilitiesAssetsCheckIsNotPatchedInGeneratedProject(t *testing.T) {
	assert.NotContains(t, filePatches, "tools/checkcapabilities/assets.go",
		"资产段不得再有任何定点补丁：第 5 条门禁必须真正校验「声明的资产路径必须存在」")
}
