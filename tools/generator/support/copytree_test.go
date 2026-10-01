package support

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCopyTreeSkipsFilteredSubtree(t *testing.T) {
	src := t.TempDir()
	write := func(rel string) {
		t.Helper()
		p := filepath.Join(src, filepath.FromSlash(rel))
		require.NoError(t, os.MkdirAll(filepath.Dir(p), 0o755))
		require.NoError(t, os.WriteFile(p, []byte(rel), 0o644))
	}
	write("queue/queue.go")
	write("queue/redis/redis.go")
	write("queue/kafka/kafka.go")

	dst := t.TempDir()
	// 过滤口径与驱动目录裁剪一致：rel 以 "queue/kafka" 开头的整棵子树不进。
	n, skipped, err := CopyTree(src, dst, func(rel string, d os.DirEntry) bool {
		return !strings.HasPrefix(rel, "queue/kafka")
	})
	require.NoError(t, err)
	assert.Equal(t, 2, n)
	// 被裁掉的子树在 skipped 里登记（T2 的 --report 要据此列出被裁的驱动目录/e2e 文件）。
	assert.Equal(t, []string{"queue/kafka"}, skipped)
	_, err = os.Stat(filepath.Join(dst, "queue", "kafka", "kafka.go"))
	assert.True(t, os.IsNotExist(err))
	_, err = os.Stat(filepath.Join(dst, "queue", "redis", "redis.go"))
	assert.NoError(t, err)
}

func TestCopyTreeFailsWhenSourceMissing(t *testing.T) {
	_, _, err := CopyTree(filepath.Join(t.TempDir(), "nope"), t.TempDir(), nil)
	require.Error(t, err)
}

func TestCopyTreeRejectsSymlinkSource(t *testing.T) {
	src := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(src, "a.go"), []byte("a"), 0o644))
	link := filepath.Join(t.TempDir(), "link")
	require.NoError(t, os.Symlink(src, link))

	// 指向目录的符号链接必须是错误：否则 WalkDir 的 Lstat 会静默复制 0 个文件。
	_, _, err := CopyTree(link, t.TempDir(), nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "symlink")
}

func TestCopyTreeReportsSkippedSymlink(t *testing.T) {
	src := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(src, "a.go"), []byte("a"), 0o644))
	require.NoError(t, os.Symlink(filepath.Join(src, "a.go"), filepath.Join(src, "b.go")))

	n, skipped, err := CopyTree(src, t.TempDir(), nil)
	require.NoError(t, err)
	assert.Equal(t, 1, n)
	assert.Equal(t, []string{"b.go"}, skipped)
}
