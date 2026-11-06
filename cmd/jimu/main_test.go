package main

import (
	"bytes"
	"os"
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
	assert.Equal(t, []string{"capability", "check", "new", "upgrade", "version"}, names)

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
	cmd.SetArgs([]string{"new", "demo", "--template=minimal", "--dry-run"})

	require.NoError(t, cmd.Execute())
	assert.Contains(t, out.String(), "template     minimal")
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
