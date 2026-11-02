package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"jimu/tools/generator"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func upgradeCommandForTest(out *bytes.Buffer) *cobra.Command {
	root := &cobra.Command{Use: "jimu"}
	root.SetOut(out)
	root.SetErr(out)
	cmd := newUpgradeCmd()
	cmd.SetOut(out)
	cmd.SetErr(out)
	root.AddCommand(cmd)
	return root
}

func TestUpgradeCmdRequiresMode(t *testing.T) {
	dir := fakeGeneratedProject(t, []string{"user", "access", "auth"})
	var out bytes.Buffer
	c := upgradeCommandForTest(&out)
	c.SetArgs([]string{"upgrade", dir})
	require.ErrorContains(t, c.Execute(), "--dry-run or --check")
}

func TestUpgradeCmdRejectsBothModes(t *testing.T) {
	dir := fakeGeneratedProject(t, []string{"user", "access", "auth"})
	var out bytes.Buffer
	c := upgradeCommandForTest(&out)
	c.SetArgs([]string{"upgrade", dir, "--dry-run", "--check"})
	require.ErrorContains(t, c.Execute(), "mutually exclusive")
}

func TestUpgradeCmdDryRunReportsConflictsWithoutWriting(t *testing.T) {
	dir := fakeGeneratedProject(t, []string{"user", "access", "auth"})
	manifest := filepath.Join(dir, ".jimu", "manifest.json")
	before, err := os.ReadFile(manifest)
	require.NoError(t, err)

	var out bytes.Buffer
	c := upgradeCommandForTest(&out)
	c.SetArgs([]string{"upgrade", dir, "--dry-run", "--no-tidy"})
	require.NoError(t, c.Execute())

	output := out.String()
	assert.Contains(t, output, "dry-run:")
	assert.Contains(t, output, "status       conflict")
	assert.Contains(t, output, "conflict")
	assert.Contains(t, output, "action: keep")
	after, err := os.ReadFile(manifest)
	require.NoError(t, err)
	assert.Equal(t, before, after, "upgrade --dry-run must not write the project")
}

func TestUpgradeCmdCheckFailsOnConflicts(t *testing.T) {
	dir := fakeGeneratedProject(t, []string{"user", "access", "auth"})
	manifest := filepath.Join(dir, ".jimu", "manifest.json")
	before, err := os.ReadFile(manifest)
	require.NoError(t, err)
	var out bytes.Buffer
	c := upgradeCommandForTest(&out)
	c.SetArgs([]string{"upgrade", dir, "--check", "--no-tidy"})
	err = c.Execute()
	require.ErrorContains(t, err, "upgrade plan has conflicts")
	assert.Contains(t, out.String(), "check:")
	assert.Contains(t, out.String(), "conflict")
	after, readErr := os.ReadFile(manifest)
	require.NoError(t, readErr)
	assert.Equal(t, before, after, "upgrade --check must not write the project")
}

func TestUpgradeCmdCheckPassesOnCleanProject(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "proj")
	_, err := generator.NewProject(generator.NewOptions{
		Dir: dir, Template: "basic-api", Module: "example.com/proj", NoTidy: true, NoSelfCheck: true,
	})
	require.NoError(t, err)

	var out bytes.Buffer
	c := upgradeCommandForTest(&out)
	c.SetArgs([]string{"upgrade", dir, "--check", "--no-tidy"})
	require.NoError(t, c.Execute())
	assert.Contains(t, out.String(), "check:")
	assert.Contains(t, out.String(), "status       up-to-date")
	assert.NotContains(t, out.String(), "action: keep")
}
