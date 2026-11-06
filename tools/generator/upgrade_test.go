package generator

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"jimu/tools/generator/manifest"
	upgradeplan "jimu/tools/generator/upgrade"
	"jimu/tools/generator/workspace"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPlanUpgradeCleanProjectIsUpToDate(t *testing.T) {
	dir := generateForTest(t, NewOptions{Template: "minimal"})
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
	assert.Equal(t, "minimal", result.Target.Selection.Template)
	assert.NoError(t, manifest.ValidateComplete(result.Target))
}

func TestPlanUpgradeDefaultTidyKeepsGeneratedProjectUpToDate(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "proj")
	_, err := NewProject(NewOptions{
		Dir: dir, Template: "minimal", Module: "example.com/proj", NoSelfCheck: true,
	})
	require.NoError(t, err)

	result, err := PlanUpgrade(UpgradeOptions{Dir: dir, From: FrameworkRoot()})
	require.NoError(t, err)
	assert.Equal(t, upgradeplan.StatusUpToDate, result.Report.Status(), "go mod tidy in the planning copy must not create false diffs")
}

func TestPlanUpgradeUserEditIsConflictAndDoesNotWrite(t *testing.T) {
	dir := generateForTest(t, NewOptions{Template: "minimal"})
	managedFile := filepath.Join(dir, "Makefile")
	require.NoError(t, os.WriteFile(managedFile, []byte("# user-owned Makefile\n"), 0o644))
	before := hashTree(t, dir)

	result, err := PlanUpgrade(UpgradeOptions{Dir: dir, From: FrameworkRoot(), NoTidy: true})
	require.NoError(t, err)
	entry := upgradeEntry(t, result.Report, "Makefile")
	assert.Equal(t, upgradeplan.Conflict, entry.Category)
	assert.Equal(t, upgradeplan.ActionKeep, entry.Action)
	assert.Contains(t, entry.Reason, "target template is unchanged")
	assert.Equal(t, upgradeplan.StatusConflict, result.Report.Status())
	assert.Equal(t, before, hashTree(t, dir), "upgrade planning must not write to the project")
}

func TestApplyUpgradeInstallsModifiedFileAndIsIdempotent(t *testing.T) {
	dir := generateForTest(t, NewOptions{Template: "minimal"})
	managedFile := filepath.Join(dir, "Makefile")
	targetContent, err := os.ReadFile(managedFile)
	require.NoError(t, err)
	previousContent := []byte("# v0.9 baseline Makefile\n")
	require.NoError(t, os.WriteFile(managedFile, previousContent, 0o644))
	rewriteBaselineHash(t, dir, "Makefile", previousContent)
	stubUpgradeVerify(t, nil)

	result, err := ApplyUpgrade(UpgradeOptions{Dir: dir, From: FrameworkRoot(), NoTidy: true})
	require.NoError(t, err)
	assert.Contains(t, result.Changed, "Makefile")
	assert.True(t, result.ManifestUpdated)
	assert.True(t, result.Verified)
	assert.Equal(t, targetContent, readFileForTest(t, managedFile))

	loaded, err := workspace.LoadProjectManifest(dir)
	require.NoError(t, err)
	assert.Equal(t, hashBytes(targetContent), loaded.GeneratedFileSHA256["Makefile"])

	repeated, err := ApplyUpgrade(UpgradeOptions{Dir: dir, From: FrameworkRoot(), NoTidy: true})
	require.NoError(t, err)
	assert.Empty(t, repeated.Changed)
	assert.False(t, repeated.ManifestUpdated)
	assert.False(t, repeated.Verified, "an up-to-date project must not rerun verification")
}

func TestApplyUpgradeConflictLeavesProjectByteIdentical(t *testing.T) {
	dir := generateForTest(t, NewOptions{Template: "minimal"})
	require.NoError(t, os.WriteFile(filepath.Join(dir, "Makefile"), []byte("user-owned README\n"), 0o644))
	before := hashTree(t, dir)

	result, err := ApplyUpgrade(UpgradeOptions{Dir: dir, From: FrameworkRoot(), NoTidy: true})
	require.ErrorIs(t, err, ErrUpgradeConflicts)
	require.NotNil(t, result)
	assert.True(t, result.Report.HasConflicts())
	assert.Equal(t, before, hashTree(t, dir))
}

