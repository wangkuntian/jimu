package generator

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 本文件是 ⑧ Tidy 与 ⑨ SelfCheck 两条原语的单测。真实生成项目上的完整路径（默认 tidy + 自检
// 全绿）由 `jimu new` 的端到端验收与 `make check-templates` 覆盖 —— 本仓测试不重复跑完整构建
// （GOCACHE 约束见 projectbuild_test.go），只钉失败语义与最小正例。

// TestTidyRewritesTheModule 最小正例：无依赖模块的 tidy 必须成功（不联网、不拉任何模块）。
func TestTidyRewritesTheModule(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"),
		[]byte("module example.com/tidy\n\ngo 1.24\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "main.go"),
		[]byte("package main\n\nfunc main() {}\n"), 0o644))

	require.NoError(t, Tidy(dir))
	content, err := os.ReadFile(filepath.Join(dir, "go.mod"))
	require.NoError(t, err)
	assert.Contains(t, string(content), "module example.com/tidy")
}

// TestTidyFailsOnBrokenGoMod 违反 go.mod 时 tidy 必须报错（不吞、不降级为警告）——
// `jimu new` 据这个错误回滚，`--no-tidy` 是唯一的绕过方式。
func TestTidyFailsOnBrokenGoMod(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte("this is not a go.mod\n"), 0o644))
	err := Tidy(dir)
	require.Error(t, err)
	assert.ErrorContains(t, err, "go mod tidy")
}

// TestSelfCheckReportsBuildFailure ⑨ 的第一道是 `go build ./...`：编译不过即失败，且错误里带上
// 编译器输出（否则最难排查）。第二条（go run ./tools/checkcapabilities）在 build 失败时不会执行。
func TestSelfCheckReportsBuildFailure(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"),
		[]byte("module example.com/broken\n\ngo 1.24\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "main.go"),
		[]byte("package main\n\nfunc main() { undefinedSymbol() }\n"), 0o644))

	err := SelfCheck(dir)
	require.Error(t, err)
	assert.ErrorContains(t, err, "go build ./...")
	assert.Contains(t, err.Error(), "undefinedSymbol", "构建失败必须带上编译器输出")
}
