package generator

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"jimu/tools/generator/manifest"
	upgradeplan "jimu/tools/generator/upgrade"
	"jimu/tools/generator/workspace"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 升级事务测试不再依赖 v0.5 快照：旧模板（basic-api/auth-api）已在 v0.10.0 退役，
// 其渲染路径与 fixture 一并删除。改为现场生成 auth 预设项目，再验证同一套事务语义
// （备份 / 逐文件安装 / 失败回滚 / 冲突检测）。
const upgradeFixturePreset = "auth"

type upgradeFixtureCase struct {
	template           string
	module             string
	generatedFiles     int
	securityChanges    []string
	securityRouterPath string
}

// upgradeFixtureCases 只保留 auth 预设：它是事务测试的代表形态（有迁移、有中间件）。
var upgradeFixtureCases = []upgradeFixtureCase{
	{template: upgradeFixturePreset, module: "example.com/upgrade-auth"},
}

// prepareUpgradeFixture 现场生成一个预设项目作为升级基线。
func prepareUpgradeFixture(t *testing.T, fixture upgradeFixtureCase, scenario string) string {
	t.Helper()
	t.Logf("upgrade fixture: preset=%s scenario=%s", fixture.template, scenario)
	dir := filepath.Join(t.TempDir(), fixture.template)
	_, err := newProjectForTest(t, NewOptions{
		Dir: dir, Template: fixture.template, Module: fixture.module, NoTidy: true,
	})
	require.NoError(t, err, "preset=%s scenario=%s generate", fixture.template, scenario)
	return dir
}

// TestUpgradeFixtures 是 v0.5 -> 当前版本的发布门禁。每个子测试先复制固定 fixture，
// 再独立执行一个场景；失败时的测试名与日志都带 project/scenario。
func TestUpgradeFixtures(t *testing.T) {
	requireHeavyMatrix(t)

	for _, fixture := range upgradeFixtureCases {
		fixture := fixture
		t.Run(fixture.template, func(t *testing.T) {
			t.Run("untouched", func(t *testing.T) {
				testUpgradeFixtureUntouched(t, fixture)
			})
			t.Run("user-modified-conflict", func(t *testing.T) {
				testUpgradeFixtureUserModified(t, fixture)
			})
			t.Run("user-owned-preserved", func(t *testing.T) {
				testUpgradeFixtureUserOwnedPreserved(t, fixture)
			})
			t.Run("user-deleted-conflict", func(t *testing.T) {
				testUpgradeFixtureUserDeleted(t, fixture)
			})
			t.Run("target-deleted", func(t *testing.T) {
				testUpgradeFixtureTargetDeleted(t, fixture)
			})
			t.Run("repeated-upgrade", func(t *testing.T) {
				testUpgradeFixtureRepeated(t, fixture)
			})
			t.Run("rollback-on-verify-failure", func(t *testing.T) {
				testUpgradeFixtureRollback(t, fixture)
			})
		})
	}
}

func testUpgradeFixtureUntouched(t *testing.T, fixture upgradeFixtureCase) {
	dir := prepareUpgradeFixture(t, fixture, "untouched")
	before := hashTree(t, dir)

	plan, err := PlanUpgrade(UpgradeOptions{Dir: dir, From: FrameworkRoot()})
	require.NoError(t, err, "project=%s scenario=untouched plan", fixture.template)
	assert.Equal(t, before, hashTree(t, dir), "project=%s scenario=untouched planning must be read-only", fixture.template)
	assert.False(t, plan.Report.HasConflicts())
	assert.Equal(t, upgradeplan.StatusUpgradeAvailable, plan.Report.Status())
	assert.Equal(t, fixture.generatedFiles-len(fixture.securityChanges), plan.Report.Count(upgradeplan.Untouched))
	assert.Equal(t, len(fixture.securityChanges), plan.Report.Count(upgradeplan.Modified))
	assert.Zero(t, plan.Report.Count(upgradeplan.New))
	assert.Zero(t, plan.Report.Count(upgradeplan.Deleted))
	for _, rel := range fixture.securityChanges {
		entry := upgradeEntry(t, plan.Report, rel)
		assert.Equal(t, upgradeplan.Modified, entry.Category, "project=%s path=%s", fixture.template, rel)
		assert.Equal(t, upgradeplan.ActionInstall, entry.Action, "project=%s path=%s", fixture.template, rel)
	}

	applied, err := ApplyUpgrade(UpgradeOptions{Dir: dir, From: FrameworkRoot()})
	require.NoError(t, err, "project=%s scenario=untouched apply", fixture.template)
	assert.True(t, applied.Verified)
	assert.True(t, applied.ManifestUpdated)
	assert.ElementsMatch(t, fixture.securityChanges, applied.Changed)
	assert.Contains(t, string(readFileForTest(t, filepath.Join(dir, "internal/platform/http/middleware.go"))), "func SecurityHeaders()")
	assert.Contains(t, string(readFileForTest(t, filepath.Join(dir, fixture.securityRouterPath))), "platformhttp.SecurityHeaders()")

	loaded, err := workspace.LoadProjectManifest(dir)
	require.NoError(t, err)
	assert.Equal(t, manifest.CurrentSchemaVersion, loaded.SchemaVersion)
	require.NotNil(t, loaded.Composition)
	assert.Equal(t, manifest.CompositionSupportCertified, loaded.Composition.Support)
	assertUpgradeManifestHashes(t, dir, loaded)
}