func TestApplyUpgradeVerifyFailureRollsBackByteForByte(t *testing.T) {
	dir := generateForTest(t, NewOptions{Template: "minimal"})
	induceModifiedFile(t, dir, "Makefile")
	before := hashTree(t, dir)
	stubUpgradeVerify(t, errors.New("generated test failed"))

	_, err := ApplyUpgrade(UpgradeOptions{Dir: dir, From: FrameworkRoot(), NoTidy: true})
	require.ErrorContains(t, err, "verify upgraded project")
	require.ErrorContains(t, err, "generated test failed")
	assert.Equal(t, before, hashTree(t, dir))
}

func TestApplyUpgradeInstallFailureRestoresEveryFile(t *testing.T) {
	dir := generateForTest(t, NewOptions{Template: "minimal"})
	induceModifiedFile(t, dir, "Makefile")
	induceModifiedFile(t, dir, "Dockerfile")
	before := hashTree(t, dir)
	stubUpgradeVerify(t, nil)
	original := workspace.UpgradeInstallFile
	t.Cleanup(func() { workspace.UpgradeInstallFile = original })
	calls := 0
	workspace.UpgradeInstallFile = func(staging, root, rel string) error {
		calls++
		if calls == 2 {
			return errors.New("injected disk failure")
		}
		return original(staging, root, rel)
	}

	_, err := ApplyUpgrade(UpgradeOptions{Dir: dir, From: FrameworkRoot(), NoTidy: true})
	require.ErrorContains(t, err, "injected disk failure")
	assert.Equal(t, before, hashTree(t, dir))
}

func TestApplyUpgradeUpdatesManifestWithoutFileChanges(t *testing.T) {
	dir := generateForTest(t, NewOptions{Template: "minimal"})
	doc, err := workspace.LoadProjectManifest(dir)
	require.NoError(t, err)
	doc.Framework.Version = "v0.5.0-test"
	require.NoError(t, manifest.Write(filepath.Join(dir, ".jimu", "manifest.json"), doc))
	stubUpgradeVerify(t, nil)

	result, err := ApplyUpgrade(UpgradeOptions{Dir: dir, From: FrameworkRoot(), NoTidy: true})
	require.NoError(t, err)
	assert.Empty(t, result.Changed)
	assert.True(t, result.ManifestUpdated)
	assert.True(t, result.Verified)
	loaded, err := workspace.LoadProjectManifest(dir)
	require.NoError(t, err)
	assert.NotEqual(t, "v0.5.0-test", loaded.Framework.Version)
}

func TestApplyUpgradePreservesEditedConfigWithoutConflict(t *testing.T) {
	dir := generateForTest(t, NewOptions{With: withProtectedProvider("queue")})
	config := filepath.Join(dir, "configs", "app.yaml")
	content, err := os.ReadFile(config)
	require.NoError(t, err)
	edited := append(slices.Clone(content), []byte("\nuser:\n  value: custom\n")...)
	require.NoError(t, os.WriteFile(config, edited, 0o644))

	result, err := ApplyUpgrade(UpgradeOptions{Dir: dir, From: FrameworkRoot(), NoTidy: true})
	require.NoError(t, err)
	assert.False(t, result.Report.HasConflicts())
	assert.Equal(t, edited, readFileForTest(t, config))
}

