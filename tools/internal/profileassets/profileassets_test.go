package profileassets

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 最长前缀：deploy/k8s 整体归 ops，但两个观测文件被更具体的前缀抢走。
func TestOwnerUsesLongestPrefix(t *testing.T) {
	tests := []struct {
		path  string
		owner string
	}{
		{"deploy/k8s/deployment.yaml", "group:ops"},
		{"deploy/k8s/backup-cronjob.yaml", "group:ops"},
		{"deploy/k8s/otel-collector.yaml", "group:observability"},
		{"deploy/k8s/openobserve.yaml", "group:observability"},
		{"deploy/helm/templates/deployment.yaml", "group:ops"},
		{"deploy/helm/templates/otel-collector.yaml", "group:observability"},
		{"deploy/openobserve/alerts/jimu_http_5xx_rate.json", "group:observability"},
		{"deploy/backup/Dockerfile", "group:ops"},
		{"docs/openapi/swagger.json", "cap:apidocs"},
	}
	for _, tt := range tests {
		owner, ok := Owner(tt.path)
		require.Truef(t, ok, "Owner(%q) 应命中", tt.path)
		assert.Equalf(t, tt.owner, owner, "Owner(%q)", tt.path)
	}

	// 不在任何资产根/声明下：configs 不纳入资产根。
	_, ok := Owner("configs/app.yaml")
	assert.False(t, ok)
	_, ok = Owner("deploy")
	assert.False(t, ok, "目录本身不是文件级归属")
	_, ok = Owner("")
	assert.False(t, ok)
}

func TestOwnerInUsesProvidedTable(t *testing.T) {
	table := map[string][]string{"group:ops": {"deploy/k8s"}}
	owner, ok := OwnerIn(table, "deploy/k8s/x.yaml")
	require.True(t, ok)
	assert.Equal(t, "group:ops", owner)
	_, ok = OwnerIn(table, "deploy/helm/x.yaml")
	assert.False(t, ok)
}

// 等长并列（两个所有者声明同一个归一化前缀）→ 视为无有效所有者（fail-closed，
// 不按所有者名排序抽一个）。门禁的重复声明检查会先报更明确的错误。
func TestOwnerInRejectsEqualLengthTie(t *testing.T) {
	table := map[string][]string{
		"group:ops":           {"deploy/k8s"},
		"group:observability": {"deploy/k8s/"},
	}
	_, ok := OwnerIn(table, "deploy/k8s/x.yaml")
	assert.False(t, ok)

	// 真正的最长前缀仍然获胜（不同长度不构成并列）。
	table["group:observability"] = []string{"deploy/k8s/openobserve.yaml"}
	owner, ok := OwnerIn(table, "deploy/k8s/openobserve.yaml")
	require.True(t, ok)
	assert.Equal(t, "group:observability", owner)
	owner, ok = OwnerIn(table, "deploy/k8s/deployment.yaml")
	require.True(t, ok)
	assert.Equal(t, "group:ops", owner)
}

// 内核资产组全形态携带；能力资产随形态（apidocs 只在 full 的清单里，
// 因此只有 full 含 docs/openapi —— 这正是 APIdocs 条件化的前提）。
func TestForProfileIncludesCapabilityAssetsAndCoreGroups(t *testing.T) {
	for _, profile := range []string{"full", "minimal", "saas", "enterprise", "machine"} {
		got, err := ForProfile(profile)
		require.NoErrorf(t, err, "ForProfile(%q)", profile)
		assert.Containsf(t, got, "deploy/openobserve", "%s 应含内核观测资产组", profile)
		assert.Containsf(t, got, "deploy/k8s", "%s 应含内核运维资产组", profile)
		assert.Containsf(t, got, "deploy/helm/templates/otel-collector.yaml", "%s 应含观测清单", profile)
	}

	full, err := ForProfile("full")
	require.NoError(t, err)
	assert.Contains(t, full, "docs/openapi")

	for _, profile := range []string{"minimal", "saas", "enterprise", "machine"} {
		got, err := ForProfile(profile)
		require.NoErrorf(t, err, "ForProfile(%q)", profile)
		assert.NotContainsf(t, got, "docs/openapi", "%s 的清单不含 apidocs，故不应带 APIdocs 资产", profile)
	}
}

func TestForProfileRejectsUnknownProfile(t *testing.T) {
	_, err := ForProfile("ghost")
	require.ErrorContains(t, err, `unknown profile "ghost"`)
}