func testUpgradeFixtureUserModified(t *testing.T, fixture upgradeFixtureCase) {
	dir := prepareUpgradeFixture(t, fixture, "user-modified-conflict")
	readme := filepath.Join(dir, "README.md")
	userContent := []byte("# user-owned README\n")
	require.NoError(t, os.WriteFile(readme, userContent, 0o644))
	before := hashTree(t, dir)

	applied, err := ApplyUpgrade(UpgradeOptions{Dir: dir, From: FrameworkRoot()})
	require.ErrorIs(t, err, ErrUpgradeConflicts, "project=%s scenario=user-modified-conflict", fixture.template)
	require.NotNil(t, applied)
	entry := upgradeEntry(t, applied.Report, "README.md")
	assert.Equal(t, upgradeplan.Conflict, entry.Category)
	assert.Equal(t, upgradeplan.ActionKeep, entry.Action)
	assert.Equal(t, userContent, readFileForTest(t, readme))
	assert.Equal(t, before, hashTree(t, dir), "a conflicted upgrade must not write any file")
}

func testUpgradeFixtureUserOwnedPreserved(t *testing.T, fixture upgradeFixtureCase) {
	dir := prepareUpgradeFixture(t, fixture, "user-owned-preserved")
	owned := filepath.Join(dir, "docs", "OWNED.md")
	require.NoError(t, os.MkdirAll(filepath.Dir(owned), 0o755))
	require.NoError(t, os.WriteFile(owned, []byte("# user-owned file\n"), 0o644))
	stubUpgradeVerify(t, nil)

	applied, err := ApplyUpgrade(UpgradeOptions{Dir: dir, From: FrameworkRoot()})
	require.NoError(t, err, "project=%s scenario=user-owned-preserved", fixture.template)
	assert.True(t, applied.ManifestUpdated)
	assert.ElementsMatch(t, fixture.securityChanges, applied.Changed)
	assert.Equal(t, []byte("# user-owned file\n"), readFileForTest(t, owned))
	loaded, err := workspace.LoadProjectManifest(dir)
	require.NoError(t, err)
	assert.NotContains(t, loaded.GeneratedFiles, "docs/OWNED.md")
}

func testUpgradeFixtureUserDeleted(t *testing.T, fixture upgradeFixtureCase) {
	dir := prepareUpgradeFixture(t, fixture, "user-deleted-conflict")
	readme := filepath.Join(dir, "README.md")
	require.NoError(t, os.Remove(readme))
	before := hashTree(t, dir)

	applied, err := ApplyUpgrade(UpgradeOptions{Dir: dir, From: FrameworkRoot()})
	require.ErrorIs(t, err, ErrUpgradeConflicts, "project=%s scenario=user-deleted-conflict", fixture.template)
	require.NotNil(t, applied)
	entry := upgradeEntry(t, applied.Report, "README.md")
	assert.Equal(t, upgradeplan.Conflict, entry.Category)
	assert.Equal(t, upgradeplan.ActionKeep, entry.Action)
	assert.Empty(t, entry.LocalSHA256)
	assert.NoFileExists(t, readme)
	assert.Equal(t, before, hashTree(t, dir), "a conflicted upgrade must not recreate a deleted file")
}