// TestApplyUpgradePreservesNonDefaultDrivers `--with` 项目的升级必须按
// composition.drivers 重建，不能把用户选择的非默认驱动静默重置为默认驱动；
// 否则构建/测试会通过，运行时却因编译进错误的驱动而 fail-closed。
func TestApplyUpgradePreservesNonDefaultDrivers(t *testing.T) {
	dir := generateForTest(t, NewOptions{With: "auth,queue:kafka"})
	before, err := workspace.LoadProjectManifest(dir)
	require.NoError(t, err)
	require.Equal(t, []string{"kafka"}, before.Composition.Drivers["queue"])

	stubUpgradeVerify(t, nil)
	result, err := ApplyUpgrade(UpgradeOptions{Dir: dir, From: FrameworkRoot(), NoTidy: true})
	require.NoError(t, err)
	assert.Empty(t, result.Changed, "upgrading an untouched --with project must not change files")

	loaded, err := workspace.LoadProjectManifest(dir)
	require.NoError(t, err)
	assert.Equal(t, []string{"kafka"}, loaded.Composition.Drivers["queue"])
	assert.Equal(t, []string{"kafka"}, loaded.Selection.Drivers["queue"])
	assert.FileExists(t, filepath.Join(dir, "internal", "capabilities", "queue", "kafka", "kafka_queue.go"))
	assert.NoFileExists(t, filepath.Join(dir, "internal", "capabilities", "queue", "redis", "redis_queue.go"))
}

// TestApplyUpgradeRefreshesExistingGeneratedReport 升级改变装配或文件集时，项目里已有的
// `--report` 产物必须同批重算刷新；本来没有报告的项目不得凭空创建。
func TestApplyUpgradeRefreshesExistingGeneratedReport(t *testing.T) {
	dir := generateForTest(t, NewOptions{With: withProtectedProvider("queue"), Report: true})
	reportPath := filepath.Join(dir, filepath.FromSlash(reportRelPath))
	stale := []byte("# stale generated report\n")
	require.NoError(t, os.WriteFile(reportPath, stale, 0o644))
	stubUpgradeVerify(t, nil)

	plan, err := PlanUpgrade(UpgradeOptions{Dir: dir, From: FrameworkRoot(), NoTidy: true})
	require.NoError(t, err)
	entry := upgradeEntry(t, plan.Report, reportRelPath)
	assert.Equal(t, upgradeplan.Modified, entry.Category)
	assert.Equal(t, upgradeplan.ActionInstall, entry.Action)
	assert.Equal(t, stale, readFileForTest(t, reportPath), "planning must not write the report")

	applied, err := ApplyUpgrade(UpgradeOptions{Dir: dir, From: FrameworkRoot(), NoTidy: true})
	require.NoError(t, err)
	assert.Contains(t, applied.Changed, reportRelPath)
	refreshed := readFileForTest(t, reportPath)
	assert.NotEqual(t, stale, refreshed)
	assert.Contains(t, string(refreshed), "# 生成项目编译面报告")

	repeated, err := ApplyUpgrade(UpgradeOptions{Dir: dir, From: FrameworkRoot(), NoTidy: true})
	require.NoError(t, err)
	assert.Empty(t, repeated.Changed, "a refreshed report must be idempotent")

	plain := generateForTest(t, NewOptions{Template: "minimal"})
	stubUpgradeVerify(t, nil)
	applied, err = ApplyUpgrade(UpgradeOptions{Dir: plain, From: FrameworkRoot(), NoTidy: true})
	require.NoError(t, err)
	assert.Empty(t, applied.Changed)
	assert.NoFileExists(t, filepath.Join(plain, filepath.FromSlash(reportRelPath)),
		"upgrade must not create a report that the project never had")
}

func TestPlanUpgradeMergedConfigIsReadOnly(t *testing.T) {
	dir := generateForTest(t, NewOptions{With: withProtectedProvider("queue")})
	config := filepath.Join(dir, "configs", "app.yaml")
	content, err := os.ReadFile(config)
	require.NoError(t, err)
	edited := append(slices.Clone(content), []byte("\nuser:\n  value: custom\n")...)
	require.NoError(t, os.WriteFile(config, edited, 0o644))
	before := hashTree(t, dir)

	result, err := PlanUpgrade(UpgradeOptions{Dir: dir, From: FrameworkRoot(), NoTidy: true})
	require.NoError(t, err)
	assert.False(t, result.Report.HasConflicts())
	assert.Equal(t, before, hashTree(t, dir))
}

