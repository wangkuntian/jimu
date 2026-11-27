package main

import (
	"bytes"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"jimu"
	"jimu/tools/generator"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReleaseCLIExposesOnlyScaffoldCommands(t *testing.T) {
	cmd, cleanup := newRootCommand()
	require.NotNil(t, cmd)
	if cleanup != nil {
		t.Cleanup(func() { _ = cleanup() })
	}

	var names []string
	for _, child := range cmd.Commands() {
		names = append(names, child.Name())
	}
	sort.Strings(names)
	assert.Equal(t, []string{"capability", "check", "new", "tui", "upgrade", "version"}, names)

	capabilityCmd, _, err := cmd.Find([]string{"capability"})
	require.NoError(t, err)
	var capabilityNames []string
	for _, child := range capabilityCmd.Commands() {
		capabilityNames = append(capabilityNames, child.Name())
	}
	sort.Strings(capabilityNames)
	assert.Equal(t, []string{"add"}, capabilityNames)
}

func TestReleaseCLINewOutsideCheckoutDryRun(t *testing.T) {
	repoRoot := generator.FrameworkRoot()
	require.NotEmpty(t, repoRoot)
	t.Chdir(t.TempDir())

	cmd, cleanup := newRootCommand()
	require.NotNil(t, cmd)
	if cleanup != nil {
		t.Cleanup(func() { _ = cleanup() })
	}
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"new", "demo", "--profile=minimal", "--dry-run"})

	require.NoError(t, cmd.Execute())
	assert.Contains(t, out.String(), "profile      minimal")
	_, err := os.Stat("demo")
	assert.True(t, os.IsNotExist(err))
}

func TestReleaseCLIVersionPrintsAllMetadata(t *testing.T) {
	oldVersion := version
	oldBundleVersion := jimuassets.BundleVersion
	oldContractVersion := jimuassets.ContractVersion
	oldCommit := jimuassets.BundleCommit
	t.Cleanup(func() {
		version = oldVersion
		jimuassets.BundleVersion = oldBundleVersion
		jimuassets.ContractVersion = oldContractVersion
		jimuassets.BundleCommit = oldCommit
	})

	version = "v9.9.9"
	jimuassets.BundleVersion = "v9.9.9"
	jimuassets.ContractVersion = "v0.1"
	jimuassets.BundleCommit = "deadbeef"

	cmd, cleanup := newRootCommand()
	require.NotNil(t, cmd)
	if cleanup != nil {
		t.Cleanup(func() { _ = cleanup() })
	}
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"version"})

	require.NoError(t, cmd.Execute())
	assert.Equal(t, "jimu v9.9.9\ntemplate bundle v9.9.9\ntemplate contract v0.1\ncommit deadbeef\n", out.String())
}

func TestReleaseCLINoArgsAndTUIStayDirectWithoutTTY(t *testing.T) {
	for _, args := range [][]string{
		{},
		{"tui"},
		{"new"},
		{"capability", "add"},
	} {
		cmd, cleanup := newRootCommand()
		if cleanup != nil {
			t.Cleanup(func() { _ = cleanup() })
		}
		var out bytes.Buffer
		cmd.SetIn(bytes.NewBuffer(nil))
		cmd.SetOut(&out)
		cmd.SetErr(&out)
		cmd.SetArgs(args)

		err := cmd.Execute()
		if len(args) == 0 {
			require.NoError(t, err)
			assert.Contains(t, out.String(), "Usage:")
			continue
		}
		require.Error(t, err, "args %v must not open a non-TTY TUI", args)
	}
}

func TestReleaseCLICheckAndUpgradeNoArgsStayDirectWithoutTTY(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "proj")
	_, err := generator.NewProject(generator.NewOptions{
		Dir: dir, Template: "minimal", Module: "example.com/proj", NoTidy: true, NoSelfCheck: true,
	})
	require.NoError(t, err)
	t.Chdir(dir)

	for _, args := range [][]string{
		{"check"},
		{"upgrade", "--check", "--no-tidy"},
	} {
		cmd, cleanup := newRootCommand()
		if cleanup != nil {
			t.Cleanup(func() { _ = cleanup() })
		}
		var out bytes.Buffer
		cmd.SetIn(bytes.NewBuffer(nil))
		cmd.SetOut(&out)
		cmd.SetErr(&out)
		cmd.SetArgs(args)
		require.NoError(t, cmd.Execute(), "args %v", args)
		assert.NotContains(t, out.String(), "Jimu CLI", "non-TTY must stay on the direct path")
	}
}

func TestReleaseCLIColorFlag(t *testing.T) {
	cmd, cleanup := newRootCommand()
	require.NotNil(t, cmd)
	if cleanup != nil {
		t.Cleanup(func() { _ = cleanup() })
	}
	require.False(t, noColorFlag(cmd))
	require.NoError(t, cmd.PersistentFlags().Set("no-color", "true"))
	require.True(t, noColorFlag(cmd))
}

func TestReleaseCLINonTTYOutputHasNoANSI(t *testing.T) {
	cmd, cleanup := newRootCommand()
	require.NotNil(t, cmd)
	if cleanup != nil {
		t.Cleanup(func() { _ = cleanup() })
	}
	var out bytes.Buffer
	cmd.SetIn(bytes.NewBuffer(nil))
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"new", filepath.Join(t.TempDir(), "proj"), "--dry-run", "--no-color"})

	require.NoError(t, cmd.Execute())
	assert.NotContains(t, out.String(), "\x1b[")
}
