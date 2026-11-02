package main

import (
	"bytes"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"jimu/tools/generator/manifest"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func checkCommandForTest(out *bytes.Buffer) *cobra.Command {
	// 根命令故意不设 Silence*，由 check 子命令自己保证不打印 Usage/重复错误。
	root := &cobra.Command{Use: "jimu"}
	root.SetOut(out)
	root.SetErr(out)
	cmd := newCheckCmd()
	cmd.SetOut(out)
	cmd.SetErr(out)
	root.AddCommand(cmd)
	return root
}

func TestCheckCmdAcceptsValidProject(t *testing.T) {
	dir := fakeGeneratedProject(t, []string{"user", "access", "auth"})
	before, err := os.ReadFile(filepath.Join(dir, ".jimu", "manifest.json"))
	require.NoError(t, err)

	var out bytes.Buffer
	c := checkCommandForTest(&out)
	c.SetArgs([]string{"check", dir})
	require.NoError(t, c.Execute())
	assert.Contains(t, out.String(), "check passed")
	assert.Contains(t, out.String(), "valid")

	after, err := os.ReadFile(filepath.Join(dir, ".jimu", "manifest.json"))
	require.NoError(t, err)
	assert.Equal(t, before, after, "jimu check 必须只读")
}

func TestCheckCmdRejectsInvalidProject(t *testing.T) {
	tests := []struct {
		name         string
		capabilities []string
		wantCode     string
	}{
		{
			name:         "missing protected middleware provider",
			capabilities: []string{"user"},
			wantCode:     "middleware_provider_missing",
		},
		{
			name:         "missing hard dependency",
			capabilities: []string{"auth"},
			wantCode:     "missing_dependency",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			dir := fakeGeneratedProject(t, []string{"user", "access", "auth", "passkey"})
			path := filepath.Join(dir, ".jimu", "manifest.json")
			doc, err := manifest.Load(path)
			require.NoError(t, err)
			doc.Selection.Capabilities = test.capabilities
			doc.Composition.Requested = slices.Clone(test.capabilities)
			doc.Composition.Resolved = slices.Clone(test.capabilities)
			doc.Composition.AutoAdded = []string{}
			require.NoError(t, manifest.Write(path, doc))

			var out bytes.Buffer
			c := checkCommandForTest(&out)
			c.SetArgs([]string{"check", dir})
			err = c.Execute()
			require.Error(t, err, "非法项目必须退出非 0")
			assert.Contains(t, out.String(), "check failed")
			assert.Contains(t, out.String(), test.wantCode)
			assert.NotContains(t, out.String(), "Usage:")
			assert.NotContains(t, out.String(), "Error:")
		})
	}
}

func TestCheckCmdRejectsDirectoryWithoutManifest(t *testing.T) {
	var out bytes.Buffer
	c := checkCommandForTest(&out)
	c.SetArgs([]string{"check", t.TempDir()})
	require.ErrorContains(t, c.Execute(), ".jimu/manifest.json")
}

func TestCheckCmdReportsInvalidPlan(t *testing.T) {
	dir := fakeGeneratedProject(t, []string{"user", "access", "auth"})
	path := filepath.Join(dir, ".jimu", "manifest.json")
	doc, err := manifest.Load(path)
	require.NoError(t, err)
	doc.GeneratedFiles = nil
	doc.GeneratedFileSHA256 = map[string]string{}
	require.NoError(t, manifest.Write(path, doc))

	var out bytes.Buffer
	c := checkCommandForTest(&out)
	c.SetArgs([]string{"check", dir})
	require.Error(t, c.Execute())
	assert.Contains(t, out.String(), "invalid_plan")
}
