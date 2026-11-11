package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"jimu/tools/generator"
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

func TestCheckCmdJSONReportsLifecycleAndKeepsFailuresMachineReadable(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "proj")
	_, err := generator.NewProject(generator.NewOptions{
		Dir: dir, Template: "minimal", Module: "example.com/proj", NoSelfCheck: true,
	})
	require.NoError(t, err)
	before := upgradeTreeDigest(t, dir)

	var out bytes.Buffer
	c := checkCommandForTest(&out)
	c.SetArgs([]string{"check", dir, "--json"})
	require.NoError(t, c.Execute())
	firstJSON := out.String()
	var report struct {
		SchemaVersion int `json:"schema_version"`
		Status        string
		Summary       struct {
			Errors int `json:"errors"`
		} `json:"summary"`
		ManagedFiles []struct {
			Path   string
			Status string
		} `json:"managed_files"`
		Upgrade struct {
			Status        string `json:"status"`
			TargetVersion string `json:"target_version"`
		} `json:"upgrade"`
		NextActions []string `json:"next_actions"`
	}
	require.NoError(t, json.Unmarshal(out.Bytes(), &report))
	assert.Equal(t, 1, report.SchemaVersion)
	assert.Equal(t, "passed", report.Status)
	assert.Zero(t, report.Summary.Errors)
	assert.NotEmpty(t, report.ManagedFiles)
	assert.Equal(t, "up-to-date", report.Upgrade.Status)
	assert.NotEmpty(t, report.Upgrade.TargetVersion)
	assert.NotEmpty(t, report.NextActions)
	assert.Equal(t, before, upgradeTreeDigest(t, dir), "check --json must be read-only")

	var repeated bytes.Buffer
	repeatedCmd := checkCommandForTest(&repeated)
	repeatedCmd.SetArgs([]string{"check", dir, "--json"})
	require.NoError(t, repeatedCmd.Execute())
	assert.Equal(t, firstJSON, repeated.String(), "repeated check JSON must be byte-for-byte stable")

	path := filepath.Join(dir, ".jimu", "manifest.json")
	require.NoError(t, os.WriteFile(path, []byte("{"), 0o644))
	var invalidOut bytes.Buffer
	invalid := checkCommandForTest(&invalidOut)
	invalid.SetArgs([]string{"check", dir, "--json"})
	require.ErrorContains(t, invalid.Execute(), "check failed")
	var invalidReport struct {
		Status      string `json:"status"`
		Diagnostics []struct {
			Code     string `json:"code"`
			Severity string `json:"severity"`
		} `json:"diagnostics"`
	}
	require.NoError(t, json.Unmarshal(invalidOut.Bytes(), &invalidReport))
	assert.Equal(t, "failed", invalidReport.Status)
	require.NotEmpty(t, invalidReport.Diagnostics)
	assert.Equal(t, "manifest_invalid", invalidReport.Diagnostics[0].Code)
	assert.Equal(t, "error", invalidReport.Diagnostics[0].Severity)
}

func TestCheckCmdDeepJSONIsDeterministic(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "proj")
	_, err := generator.NewProject(generator.NewOptions{
		Dir: dir, Template: "minimal", Module: "example.com/proj", NoSelfCheck: true,
	})
	require.NoError(t, err)

	var first bytes.Buffer
	firstCmd := checkCommandForTest(&first)
	firstCmd.SetArgs([]string{"check", dir, "--json", "--deep"})
	require.NoError(t, firstCmd.Execute())

	var second bytes.Buffer
	secondCmd := checkCommandForTest(&second)
	secondCmd.SetArgs([]string{"check", dir, "--json", "--deep"})
	require.NoError(t, secondCmd.Execute())

	assert.Equal(t, first.String(), second.String(), "deep check JSON must be deterministic")
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
			for name := range doc.Selection.Drivers {
				if !slices.Contains(test.capabilities, name) {
					delete(doc.Selection.Drivers, name)
				}
			}
			for name := range doc.Composition.Drivers {
				if !slices.Contains(test.capabilities, name) {
					delete(doc.Composition.Drivers, name)
				}
			}
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
