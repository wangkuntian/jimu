package generator

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"jimu/tools/generator/manifest"
	upgradeplan "jimu/tools/generator/upgrade"
	"jimu/tools/generator/workspace"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	upgradeFixtureRoot    = "testdata/upgrade/v0.5"
	upgradeFixtureVersion = "v0.5.0"
	upgradeFixtureCommit  = "13cdc2a2519ed5f776e2c2a9c7e3fb36abfdedc0"
)

type upgradeFixtureCase struct {
	template       string
	module         string
	generatedFiles int
}

var upgradeFixtureCases = []upgradeFixtureCase{
	{template: "basic-api", module: "example.com/upgrade-basic-api", generatedFiles: 22},
	{template: "auth-api", module: "example.com/upgrade-auth-api", generatedFiles: 49},
}

// TestUpgradeFixturesMatchV050 钉住 fixture 的固定来源和内容完整性。这是轻量测试，
// 不执行真实升级；升级场景在 TestUpgradeFixtures 中由重型门禁覆盖。
func TestUpgradeFixturesMatchV050(t *testing.T) {
	for _, fixture := range upgradeFixtureCases {
		fixture := fixture
		t.Run(fixture.template, func(t *testing.T) {
			root := filepath.Join(upgradeFixtureRoot, fixture.template)
			manifestPath := filepath.Join(root, ".jimu", "manifest.json")
			content, err := os.ReadFile(manifestPath)
			require.NoError(t, err)

			var raw struct {
				SchemaVersion int                `json:"schema_version"`
				Framework     manifest.Framework `json:"framework"`
				Selection     manifest.Selection `json:"selection"`
				Composition   json.RawMessage    `json:"composition"`
			}
			require.NoError(t, json.Unmarshal(content, &raw))
			assert.Equal(t, manifest.LegacySchemaVersion, raw.SchemaVersion)
			assert.Empty(t, raw.Composition, "v0.5 fixture must stay schema v2 without composition")
			assert.Equal(t, upgradeFixtureVersion, raw.Framework.Version)
			assert.Equal(t, upgradeFixtureCommit, raw.Framework.Commit)
			assert.False(t, raw.Framework.Dirty)
			assert.Equal(t, fixture.template, raw.Selection.Template)

			doc, err := manifest.Load(manifestPath)
			require.NoError(t, err, "fixture manifest digest must verify")
			assert.Equal(t, fixture.template, doc.Selection.Template)
			assert.Equal(t, manifest.CompositionSupportCertified, doc.Composition.Support)
			assert.Len(t, doc.GeneratedFiles, fixture.generatedFiles)

			hashes, err := manifest.HashFiles(root, doc.GeneratedFiles)
			require.NoError(t, err)
			assert.Equal(t, doc.GeneratedFileSHA256, hashes, "fixture files must match the manifest baseline")

			gomod, err := os.ReadFile(filepath.Join(root, "go.mod"))
			require.NoError(t, err)
			firstLine, _, _ := strings.Cut(string(gomod), "\n")
			assert.Equal(t, "module "+fixture.module, strings.TrimSpace(firstLine))

			expected := append([]string{".jimu/manifest.json"}, doc.GeneratedFiles...)
			var actual []string
			require.NoError(t, filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
				if walkErr != nil {
					return walkErr
				}
				if entry.IsDir() {
					return nil
				}
				rel, err := filepath.Rel(root, path)
				if err != nil {
					return err
				}
				actual = append(actual, filepath.ToSlash(rel))
				return nil
			}))
			assert.ElementsMatch(t, expected, actual, "fixture must contain exactly the managed files and manifest")
		})
	}
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
	assert.Equal(t, upgradeplan.StatusUpToDate, plan.Report.Status())
	assert.Positive(t, plan.Report.Count(upgradeplan.Untouched))
	assert.Zero(t, plan.Report.Count(upgradeplan.Modified))
	assert.Zero(t, plan.Report.Count(upgradeplan.New))
	assert.Zero(t, plan.Report.Count(upgradeplan.Deleted))
	assert.Equal(t, fixture.generatedFiles, plan.Report.Count(upgradeplan.Untouched))

	applied, err := ApplyUpgrade(UpgradeOptions{Dir: dir, From: FrameworkRoot()})
	require.NoError(t, err, "project=%s scenario=untouched apply", fixture.template)
	assert.True(t, applied.Verified)
	assert.True(t, applied.ManifestUpdated)
	assert.Empty(t, applied.Changed, "v0.5 official templates must upgrade without file changes")

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
	assert.Empty(t, applied.Changed)
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
	assert.Empty(t, first.Changed)
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

func prepareUpgradeFixture(t *testing.T, fixture upgradeFixtureCase, scenario string) string {
	t.Helper()
	t.Logf("upgrade fixture: project=%s scenario=%s source=%s", fixture.template, scenario, upgradeFixtureRoot)
	src := filepath.Join(upgradeFixtureRoot, fixture.template)
	dir := filepath.Join(t.TempDir(), fixture.template)
	copied, skipped, err := CopyTree(src, dir, nil)
	require.NoError(t, err, "project=%s scenario=%s copy fixture", fixture.template, scenario)
	require.Empty(t, skipped, "project=%s scenario=%s fixture must not contain symlinks", fixture.template, scenario)
	require.Equal(t, fixture.generatedFiles+1, copied, "project=%s scenario=%s fixture file count", fixture.template, scenario)
	return dir
}

func assertUpgradeManifestHashes(t *testing.T, dir string, doc manifest.Document) {
	t.Helper()
	hashes, err := manifest.HashFiles(dir, doc.GeneratedFiles)
	require.NoError(t, err)
	assert.Equal(t, doc.GeneratedFileSHA256, hashes, "upgraded manifest must hash every generated file")
}
