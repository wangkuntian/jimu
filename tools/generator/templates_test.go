package generator

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestTemplatesDrift 是 `make check-templates` 的实现：用生成器在 t.TempDir() 里生成一个最小项目
// （--profile=minimal、--module=example.com/proj-templates），然后在该目录里跑 `go build ./...`
// 与 `go run ./tools/checkcapabilities`；任一失败即测试失败。它把「模板/复制口径 vs 真实框架结构」
// 的漂移变成一次可复现的构建（recon §9 风险 3）。
//
// 两个口径要分清：
//
//	NoTidy:     离线可跑 —— 生成器复制的 go.mod/go.sum 已含全部依赖，模块缓存在 CI 上预热；
//	NoSelfCheck: 本测试**自己**跑那两条命令（用专用 GOCACHE，遵守 projectbuild_test.go 的
//	             Fix round 4 约束：不得让子进程写默认构建缓存）。NewProject 的默认自检路径由
//	             `jimu new` 的端到端验收覆盖，这里不重复一遍。
//
// 跳过条件是 requireHeavyMatrix（`-short` 或未设 JIMU_HEAVY_MATRIX）。
func TestTemplatesDrift(t *testing.T) {
	requireHeavyMatrix(t)
	dir := filepath.Join(t.TempDir(), "proj")
	_, err := newProjectForTest(t, NewOptions{
		Dir: dir, Profile: "minimal", Module: "example.com/proj-templates", NoTidy: true,
	})
	require.NoError(t, err)
	cache := newTestGoCache(t)
	runGoInProject(t, dir, cache, "build", "./...")
	out, err := runGoInProjectOutput(t, dir, cache, "run", "./tools/checkcapabilities")
	require.NoError(t, err, "生成项目的 checkcapabilities 必须绿:\n%s", out)
	// 第七条覆盖生成器核心与框架内部包的边界，少一条就是门禁被削弱的信号。
	assert.Equal(t, 7, strings.Count(out, "✅ check-capabilities:"), "门禁必须仍是 7 条:\n%s", out)
}