func TestApplyUpgradeMigratesV2ManifestForOfficialTemplates(t *testing.T) {
	for _, template := range []string{"minimal", "auth"} {
		t.Run(template, func(t *testing.T) {
			dir := generateForTest(t, NewOptions{Template: template})
			downgradeManifestToV2(t, dir)
			stubUpgradeVerify(t, nil)

			result, err := ApplyUpgrade(UpgradeOptions{Dir: dir, From: FrameworkRoot(), NoTidy: true})
			require.NoError(t, err)
			assert.Empty(t, result.Changed)
			assert.True(t, result.ManifestUpdated)
			assert.True(t, result.Verified)

			loaded, err := workspace.LoadProjectManifest(dir)
			require.NoError(t, err)
			assert.Equal(t, manifest.CurrentSchemaVersion, loaded.SchemaVersion)
			require.NotNil(t, loaded.Composition)

			repeated, err := ApplyUpgrade(UpgradeOptions{Dir: dir, From: FrameworkRoot(), NoTidy: true})
			require.NoError(t, err)
			assert.Empty(t, repeated.Changed)
			assert.False(t, repeated.ManifestUpdated)
		})
	}
}

// downgradeManifestToV2 rewrites a generated project's manifest into the
// read-only v2 shape with its legacy digest, so the upgrade path exercises the
// same compatibility branch as a v0.5 project.
func downgradeManifestToV2(t *testing.T, dir string) {
	t.Helper()
	doc, err := workspace.LoadProjectManifest(dir)
	require.NoError(t, err)
	doc.SchemaVersion = manifest.LegacySchemaVersion
	doc.Composition = nil
	doc.Digest = ""
	content, err := json.Marshal(doc)
	require.NoError(t, err)
	sum := sha256.Sum256(content)
	doc.Digest = "sha256:" + hex.EncodeToString(sum[:])
	content, err = json.Marshal(doc)
	require.NoError(t, err)
	path := filepath.Join(dir, ".jimu", "manifest.json")
	require.NoError(t, os.WriteFile(path, append(content, '\n'), 0o644))
}

func stubUpgradeVerify(t *testing.T, stubErr error) {
	t.Helper()
	original := upgradeVerify
	t.Cleanup(func() { upgradeVerify = original })
	upgradeVerify = func(string) error { return stubErr }
}

func induceModifiedFile(t *testing.T, dir, rel string) {
	t.Helper()
	path := filepath.Join(dir, filepath.FromSlash(rel))
	previous := []byte("v0.5 baseline for " + rel + "\n")
	require.NoError(t, os.WriteFile(path, previous, 0o644))
	rewriteBaselineHash(t, dir, rel, previous)
}

func rewriteBaselineHash(t *testing.T, dir, rel string, content []byte) {
	t.Helper()
	doc, err := workspace.LoadProjectManifest(dir)
	require.NoError(t, err)
	doc.GeneratedFileSHA256[rel] = hashBytes(content)
	require.NoError(t, manifest.Write(filepath.Join(dir, ".jimu", "manifest.json"), doc))
}

func hashBytes(content []byte) string {
	sum := sha256.Sum256(content)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func readFileForTest(t *testing.T, path string) []byte {
	t.Helper()
	content, err := os.ReadFile(path)
	require.NoError(t, err)
	return content
}

func upgradeEntry(t *testing.T, report upgradeplan.Report, path string) upgradeplan.Entry {
	t.Helper()
	index := slices.IndexFunc(report.Entries, func(entry upgradeplan.Entry) bool {
		return entry.Path == path
	})
	require.NotEqual(t, -1, index, "report has no entry for %s", path)
	return report.Entries[index]
}

func TestUpgradeRejectsRetiredTemplates(t *testing.T) {
	for _, template := range []string{"basic-api", "auth-api"} {
		t.Run(template, func(t *testing.T) {
			dir := writeManifestWithTemplate(t, template)
			before := hashTree(t, dir)
			_, err := PlanUpgrade(UpgradeOptions{Dir: dir, From: FrameworkRoot()})
			require.ErrorIs(t, err, ErrRetiredTemplate)
			require.Contains(t, err.Error(), template)
			require.Contains(t, err.Error(), "--template=")
			require.Equal(t, before, hashTree(t, dir), "retired template must not write")
		})
	}
}
