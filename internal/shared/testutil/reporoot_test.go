package testutil

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRepoRootIsAnAbsoluteDiskPath 钉住 RepoRoot 的关键性质：返回**磁盘上的绝对路径**。
//
// 这正是它取代的 `runtime.Caller(0)` 写法在 `-trimpath` 下丢掉的性质：编译期路径被重写成模块相对
// 路径后（如 `example.com/proj/internal/app/seed_test.go`），据此推出的「根」是字符串
// `example.com/proj`，拿去 os.Stat / t.Chdir 全部失败 —— CI 的 Scaffold Gate 实测过
// （`chdir example.com/proj: no such file or directory`）。本用例不依赖 -trimpath 也能判定：
// 任何「模块相对字符串」都不是绝对路径。
func TestRepoRootIsAnAbsoluteDiskPath(t *testing.T) {
	root := RepoRoot(t)
	require.True(t, filepath.IsAbs(root), "RepoRoot 必须是绝对路径，实际 %q", root)
	require.FileExists(t, filepath.Join(root, "go.mod"), "模块根下必须有 go.mod")
	assert.Equal(t, MustRepoRoot(), root, "MustRepoRoot 与 RepoRoot 共用同一实现")

	// 包内 testdata 走相对路径（cwd 即包目录），与上面同一理由不受 -trimpath 影响。
	assert.Equal(t, filepath.Join("testdata", "capmigs"), TestdataDir("capmigs"))
	assert.False(t, filepath.IsAbs(TestdataDir("capmigs")), "TestdataDir 应是相对 cwd 的包内路径")
}