// 真实仓库的资产分割必须与设计一致：48 文件 = cap:apidocs 3 / group:observability 22 / group:ops 23。
func TestOwnershipPartitionsTheRepo(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", "..", ".."))
	require.NoError(t, err)
	require.FileExists(t, filepath.Join(root, "go.mod"), "应当在仓库内运行测试")

	got, err := Ownership(root)
	require.NoError(t, err)

	counts := map[string]int{}
	for _, owner := range got {
		counts[owner]++
	}
	assert.Equal(t, 3, counts["cap:apidocs"])
	assert.Equal(t, 22, counts["group:observability"])
	assert.Equal(t, 23, counts["group:ops"])
	assert.Len(t, got, 48)
	assert.NotContains(t, got, "configs/app.yaml", "configs 不纳入资产根")
}

// 资产根不存在时（生成项目的裁剪结果）不报错、返回空集。
func TestOwnershipToleratesMissingRoots(t *testing.T) {
	got, err := OwnershipIn(t.TempDir(), map[string][]string{"group:ops": {"deploy"}})
	require.NoError(t, err)
	assert.Empty(t, got)
}

// 无有效所有者的文件必须报错（门禁断言②的底层行为）。
func TestOwnershipInRejectsUnownedFile(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, "deploy", "k8s"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(root, "deploy", "stray.yaml"), []byte("x"), 0o644))

	_, err := OwnershipIn(root, map[string][]string{"group:ops": {"deploy/k8s"}})
	require.ErrorContains(t, err, `asset "deploy/stray.yaml" has no effective owner`)
}

// 以 "." 开头的文件与目录被跳过（覆盖 .DS_Store 这类 macOS 残留）。
func TestOwnershipSkipsDotEntries(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, "deploy", ".git"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(root, "deploy", ".DS_Store"), []byte("x"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(root, "deploy", "kept.yaml"), []byte("x"), 0o644))

	got, err := OwnershipIn(root, map[string][]string{"group:ops": {"deploy"}})
	require.NoError(t, err)
	assert.Equal(t, map[string]string{"deploy/kept.yaml": "group:ops"}, got)
}

// 派生输出必须与归属判定同口径：每个形态的资产路径都已是 canonical（幂等）。
func TestForProfileEmitsCanonicalPaths(t *testing.T) {
	for _, profile := range []string{"full", "minimal", "saas", "enterprise", "machine"} {
		paths, err := ForProfile(profile)
		require.NoErrorf(t, err, "ForProfile(%q)", profile)
		for _, p := range paths {
			assert.Equalf(t, p, Canonical(p), "%s 的资产路径 %q 不是 canonical", profile, p)
		}
	}
}

func TestCoreGroupsReturnsCopy(t *testing.T) {
	got := CoreGroups()
	got["ops"][0] = "tampered"
	assert.NotEqual(t, "tampered", CoreGroups()["ops"][0])
}

// Canonical 是唯一归一化口径：去空白、解析 "."/".."、折叠重复斜杠、去结尾 "/"。
func TestCanonical(t *testing.T) {
	tests := map[string]string{
		"deploy/k8s":                 "deploy/k8s",
		"deploy/k8s/":                "deploy/k8s",
		"deploy/k8s//":               "deploy/k8s",
		"deploy//k8s":                "deploy/k8s",
		"deploy/./k8s":               "deploy/k8s",
		"  deploy/k8s  ":             "deploy/k8s",
		"deploy/../configs":          "configs",
		"deploy/../configs/app.yaml": "configs/app.yaml",
		"./docs/openapi":             "docs/openapi",
		".":                          "",
		"/":                          "",
		"":                           "",
	}
	for in, want := range tests {
		assert.Equalf(t, want, Canonical(in), "Canonical(%q)", in)
	}
}

// 查询路径与声明前缀都走 Canonical：任一写法都能正确匹配（且不静默归错）。
func TestOwnerInNormalizesQueryAndDeclaration(t *testing.T) {
	table := map[string][]string{"group:ops": {"deploy/k8s//"}, "group:observability": {"deploy/k8s/./otel-collector.yaml "}}
	owner, ok := OwnerIn(table, "deploy/k8s/deployment.yaml")
	require.True(t, ok)
	assert.Equal(t, "group:ops", owner)

	owner, ok = OwnerIn(table, "deploy//k8s/otel-collector.yaml")
	require.True(t, ok)
	assert.Equal(t, "group:observability", owner, "更长前缀仍获胜，且大小写/斜杠写法不影响")
}