func testUpgradeFixtureTargetDeleted(t *testing.T, fixture upgradeFixtureCase) {
	dir := prepareUpgradeFixture(t, fixture, "target-deleted")
	legacy := "legacy/REMOVED.md"
	legacyPath := filepath.Join(dir, filepath.FromSlash(legacy))
	require.NoError(t, os.MkdirAll(filepath.Dir(legacyPath), 0o755))
	require.NoError(t, os.WriteFile(legacyPath, []byte("legacy file\n"), 0o644))

	doc, err := workspace.LoadProjectManifest(dir)
	require.NoError(t, err)
	doc.GeneratedFiles = append(doc.GeneratedFiles, legacy)
	hashes, err := manifest.HashFiles(dir, []string{legacy})
	require.NoError(t, err)
	doc.GeneratedFileSHA256[legacy] = hashes[legacy]
	require.NoError(t, manifest.Write(filepath.Join(dir, ".jimu", "manifest.json"), doc))

	plan, err := PlanUpgrade(UpgradeOptions{Dir: dir, From: FrameworkRoot()})
	require.NoError(t, err, "project=%s scenario=target-deleted plan", fixture.template)
	entry := upgradeEntry(t, plan.Report, legacy)
	assert.Equal(t, upgradeplan.Deleted, entry.Category)
	assert.Equal(t, upgradeplan.ActionKeep, entry.Action)

	stubUpgradeVerify(t, nil)
	applied, err := ApplyUpgrade(UpgradeOptions{Dir: dir, From: FrameworkRoot()})
	require.NoError(t, err, "project=%s scenario=target-deleted apply", fixture.template)
	assert.True(t, applied.ManifestUpdated)
	assert.FileExists(t, legacyPath, "target-deleted files must remain in the project")

	loaded, err := workspace.LoadProjectManifest(dir)
	require.NoError(t, err)
	assert.NotContains(t, loaded.GeneratedFiles, legacy)
}

func testUpgradeFixtureRepeated(t *testing.T, fixture upgradeFixtureCase) {
	dir := prepareUpgradeFixture(t, fixture, "repeated-upgrade")
	stubUpgradeVerify(t, nil)

	first, err := ApplyUpgrade(UpgradeOptions{Dir: dir, From: FrameworkRoot()})
	require.NoError(t, err, "project=%s scenario=repeated-upgrade first apply", fixture.template)
	assert.ElementsMatch(t, fixture.securityChanges, first.Changed)
	assert.True(t, first.ManifestUpdated)
	assert.True(t, first.Verified)
	afterFirst := hashTree(t, dir)

	second, err := ApplyUpgrade(UpgradeOptions{Dir: dir, From: FrameworkRoot()})
	require.NoError(t, err, "project=%s scenario=repeated-upgrade second apply", fixture.template)
	assert.Empty(t, second.Changed)
	assert.False(t, second.ManifestUpdated)
	assert.False(t, second.Verified, "an up-to-date project must not rerun verification")
	assert.Equal(t, upgradeplan.StatusUpToDate, second.Report.Status())
	assert.Equal(t, afterFirst, hashTree(t, dir))
}

func testUpgradeFixtureRollback(t *testing.T, fixture upgradeFixtureCase) {
	dir := prepareUpgradeFixture(t, fixture, "rollback-on-verify-failure")
	before := hashTree(t, dir)
	stubUpgradeVerify(t, errors.New("fixture verification failed"))

	_, err := ApplyUpgrade(UpgradeOptions{Dir: dir, From: FrameworkRoot()})
	require.ErrorContains(t, err, "verify upgraded project", "project=%s scenario=rollback-on-verify-failure", fixture.template)
	require.ErrorContains(t, err, "fixture verification failed")
	assert.Equal(t, before, hashTree(t, dir), "a failed verification must roll back byte for byte")

	var raw struct {
		SchemaVersion int `json:"schema_version"`
	}
	content, readErr := os.ReadFile(filepath.Join(dir, ".jimu", "manifest.json"))
	require.NoError(t, readErr)
	require.NoError(t, json.Unmarshal(content, &raw))
	assert.Equal(t, manifest.LegacySchemaVersion, raw.SchemaVersion, "rollback must restore the v2 manifest")
}

func assertUpgradeManifestHashes(t *testing.T, dir string, doc manifest.Document) {
	t.Helper()
	hashes, err := manifest.HashFiles(dir, doc.GeneratedFiles)
	require.NoError(t, err)
	assert.Equal(t, doc.GeneratedFileSHA256, hashes, "upgraded manifest must hash every generated file")
}
