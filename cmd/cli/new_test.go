package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"jimu/internal/profiles"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCLIVersionIsSharedWithGenerator(t *testing.T) {
	oldVersion, oldProfileVersion := version, profiles.Version
	t.Cleanup(func() {
		version, profiles.Version = oldVersion, oldProfileVersion
	})
	version = "v0.5.0-test"
	applyVersion()
	assert.Equal(t, version, profiles.Version)
}

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
	c.SetArgs([]string{"new", dir, "--template=basic-api", "--module=example.com/proj", "--dry-run"})
	require.NoError(t, c.Execute())

	_, statErr := os.Stat(dir)
	assert.True(t, os.IsNotExist(statErr), "--dry-run 绝不落盘")
	assert.Contains(t, out.String(), "example.com/proj")
	assert.Contains(t, out.String(), "basic-api")
}

// TestNewCmdDryRunDistinguishesAssemblyFromCopySet ④：`--dry-run` 的两个能力集口径不同且必须各自
// 标注 —— 装配集（写进 marker/assembly 的声明集）与复制集（额外含编译闭包/迁移携带/内核编译期
// domain 依赖的目录）。曾经只打一行 `capabilities` 且打的是复制集，读者会把它当装配集。
// `minimal` 装配集不含 schema 依赖 tenant，复制集会额外携带它。
func TestNewCmdDryRunDistinguishesAssemblyFromCopySet(t *testing.T) {
	var out bytes.Buffer
	c := newCommandForTest(&out)
	c.SetArgs([]string{"new", filepath.Join(t.TempDir(), "proj"), "--profile=minimal", "--module=example.com/proj", "--dry-run"})
	require.NoError(t, c.Execute())

	plan := out.String()
	assembly := planLine(t, plan, "capabilities")
	copySet := planLine(t, plan, "copy set")
	assert.Contains(t, assembly, "(装配集)")
	assert.Contains(t, copySet, "(复制集")
	// tenant 只作迁移携带，不进装配集。
	assert.NotContains(t, assembly, "outbox")
	assert.NotContains(t, assembly, "queue")
	assert.NotContains(t, copySet, "outbox")
	assert.NotContains(t, copySet, "queue")
	assert.Contains(t, copySet, "tenant")
	assert.NotEqual(t, assembly, copySet, "两个能力集不相等，必须分两行打")
}

// planLine 取 `--dry-run` 输出里以 "  <label> " 开头的那一行（label 后必须有空格，避免
// `capabilities` 误匹配到别的前缀）。
func planLine(t *testing.T, plan, label string) string {
	t.Helper()
	for _, line := range strings.Split(plan, "\n") {
		if strings.HasPrefix(line, "  "+label+" ") {
			return line
		}
	}
	t.Fatalf("dry-run 输出里没有 %q 行：\n%s", label, plan)
	return ""
}

func TestNewCmdDefaultsToBasicAPI(t *testing.T) {
	var out bytes.Buffer
	c := newCommandForTest(&out)
	c.SetArgs([]string{"new", filepath.Join(t.TempDir(), "proj"), "--dry-run"})
	require.NoError(t, c.Execute())
	assert.Contains(t, out.String(), "template     basic-api")
	assert.Contains(t, out.String(), "support      official")
}

func TestNewCmdMapsLegacyMinimalToAuthAPI(t *testing.T) {
	var out bytes.Buffer
	c := newCommandForTest(&out)
	c.SetArgs([]string{"new", filepath.Join(t.TempDir(), "proj"), "--profile=minimal", "--dry-run"})
	require.NoError(t, c.Execute())
	assert.Contains(t, out.String(), "warning: --profile=minimal")
	assert.Contains(t, out.String(), "template     auth-api")
	assert.Contains(t, out.String(), "support      official")
}

func TestNewCmdAcceptsAuthAPITemplate(t *testing.T) {
	var out bytes.Buffer
	c := newCommandForTest(&out)
	c.SetArgs([]string{"new", filepath.Join(t.TempDir(), "proj"), "--template=auth-api", "--dry-run"})
	require.NoError(t, c.Execute())
	assert.Contains(t, out.String(), "template     auth-api")
	assert.Contains(t, out.String(), "support      official")
}

func TestNewCmdRejectsUnknownTemplate(t *testing.T) {
	c := newCommandForTest(&bytes.Buffer{})
	c.SetArgs([]string{"new", filepath.Join(t.TempDir(), "proj"), "--template=ghost", "--dry-run"})
	require.ErrorContains(t, c.Execute(), "unknown template")
}

func TestNewCmdRejectsInternalProfile(t *testing.T) {
	c := newCommandForTest(&bytes.Buffer{})
	c.SetArgs([]string{"new", filepath.Join(t.TempDir(), "proj"), "--profile=full", "--dry-run"})
	err := c.Execute()
	require.ErrorContains(t, err, "framework-internal")
	require.ErrorContains(t, err, "--template")
}

func TestNewCmdMarksWithAsExperimental(t *testing.T) {
	var out bytes.Buffer
	c := newCommandForTest(&out)
	c.SetArgs([]string{"new", filepath.Join(t.TempDir(), "proj"), "--with=apidocs", "--dry-run"})
	require.NoError(t, c.Execute())
	assert.Contains(t, out.String(), "support      experimental")
}

func TestNewCmdRejectsTemplateSelectorsTogether(t *testing.T) {
	for _, args := range [][]string{
		{"--template=basic-api", "--profile=minimal"},
		{"--template=basic-api", "--with=apidocs"},
	} {
		c := newCommandForTest(&bytes.Buffer{})
		c.SetArgs(append([]string{"new", filepath.Join(t.TempDir(), "proj"), "--dry-run"}, args...))
		require.ErrorContains(t, c.Execute(), "mutually exclusive")
	}
}

func TestNewCmdRejectsShapeWithoutWith(t *testing.T) {
	c := newCommandForTest(&bytes.Buffer{})
	c.SetArgs([]string{"new", filepath.Join(t.TempDir(), "proj"), "--template=basic-api", "--shape=app", "--dry-run"})
	require.ErrorContains(t, c.Execute(), "--shape is only valid with --with")
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
