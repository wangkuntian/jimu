package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"jimu/tools/internal/profileassets"
)

// 正例：声明覆盖了资产根下的全部文件 → 无违规。
func TestCheckAssetsInAcceptsFullyOwnedTree(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, "deploy", "k8s"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(root, "deploy", "k8s", "x.yaml"), []byte("x"), 0o644))

	require.NoError(t, checkAssetsIn(root, map[string][]string{"group:ops": {"deploy/k8s"}}))
}

// 断言③：资产根下出现无有效所有者的文件 → 报错。
func TestCheckAssetsInRejectsUnownedFile(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, "deploy", "k8s"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(root, "deploy", "k8s", "x.yaml"), []byte("x"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(root, "deploy", "stray.yaml"), []byte("x"), 0o644))

	err := checkAssetsIn(root, map[string][]string{"group:ops": {"deploy/k8s"}})
	require.ErrorContains(t, err, "no effective owner")
}

// 断言①：声明的路径不存在 → 报错。
func TestCheckAssetsInRejectsDeclaredButMissing(t *testing.T) {
	root := t.TempDir()
	err := checkAssetsIn(root, map[string][]string{"cap:x": {"deploy/nope"}})
	require.ErrorContains(t, err, "declared by \"cap:x\" is missing")
}

// 断言②：同一路径被两个所有者声明 → 报错（能力↔能力、能力↔内核组同理）。
func TestCheckAssetsInRejectsDoubleDeclaration(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, "deploy", "k8s"), 0o755))

	err := checkAssetsIn(root, map[string][]string{
		"group:ops":           {"deploy/k8s"},
		"group:observability": {"deploy/k8s"},
	})
	require.ErrorContains(t, err, "declared by both")
}

// 断言：空路径是声明错误（不是静默忽略）。
func TestCheckAssetsInRejectsEmptyPath(t *testing.T) {
	err := checkAssetsIn(t.TempDir(), map[string][]string{"group:ops": {""}})
	require.ErrorContains(t, err, "empty asset path")
}

// 归一化变体不得绕过重复声明检查：deploy/k8s/ 与 deploy/k8s 是同一个前缀。
// （否则等长匹配会按所有者名排序静默误判归属 —— 审查发现的 Critical。）
func TestCheckAssetsInRejectsNormalizationVariantDuplicate(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, "deploy", "k8s"), 0o755))

	err := checkAssetsIn(root, map[string][]string{
		"group:ops":           {"deploy/k8s"},
		"group:observability": {"deploy/k8s/"},
	})
	require.ErrorContains(t, err, "declared by both")
}

// 声明必须落在资产根内：根外的路径永远不会被扫描，等于把未经校验的路径交给生成器。
func TestCheckAssetsInRejectsDeclarationOutsideRoots(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, "configs"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(root, "configs", "app.yaml"), []byte("x"), 0o644))

	err := checkAssetsIn(root, map[string][]string{"cap:x": {"configs/app.yaml"}})
	require.ErrorContains(t, err, "outside the asset roots")
}

// 断言④：每个形态都必须覆盖全部内核资产组（可注入夹具，故不是恒真）。
func TestCheckProfileCoreCoverage(t *testing.T) {
	names := []string{"a", "b"}
	core := map[string][]string{"ops": {"deploy/k8s"}}

	ok := func(string) ([]string, error) { return []string{"deploy/k8s"}, nil }
	require.NoError(t, checkProfileCoreCoverage(names, ok, core))

	missingB := func(name string) ([]string, error) {
		if name == "b" {
			return []string{"deploy/helm"}, nil
		}
		return []string{"deploy/k8s"}, nil
	}
	require.ErrorContains(t, checkProfileCoreCoverage(names, missingB, core), "ops/b:deploy/k8s")
}

// 真仓正例：仓库当前的资产声明必须自洽（这是门禁接入后的常态）。
func TestCheckAssetsOnRepo(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", ".."))
	require.NoError(t, err)
	require.FileExists(t, filepath.Join(root, "go.mod"))

	require.NoError(t, checkAssets(root))
	assert.NotEmpty(t, profileassets.Declared())
}

// 归一化变体（尾随空格、"//"、"./"、"." 段）不得绕过重复声明检查：
// 这些都归一化到同一个前缀，必须报「重复声明」而不是静默归错所有者。
func TestCheckAssetsInRejectsCanonicalVariantDuplicates(t *testing.T) {
	variants := []string{"deploy/k8s/", "deploy/k8s//", "deploy/./k8s", "deploy/k8s/.", "  deploy/k8s  "}
	for _, variant := range variants {
		t.Run(variant, func(t *testing.T) {
			root := t.TempDir()
			require.NoError(t, os.MkdirAll(filepath.Join(root, "deploy", "k8s"), 0o755))

			err := checkAssetsIn(root, map[string][]string{
				"group:ops":           {"deploy/k8s"},
				"group:observability": {variant},
			})
			require.ErrorContains(t, err, "declared by both", "变体 %q 必须与 deploy/k8s 冲突", variant)
		})
	}
}

// 归一化后仍落在资产根外（含 ".." 逃逸）→ 必须报错，不能因为字符串前缀而放行。
func TestCheckAssetsInRejectsDotDotEscape(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, "configs"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(root, "configs", "app.yaml"), []byte("x"), 0o644))

	for _, escape := range []string{"deploy/../configs", "deploy/../configs/app.yaml"} {
		err := checkAssetsIn(root, map[string][]string{"cap:x": {escape}})
		require.ErrorContainsf(t, err, "outside the asset roots", "逃逸写法 %q 必须被拒", escape)
	}
}
