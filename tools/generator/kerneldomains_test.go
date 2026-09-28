package generator

import (
	"go/parser"
	"go/token"
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestKernelRequiredDomainsCoverAppImports 是**防漂移测试**：internal/app（seed.go）在编译期
// 直接 import 的每一个能力 domain 叶子包，都必须被 kernelRequiredDomains 覆盖。将来有人往
// internal/app 里再加一个能力 domain，而 kernelRequiredDomains 没同步 → 这条测试立刻红，
// 否则「不含该能力的选择」会生成编译不过的项目（--with=queue 曾经就是这样坏的）。
func TestKernelRequiredDomainsCoverAppImports(t *testing.T) {
	root := FrameworkRoot()
	require.NotEmpty(t, root, "缺少框架源根")
	imported := appDomainImports(t, root)
	require.NotEmpty(t, imported, "internal/app 应当至少 import 一个能力 domain 叶子包（来源：seed.go）")
	for _, capDomain := range imported {
		assert.Contains(t, kernelRequiredDomains, capDomain,
			"internal/app import 了 %s，但 kernelRequiredDomains 未覆盖（生成项目会编译失败）", capDomain)
	}
}

// TestKernelRequiredDomainsAreSelfContained 钉住「只复制 domain/ 就够」：内核携带的 domain
// 叶子包不得依赖同一能力的其它包（否则部分复制会缺文件）。
func TestKernelRequiredDomainsAreSelfContained(t *testing.T) {
	root := FrameworkRoot()
	for _, capDomain := range kernelRequiredDomains {
		capName, _, _ := strings.Cut(capDomain, "/")
		for _, dep := range productionDeps(t, root, "./internal/capabilities/"+capDomain) {
			if dep == frameworkModule+"/"+capabilityDirPrefix+"/"+capDomain {
				continue // 包自身
			}
			rest, ok := strings.CutPrefix(dep, frameworkModule+"/"+capabilityDirPrefix+"/")
			if !ok {
				continue
			}
			owner, _, _ := strings.Cut(rest, "/")
			assert.NotEqual(t, capName, owner, "%s 依赖 %s：只复制 domain/ 不够", capDomain, dep)
		}
	}
}

// TestGeneratedProjectCarriesKernelRequiredDomainsAndBuilds 是裁定 ④ 的落地验收：
// 对**不含 user/access/tenant**（或只部分包含）的选择，三个内核 domain 必须存在，
// 且生成项目 `go build ./...` 绿。
func TestGeneratedProjectCarriesKernelRequiredDomainsAndBuilds(t *testing.T) {
	requireHeavyMatrix(t)
	cases := []struct {
		name string
		opts NewOptions
	}{
		{"with queue", NewOptions{With: "queue"}},
		{"with apikey", NewOptions{With: "apikey"}},
		{"profile machine", NewOptions{Profile: "machine"}},
	}
	cache := newTestGoCache(t) // 全组共用一个随测试删除的专用缓存
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := filepath.Join(t.TempDir(), "proj")
			tc.opts.Dir = dir
			tc.opts.Module = "example.com/proj"
			tc.opts.NoTidy = true
			_, err := newProjectForTest(t, tc.opts)
			require.NoError(t, err)

			for _, capDomain := range kernelRequiredDomains {
				assert.DirExists(t, filepath.Join(dir, "internal", "capabilities", filepath.FromSlash(capDomain)),
					"内核编译期 domain 依赖 %s 必须恒携带", capDomain)
			}
			assertProjectBuilds(t, dir, cache)
		})
	}
}

// appDomainImports 解析 internal/app 下全部非测试 .go 的 import，返回其中能力 domain 叶子包
// 的 `<cap>/domain` 列表（排序去重）。
func appDomainImports(t *testing.T, root string) []string {
	t.Helper()
	entries, err := os.ReadDir(filepath.Join(root, "internal", "app"))
	require.NoError(t, err)
	seen := map[string]bool{}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") || strings.HasSuffix(e.Name(), "_test.go") {
			continue
		}
		file, err := parser.ParseFile(token.NewFileSet(), filepath.Join(root, "internal", "app", e.Name()), nil, parser.ImportsOnly)
		require.NoError(t, err)
		for _, spec := range file.Imports {
			rest, ok := strings.CutPrefix(strings.Trim(spec.Path.Value, `"`), frameworkModule+"/"+capabilityDirPrefix+"/")
			if !ok {
				continue
			}
			capName, sub, hasSub := strings.Cut(rest, "/")
			if hasSub && (sub == "domain" || strings.HasPrefix(sub, "domain/")) {
				seen[capName+"/domain"] = true
			}
		}
	}
	return slices.Sorted(maps.Keys(seen))
}

// productionDeps 用 `go list -deps` 取一个包的**生产**依赖（相对框架仓根解析）。
func productionDeps(t *testing.T, root, pattern string) []string {
	t.Helper()
	cmd := exec.Command("go", "list", "-deps", pattern)
	cmd.Dir = root
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "go list -deps %s:\n%s", pattern, output)
	var deps []string
	for _, line := range strings.Split(strings.TrimSpace(string(output)), "\n") {
		if line != "" {
			deps = append(deps, line)
		}
	}
	return deps
}
