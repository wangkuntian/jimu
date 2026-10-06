package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"jimu/tools/generator"
	"jimu/tools/generator/frameworkmanifest"
	"jimu/tools/generator/manifest"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeGeneratedProject 造一个最小可用的 manifest 项目。add 会把产物渲染进暂存目录再与这个目录比对差异，因此 CLI
// 层的 flag 接线可以在不跑完整 `jimu new` 的前提下被覆盖。
func fakeGeneratedProject(t *testing.T, capabilities []string) string {
	t.Helper()
	dir := t.TempDir()
	root := generator.FrameworkRoot()
	require.NotEmpty(t, root, "测试必须能定位框架源根")
	doc, err := frameworkmanifest.Export(frameworkmanifest.Request{
		Root: root, With: strings.Join(capabilities, ","), Shape: "app", Module: "example.com/proj",
	})
	require.NoError(t, err)
	require.NoError(t, manifest.Write(filepath.Join(dir, ".jimu", "manifest.json"), doc))
	return dir
}

// capabilityCommandForTest 复制一份只含 capability 子命令的根命令，避免测试间共享 cobra flag 状态。
func capabilityCommandForTest(out *bytes.Buffer) *cobra.Command {
	root := &cobra.Command{Use: "jimu", SilenceUsage: true, SilenceErrors: true}
	root.SetOut(out)
	root.SetErr(out)
	cmd := newCapabilityCmd()
	cmd.SetOut(out)
	cmd.SetErr(out)
	root.AddCommand(cmd)
	return root
}

func TestCapabilityCreateCmdCreatesSkeleton(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(root, "go.mod"), []byte("module jimu\n"), 0o644))
	require.NoError(t, os.MkdirAll(filepath.Join(root, "internal", "capabilities"), 0o755))
	t.Chdir(root)

	cmd := capabilityCommandForTest(&bytes.Buffer{})
	cmd.SetArgs([]string{"capability", "create", "product"})
	require.NoError(t, cmd.Execute())
	assert.FileExists(t, filepath.Join(root, "internal", "capabilities", "product", "module.go"))
}

func TestModuleCreateCmdIsRemoved(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(root, "go.mod"), []byte("module jimu\n"), 0o644))
	require.NoError(t, os.MkdirAll(filepath.Join(root, "internal", "capabilities"), 0o755))
	t.Chdir(root)

	rootCmd.SetArgs([]string{"module", "create", "product"})
	t.Cleanup(func() { rootCmd.SetArgs(nil) })
	require.ErrorContains(t, rootCmd.Execute(), "unknown command")
	assert.NoDirExists(t, filepath.Join(root, "internal", "capabilities", "product"))
}

// TestCapabilityAddCmdDryRunWiresFlags 钉住 CLI → generator.AddOptions 的 flag 映射与
// 「--dry-run 绝不落盘」：输出必须列出将改动的文件，且目录里除了 manifest 目录什么都不多。
func TestCapabilityAddCmdDryRunWiresFlags(t *testing.T) {
	dir := fakeGeneratedProject(t, []string{"user", "access", "auth", "encryption", "notification"})
	var out bytes.Buffer
	c := capabilityCommandForTest(&out)
	c.SetArgs([]string{"capability", "add", "dataops", "--dir=" + dir, "--dry-run"})
	require.NoError(t, c.Execute())

	assert.Contains(t, out.String(), "dataops")
	assert.Contains(t, out.String(), "internal/capabilities/catalog/catalog.go")
	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	require.Len(t, entries, 1, "--dry-run 只允许留下 manifest 目录")
	assert.Equal(t, ".jimu", entries[0].Name())
}

// TestCapabilityAddCmdUsesCurrentFramework --from 缺省取当前框架 checkout。
func TestCapabilityAddCmdUsesMarkerSourceRoot(t *testing.T) {
	dir := fakeGeneratedProject(t, []string{"user", "access", "auth"})
	var out bytes.Buffer
	c := capabilityCommandForTest(&out)
	c.SetArgs([]string{"capability", "add", "dataops", "--dir=" + dir, "--dry-run"})
	require.NoError(t, c.Execute(), "--from 缺省必须回落到 marker.SourceRoot")
	assert.Contains(t, out.String(), "dataops")
}

// TestCapabilityAddCmdRejectsExistingUnlessForce 钉住 flag 语义：已在 entries 里 → 报错；
// --force 才允许重建。
func TestCapabilityAddCmdRejectsExistingUnlessForce(t *testing.T) {
	dir := fakeGeneratedProject(t, []string{"user", "access", "auth"})

	c := capabilityCommandForTest(&bytes.Buffer{})
	c.SetArgs([]string{"capability", "add", "user", "--dir=" + dir, "--dry-run"})
	err := c.Execute()
	require.ErrorContains(t, err, "already present")

	force := capabilityCommandForTest(&bytes.Buffer{})
	force.SetArgs([]string{"capability", "add", "user", "--dir=" + dir, "--dry-run", "--force"})
	require.NoError(t, force.Execute())
}

// TestCapabilityAddCmdRejectsForeignDirectory 没有 manifest 的目录一律拒绝。
func TestCapabilityAddCmdRejectsForeignDirectory(t *testing.T) {
	c := capabilityCommandForTest(&bytes.Buffer{})
	c.SetArgs([]string{"capability", "add", "dataops", "--dir=" + t.TempDir()})
	err := c.Execute()
	require.ErrorContains(t, err, ".jimu/manifest.json")
}

// TestCapabilityAddCmdRejectsMissingDependency 缺硬依赖的报错必须带「先加依赖」的指引。
func TestCapabilityAddCmdRejectsMissingDependency(t *testing.T) {
	dir := fakeGeneratedProject(t, []string{"user"})
	c := capabilityCommandForTest(&bytes.Buffer{})
	c.SetArgs([]string{"capability", "add", "passkey", "--dir=" + dir, "--dry-run"})
	err := c.Execute()
	require.ErrorContains(t, err, `requires "auth"`)
	require.ErrorContains(t, err, "add it first")
}
