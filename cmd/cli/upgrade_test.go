package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io/fs"
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

func TestUpgradeCmdDefaultsToReadOnlyNoOpWhenUpToDate(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "proj")
	_, err := generator.NewProject(generator.NewOptions{
		Dir: dir, Template: "basic-api", Module: "example.com/proj", NoTidy: true, NoSelfCheck: true,
	})
	require.NoError(t, err)
	manifest := filepath.Join(dir, ".jimu", "manifest.json")
	before, err := os.ReadFile(manifest)
	require.NoError(t, err)

	var out bytes.Buffer
	c := upgradeCommandForTest(&out)
	c.SetArgs([]string{"upgrade", dir, "--no-tidy"})
	require.NoError(t, c.Execute())
	assert.Contains(t, out.String(), "upgrade:")
	assert.Contains(t, out.String(), "status       up-to-date")
	assert.Contains(t, out.String(), "no changes")
	after, err := os.ReadFile(manifest)
	require.NoError(t, err)
	assert.Equal(t, before, after)
}

func TestUpgradeCmdDefaultAbortsOnConflictsWithoutWriting(t *testing.T) {
	dir := fakeGeneratedProject(t, []string{"user", "access", "auth"})
	manifest := filepath.Join(dir, ".jimu", "manifest.json")
	before, err := os.ReadFile(manifest)
	require.NoError(t, err)

	var out bytes.Buffer
	c := upgradeCommandForTest(&out)
	c.SetArgs([]string{"upgrade", dir, "--no-tidy"})
	err = c.Execute()
	require.ErrorContains(t, err, "upgrade plan has conflicts")
	assert.Contains(t, out.String(), "upgrade:")
	assert.Contains(t, out.String(), "status       conflict")
	after, readErr := os.ReadFile(manifest)
	require.NoError(t, readErr)
	assert.Equal(t, before, after)
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

func TestUpgradeCmdJSONDiffReportsConflictWithoutWriting(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "proj")
	_, err := generator.NewProject(generator.NewOptions{
		Dir: dir, Template: "basic-api", Module: "example.com/proj", NoTidy: true, NoSelfCheck: true,
	})
	require.NoError(t, err)
	readme := filepath.Join(dir, "README.md")
	require.NoError(t, os.WriteFile(readme, []byte("local README\n"), 0o644))
	before := upgradeTreeDigest(t, dir)

	var first bytes.Buffer
	c := upgradeCommandForTest(&first)
	c.SetArgs([]string{"upgrade", dir, "--check", "--json", "--diff", "--no-tidy"})
	err = c.Execute()
	require.ErrorContains(t, err, "upgrade plan has conflicts")
	assert.NotContains(t, first.String(), dir)
	assert.NotContains(t, first.String(), `"timestamp"`)

	var report struct {
		SchemaVersion string `json:"schema_version"`
		Mode          string `json:"mode"`
		Status        string `json:"status"`
		Counts        struct {
			Conflict int `json:"conflict"`
		} `json:"counts"`
		Entries []struct {
			Path                     string `json:"path"`
			Category                 string `json:"category"`
			BaselineExists           bool   `json:"baseline_exists"`
			LocalExists              bool   `json:"local_exists"`
			TargetExists             bool   `json:"target_exists"`
			BaselineContentAvailable bool   `json:"baseline_content_available"`
			Diff                     string `json:"diff"`
		} `json:"entries"`
	}
	require.NoError(t, json.Unmarshal(first.Bytes(), &report))
	assert.Equal(t, "v1", report.SchemaVersion)
	assert.Equal(t, "check", report.Mode)
	assert.Equal(t, "conflict", report.Status)
	assert.Equal(t, 1, report.Counts.Conflict)
	require.NotEmpty(t, report.Entries)
	assert.Equal(t, "README.md", report.Entries[0].Path)
	assert.True(t, report.Entries[0].BaselineExists)
	assert.True(t, report.Entries[0].LocalExists)
	assert.True(t, report.Entries[0].TargetExists)
	assert.False(t, report.Entries[0].BaselineContentAvailable)
	assert.Contains(t, report.Entries[0].Diff, "--- local/README.md")
	assert.Contains(t, report.Entries[0].Diff, "+++ target/README.md")
	assert.Equal(t, before, upgradeTreeDigest(t, dir), "JSON diff diagnostics must be read-only")

	var second bytes.Buffer
	repeated := upgradeCommandForTest(&second)
	repeated.SetArgs([]string{"upgrade", dir, "--check", "--json", "--diff", "--no-tidy"})
	require.ErrorContains(t, repeated.Execute(), "upgrade plan has conflicts")
	assert.Equal(t, first.String(), second.String(), "repeated machine output must be byte-for-byte stable")
}

func TestUpgradeCmdHumanDiffAndDiagnosticFlagsRequireReadOnlyMode(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "proj")
	_, err := generator.NewProject(generator.NewOptions{
		Dir: dir, Template: "basic-api", Module: "example.com/proj", NoTidy: true, NoSelfCheck: true,
	})
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "README.md"), []byte("local README\n"), 0o644))

	var out bytes.Buffer
	c := upgradeCommandForTest(&out)
	c.SetArgs([]string{"upgrade", dir, "--dry-run", "--diff", "--no-tidy"})
	require.NoError(t, c.Execute())
	assert.Contains(t, out.String(), "--- local/README.md")
	assert.Contains(t, out.String(), "+++ target/README.md")

	for _, args := range [][]string{
		{"upgrade", dir, "--json", "--no-tidy"},
		{"upgrade", dir, "--diff", "--no-tidy"},
	} {
		rejected := upgradeCommandForTest(&bytes.Buffer{})
		rejected.SetArgs(args)
		require.ErrorContains(t, rejected.Execute(), "require --dry-run or --check")
	}
}

func TestUpgradeCmdJSONDiffReportsBinaryMetadataWithoutPatch(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "proj")
	_, err := generator.NewProject(generator.NewOptions{
		Dir: dir, Template: "basic-api", Module: "example.com/proj", NoTidy: true, NoSelfCheck: true,
	})
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "README.md"), []byte("local\x00payload"), 0o644))

	var out bytes.Buffer
	c := upgradeCommandForTest(&out)
	c.SetArgs([]string{"upgrade", dir, "--check", "--json", "--diff", "--no-tidy"})
	require.ErrorContains(t, c.Execute(), "upgrade plan has conflicts")

	var report struct {
		Entries []struct {
			Path                  string `json:"path"`
			Binary                bool   `json:"binary"`
			Diff                  string `json:"diff"`
			DiffUnavailableReason string `json:"diff_unavailable_reason"`
			LocalSHA256           string `json:"local_sha256"`
			TargetSHA256          string `json:"target_sha256"`
		} `json:"entries"`
	}
	require.NoError(t, json.Unmarshal(out.Bytes(), &report))
	require.NotEmpty(t, report.Entries)
	assert.Equal(t, "README.md", report.Entries[0].Path)
	assert.True(t, report.Entries[0].Binary)
	assert.Empty(t, report.Entries[0].Diff)
	assert.Contains(t, report.Entries[0].DiffUnavailableReason, "binary")
	assert.NotEmpty(t, report.Entries[0].LocalSHA256)
	assert.NotEmpty(t, report.Entries[0].TargetSHA256)
}

func upgradeTreeDigest(t *testing.T, root string) string {
	t.Helper()
	sum := sha256.New()
	require.NoError(t, filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		_, _ = sum.Write([]byte(filepath.ToSlash(rel)))
		_, _ = sum.Write(content)
		return nil
	}))
	return hex.EncodeToString(sum.Sum(nil))
}
