package main

import (
	"os"
	"path/filepath"
	"testing"

	"jimu/tools/internal/profileoverlay"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestWriteMatchesTheSharedPackage 命令行写出的产物必须与共享包 tools/internal/profileoverlay
// 逐字节相同：模板与路径口径只有一份；本命令若再长出副本，这条断言失败。
func TestWriteMatchesTheSharedPackage(t *testing.T) {
	root := t.TempDir()
	jsonPath, err := write(root, "minimal")
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(profileoverlay.Dir(root, "minimal"), "overlay.json"), jsonPath)

	want, err := profileoverlay.Source("minimal")
	require.NoError(t, err)
	got, err := os.ReadFile(filepath.Join(profileoverlay.Dir(root, "minimal"), "active.go"))
	require.NoError(t, err)
	assert.Equal(t, want, string(got))
}

// TestWriteRejectsUnknownProfile 形态名非法时返回错误（make/Docker 依赖它非零退出，而不是
// 留下一个空的 -overlay=）。
func TestWriteRejectsUnknownProfile(t *testing.T) {
	_, err := write(t.TempDir(), "ghost")
	require.ErrorContains(t, err, `unknown profile "ghost"`)
}
