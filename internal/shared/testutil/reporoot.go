package testutil

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// RepoRoot 返回当前模块的根目录：从 cwd（`go test` 下 cwd 即被测包目录）向上找第一个 go.mod。
//
// 这是测试里定位**模块根**（`configs/`、`conf/`、`internal/capabilities/*/migrations` …）的唯一入口。
// 不要用 `runtime.Caller(0)` 再向上拼层数：生成项目的重型矩阵构建一律带 `-trimpath`
// （`tools/generator` 的 trimpathGoflags，为了让测试用的 GOCACHE 跨运行复用），编译期路径会被重写成
// **模块相对路径**（如 `example.com/proj/internal/app/seed_test.go`），据此推出的「根」是字符串而不是
// 磁盘目录 —— 实测报 `chdir example.com/proj: no such file or directory`（CI 的 Scaffold Gate，
// 由 `internal/app/seed_test.go` 的 TestRunSeedWithCasbin 暴露）。
//
// 判据用 go.mod 而不是固定层数：框架仓与生成项目各自有 go.mod，同一份测试代码在两边都能定位到**自己**
// 的根，且与被测包在树里的深度无关。
//
// 包内 testdata 用 TestdataDir；其余路径需要跨包一致时也应经本 helper 取根后拼接。
func RepoRoot(t *testing.T) string {
	t.Helper()
	root, err := repoRoot()
	require.NoError(t, err)
	return root
}

// MustRepoRoot 与 RepoRoot 同一实现，供**拿不到 *testing.T** 的场合（包级 init() 注入迁移目录等）：
// 定位失败直接 panic —— 测试二进制连模块根都找不到时，继续跑没有任何意义。
func MustRepoRoot() string {
	root, err := repoRoot()
	if err != nil {
		panic(err)
	}
	return root
}

// repoRoot 是 RepoRoot / MustRepoRoot 的共同实现。
func repoRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("resolve working directory: %w", err)
	}
	for {
		if _, statErr := os.Stat(filepath.Join(dir, "go.mod")); statErr == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("从 %s 向上找不到模块根（go.mod）", dir)
		}
		dir = parent
	}
}

// TestdataDir 返回**被测包内** testdata 下的目录（相对 cwd，而 `go test` 的 cwd 就是包目录）。
// 与 RepoRoot 同一理由不用 `runtime.Caller`：相对路径同样不受 `-trimpath` 影响。
func TestdataDir(rel string) string {
	return filepath.Join("testdata", rel)
}
