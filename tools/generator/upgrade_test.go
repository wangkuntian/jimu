package generator

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"jimu/tools/generator/manifest"
	upgradeplan "jimu/tools/generator/upgrade"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPlanUpgradeCleanProjectIsUpToDate(t *testing.T) {
	dir := generateForTest(t, NewOptions{Template: "basic-api"})
	result, err := PlanUpgrade(UpgradeOptions{Dir: dir, From: FrameworkRoot(), NoTidy: true})
	require.NoError(t, err)

	assert.Equal(t, upgradeplan.StatusUpToDate, result.Report.Status())
	assert.False(t, result.Report.HasConflicts())
	assert.Positive(t, result.Report.Count(upgradeplan.Untouched))
	assert.Zero(t, result.Report.Count(upgradeplan.Modified))
	assert.Zero(t, result.Report.Count(upgradeplan.Conflict))
	assert.Zero(t, result.Report.Count(upgradeplan.New))
	assert.Zero(t, result.Report.Count(upgradeplan.Deleted))
	assert.Equal(t, dir, result.Dir)
	assert.Equal(t, "example.com/proj", result.Module)
	assert.Equal(t, "basic-api", result.Target.Selection.Template)
	assert.NoError(t, manifest.ValidateComplete(result.Target))
}

func TestPlanUpgradeDefaultTidyKeepsGeneratedProjectUpToDate(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "proj")
	_, err := NewProject(NewOptions{
		Dir: dir, Template: "basic-api", Module: "example.com/proj", NoSelfCheck: true,
	})
	require.NoError(t, err)

	result, err := PlanUpgrade(UpgradeOptions{Dir: dir, From: FrameworkRoot()})
	require.NoError(t, err)
	assert.Equal(t, upgradeplan.StatusUpToDate, result.Report.Status(), "go mod tidy in the planning copy must not create false diffs")
}

func TestPlanUpgradeUserEditIsConflictAndDoesNotWrite(t *testing.T) {
	dir := generateForTest(t, NewOptions{Template: "basic-api"})
	readme := filepath.Join(dir, "README.md")
	require.NoError(t, os.WriteFile(readme, []byte("user-owned README\n"), 0o644))
	before := hashTree(t, dir)

	result, err := PlanUpgrade(UpgradeOptions{Dir: dir, From: FrameworkRoot(), NoTidy: true})
	require.NoError(t, err)
	entry := upgradeEntry(t, result.Report, "README.md")
	assert.Equal(t, upgradeplan.Conflict, entry.Category)
	assert.Equal(t, upgradeplan.ActionKeep, entry.Action)
	assert.Contains(t, entry.Reason, "target template is unchanged")
	assert.Equal(t, upgradeplan.StatusConflict, result.Report.Status())
	assert.Equal(t, before, hashTree(t, dir), "upgrade planning must not write to the project")
}

func upgradeEntry(t *testing.T, report upgradeplan.Report, path string) upgradeplan.Entry {
	t.Helper()
	index := slices.IndexFunc(report.Entries, func(entry upgradeplan.Entry) bool {
		return entry.Path == path
	})
	require.NotEqual(t, -1, index, "report has no entry for %s", path)
	return report.Entries[index]
}
