package generator

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	upgradeplan "jimu/tools/generator/upgrade"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCheckProjectFastReportsHealthyLifecycleAndIsReadOnly(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "proj")
	_, err := newProjectForTest(t, NewOptions{
		Dir: dir, Template: "minimal", Module: "example.com/proj",
	})
	require.NoError(t, err)
	before := hashTree(t, dir)

	result, err := CheckProject(CheckOptions{Dir: dir, From: FrameworkRoot()})
	require.NoError(t, err)
	assert.Equal(t, CheckStatusPassed, result.Status())
	assert.False(t, result.HasErrors())
	assert.False(t, result.HasWarnings())
	assert.NotEmpty(t, result.ManagedFiles)
	for _, file := range result.ManagedFiles {
		assert.Equal(t, ManagedFileClean, file.Status, file.Path)
	}
	require.NotNil(t, result.Upgrade)
	assert.Equal(t, string(upgradeplan.StatusUpToDate), result.Upgrade.Status)
	assert.NotEmpty(t, result.Upgrade.TargetVersion)
	assert.NotEmpty(t, result.Upgrade.TargetTemplateVersion)
	assert.NotEmpty(t, result.NextActions)
	assert.Equal(t, before, hashTree(t, dir), "fast check must not write the project")
}

func TestCheckProjectFastWarnsOnManagedDriftWithoutFailing(t *testing.T) {
	dir := generateForTest(t, NewOptions{Template: "minimal"})
	require.NoError(t, os.WriteFile(filepath.Join(dir, "README.md"), []byte("local change\n"), 0o644))
	require.NoError(t, os.Remove(filepath.Join(dir, "Makefile")))
	before := hashTree(t, dir)

	result, err := CheckProject(CheckOptions{Dir: dir, From: FrameworkRoot()})
	require.NoError(t, err)
	assert.False(t, result.HasErrors())
	assert.True(t, result.HasWarnings())
	assert.Contains(t, diagnosticCodes(result), "managed_file_modified")
	assert.Contains(t, diagnosticCodes(result), "managed_file_missing")
	assert.Contains(t, diagnosticCodes(result), "upgrade_conflict")
	assert.Equal(t, ManagedFileModified, managedFile(result, "README.md").Status)
	assert.Equal(t, ManagedFileMissing, managedFile(result, "Makefile").Status)
	assert.Equal(t, before, hashTree(t, dir), "drift warnings must not write the project")
}

func TestCheckProjectFastReportsOldFixtureUpgradeTargetWithoutWriting(t *testing.T) {
	fixture := upgradeFixtureCases[0]
	dir := prepareUpgradeFixture(t, fixture, "lifecycle-check")
	before := hashTree(t, dir)

	result, err := CheckProject(CheckOptions{Dir: dir, From: FrameworkRoot()})
	require.NoError(t, err)
	assert.False(t, result.HasErrors())
	require.NotNil(t, result.Upgrade)
	assert.Equal(t, string(upgradeplan.StatusUpgradeAvailable), result.Upgrade.Status)
	assert.NotEqual(t, "v0.5.0", result.Upgrade.TargetVersion)
	assert.NotEmpty(t, result.Upgrade.Entries)
	assert.True(t, slices.ContainsFunc(result.NextActions, func(action string) bool {
		return strings.Contains(action, "upgrade") && strings.Contains(action, "--check")
	}))
	assert.Equal(t, before, hashTree(t, dir), "upgrade availability must not write the project")
}

func TestCheckProjectReturnsStructuredManifestFailure(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".jimu"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".jimu", "manifest.json"), []byte("{"), 0o644))

	result, err := CheckProject(CheckOptions{Dir: dir, From: FrameworkRoot()})
	require.NoError(t, err)
	assert.Equal(t, CheckStatusFailed, result.Status())
	assert.True(t, result.HasErrors())
	assert.Contains(t, diagnosticCodes(result), "manifest_invalid")
	require.NotNil(t, result.Upgrade)
	assert.NotEmpty(t, result.Upgrade.TargetVersion)
}

func TestCheckProjectDeepFailureIsReadOnly(t *testing.T) {
	dir := generateForTest(t, NewOptions{Template: "minimal"})
	testPath := filepath.Join(dir, "internal", "api", "v1", "user_check_failure_test.go")
	require.NoError(t, os.WriteFile(testPath, []byte("package v1\n\nimport \"testing\"\n\nfunc TestUserFailure(t *testing.T) { t.Fatal(\"intentional deep failure\") }\n"), 0o644))
	before := hashTree(t, dir)

	result, err := CheckProject(CheckOptions{Dir: dir, From: FrameworkRoot(), Deep: true})
	require.NoError(t, err)
	assert.Equal(t, CheckStatusFailed, result.Status())
	require.NotNil(t, result.Deep)
	assert.True(t, result.Deep.Enabled)
	assert.False(t, result.Deep.Passed)
	assert.Contains(t, diagnosticCodes(result), "deep_test_failed")
	assert.Equal(t, before, hashTree(t, dir), "deep check must not write the project, even on failure")
}

func TestCheckProjectDeepSupportsCertifiedTemplates(t *testing.T) {
	for _, template := range []string{"minimal", "auth"} {
		t.Run(template, func(t *testing.T) {
			dir := filepath.Join(t.TempDir(), "proj")
			_, err := newProjectForTest(t, NewOptions{
				Dir: dir, Template: template, Module: "example.com/proj",
			})
			require.NoError(t, err)
			before := hashTree(t, dir)

			result, err := CheckProject(CheckOptions{Dir: dir, From: FrameworkRoot(), Deep: true})
			require.NoError(t, err)
			assert.Equal(t, CheckStatusPassed, result.Status(), "diagnostics: %#v", result.Diagnostics)
			require.NotNil(t, result.Deep)
			assert.True(t, result.Deep.Passed)
			if assert.Len(t, result.Deep.Steps, 3) {
				assert.Equal(t, []string{"config", "build", "test"}, []string{
					result.Deep.Steps[0].Name,
					result.Deep.Steps[1].Name,
					result.Deep.Steps[2].Name,
				})
			}
			assert.Equal(t, before, hashTree(t, dir), "deep check must be read-only")
		})
	}
}

func diagnosticCodes(result *CheckResult) []string {
	codes := make([]string, 0, len(result.Diagnostics))
	for _, diagnostic := range result.Diagnostics {
		codes = append(codes, diagnostic.Code)
	}
	return codes
}

func managedFile(result *CheckResult, path string) ManagedFile {
	index := slices.IndexFunc(result.ManagedFiles, func(file ManagedFile) bool {
		return file.Path == path
	})
	if index == -1 {
		return ManagedFile{}
	}
	return result.ManagedFiles[index]
}
