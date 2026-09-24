package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewCmdRejectsBothProfileAndWith 参数互斥校验只在 ParseCapabilitySet 一处：
// 命令层不重复实现，但必须把同一错误透出来（--dry-run 也走同一校验）。
func TestNewCmdRejectsBothProfileAndWith(t *testing.T) {
	c := newCommandForTest(&bytes.Buffer{})
	c.SetArgs([]string{"new", filepath.Join(t.TempDir(), "proj"), "--profile=minimal", "--with=user", "--dry-run"})
	err := c.Execute()
	require.ErrorContains(t, err, "mutually exclusive")
}

// TestNewCmdWiresEveryFlag 钉住 CLI 到 generator.NewOptions 的字段映射：少接一个 flag
// 就是「参数静默失效」，--dry-run 路径能在不落盘的前提下暴露它。
func TestNewCmdWiresEveryFlag(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "proj")
	var out bytes.Buffer
	c := newCommandForTest(&out)
	c.SetArgs([]string{"new", dir, "--profile=minimal", "--module=example.com/proj", "--dry-run"})
	require.NoError(t, c.Execute())

	_, statErr := os.Stat(dir)
	assert.True(t, os.IsNotExist(statErr), "--dry-run 绝不落盘")
	assert.Contains(t, out.String(), "example.com/proj")
	assert.Contains(t, out.String(), "minimal")
}

// TestNewCmdRejectsMissingCapabilitySelection 两个选择都不给 → 报错（要求二选一）。
func TestNewCmdRejectsMissingCapabilitySelection(t *testing.T) {
	c := newCommandForTest(&bytes.Buffer{})
	c.SetArgs([]string{"new", filepath.Join(t.TempDir(), "proj"), "--dry-run"})
	err := c.Execute()
	require.ErrorContains(t, err, "--profile")
	require.ErrorContains(t, err, "--with")
}

// newCommandForTest 复制一份只含 new 子命令的根命令，避免测试间共享 cobra flag 状态。
func newCommandForTest(out *bytes.Buffer) *cobra.Command {
	root := &cobra.Command{Use: "jimu", SilenceUsage: true, SilenceErrors: true}
	root.SetOut(out)
	root.SetErr(out)
	// 用构造函数拿一份全新的命令：共享 newCmd 会让 pflag 报 "flag redefined"
	// 且一个用例设过的 --with 会泄漏到下一个用例。
	cmd := newScaffoldCmd()
	cmd.SetOut(out)
	cmd.SetErr(out)
	root.AddCommand(cmd)
	return root
}
