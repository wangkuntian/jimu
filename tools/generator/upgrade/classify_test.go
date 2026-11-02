package upgrade

import (
	"crypto/sha256"
	"encoding/hex"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"jimu/tools/generator/manifest"
)

func TestClassifyCoversEveryCategory(t *testing.T) {
	report := classifyFixture(t,
		map[string]string{
			"untouched.txt": "same",
			"modified.txt":  "old",
			"conflict.txt":  "old",
			"deleted.txt":   "same",
		},
		map[string]string{
			"untouched.txt": "same",
			"modified.txt":  "old",
			"conflict.txt":  "local edit",
			"deleted.txt":   "same",
		},
		map[string]string{
			"untouched.txt": "same",
			"modified.txt":  "new",
			"conflict.txt":  "target edit",
			"new.txt":       "added",
		},
	)

	want := map[string]struct {
		category Category
		action   Action
	}{
		"untouched.txt": {Untouched, ActionNone},
		"modified.txt":  {Modified, ActionInstall},
		"conflict.txt":  {Conflict, ActionKeep},
		"new.txt":       {New, ActionInstall},
		"deleted.txt":   {Deleted, ActionKeep},
	}
	if len(report.Entries) != len(want) {
		t.Fatalf("entries = %d, want %d", len(report.Entries), len(want))
	}
	for _, entry := range report.Entries {
		expected, ok := want[entry.Path]
		if !ok {
			t.Fatalf("unexpected entry %q", entry.Path)
		}
		if entry.Category != expected.category || entry.Action != expected.action {
			t.Fatalf("%s = %s/%s, want %s/%s", entry.Path, entry.Category, entry.Action, expected.category, expected.action)
		}
		if entry.Reason == "" {
			t.Fatalf("%s has no reason", entry.Path)
		}
	}
	if report.Count(Untouched) != 1 || report.Count(Modified) != 1 || report.Count(Conflict) != 1 ||
		report.Count(New) != 1 || report.Count(Deleted) != 1 {
		t.Fatalf("counts = %#v", report.Counts)
	}
	if !report.HasConflicts() {
		t.Fatal("report should have conflicts")
	}
	if report.Status() != StatusConflict {
		t.Fatalf("status = %s, want conflict", report.Status())
	}
}

func TestClassifyTreatsEveryLocalModificationAsConflict(t *testing.T) {
	tests := []struct {
		name   string
		local  map[string]string
		target map[string]string
		reason string
	}{
		{
			name:   "local edit with unchanged target",
			local:  map[string]string{"file.txt": "local"},
			target: map[string]string{"file.txt": "old"},
			reason: "target template is unchanged",
		},
		{
			name:   "local edit with changed target",
			local:  map[string]string{"file.txt": "local"},
			target: map[string]string{"file.txt": "target"},
			reason: "both changed",
		},
		{
			name:   "local deletion",
			local:  map[string]string{},
			target: map[string]string{"file.txt": "old"},
			reason: "missing locally",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			report := classifyFixture(t, map[string]string{"file.txt": "old"}, test.local, test.target)
			entry := onlyEntry(t, report)
			if entry.Category != Conflict || entry.Action != ActionKeep {
				t.Fatalf("entry = %s/%s, want conflict/keep", entry.Category, entry.Action)
			}
			if !strings.Contains(entry.Reason, test.reason) {
				t.Fatalf("reason = %q, want substring %q", entry.Reason, test.reason)
			}
		})
	}
}

func TestClassifyUntouched(t *testing.T) {
	report := classifyFixture(t,
		map[string]string{"file.txt": "same"},
		map[string]string{"file.txt": "same"},
		map[string]string{"file.txt": "same"},
	)
	entry := onlyEntry(t, report)
	if entry.Category != Untouched || entry.Action != ActionNone {
		t.Fatalf("entry = %s/%s, want untouched/none", entry.Category, entry.Action)
	}
	if report.Status() != StatusUpToDate {
		t.Fatalf("status = %s, want up-to-date", report.Status())
	}
}

func TestClassifyModifiedIsInstallable(t *testing.T) {
	report := classifyFixture(t,
		map[string]string{"file.txt": "old"},
		map[string]string{"file.txt": "old"},
		map[string]string{"file.txt": "new"},
	)
	entry := onlyEntry(t, report)
	if entry.Category != Modified || entry.Action != ActionInstall {
		t.Fatalf("entry = %s/%s, want modified/install", entry.Category, entry.Action)
	}
	if report.Status() != StatusUpgradeAvailable {
		t.Fatalf("status = %s, want upgrade-available", report.Status())
	}
}

func TestClassifyNewFileIsInstallable(t *testing.T) {
	report := classifyFixture(t,
		map[string]string{},
		map[string]string{},
		map[string]string{"file.txt": "new"},
	)
	entry := onlyEntry(t, report)
	if entry.Category != New || entry.Action != ActionInstall {
		t.Fatalf("entry = %s/%s, want new/install", entry.Category, entry.Action)
	}
}

func TestClassifyDeletedFileIsKept(t *testing.T) {
	report := classifyFixture(t,
		map[string]string{"file.txt": "old"},
		map[string]string{"file.txt": "old"},
		map[string]string{},
	)
	entry := onlyEntry(t, report)
	if entry.Category != Deleted || entry.Action != ActionKeep {
		t.Fatalf("entry = %s/%s, want deleted/keep", entry.Category, entry.Action)
	}
	if !strings.Contains(entry.Reason, "left in place") {
		t.Fatalf("reason = %q", entry.Reason)
	}
}

func TestClassifyNewFileCollisionIsConflict(t *testing.T) {
	report := classifyFixture(t,
		map[string]string{},
		map[string]string{"user.txt": "user content"},
		map[string]string{"user.txt": "template content"},
	)
	entry := onlyEntry(t, report)
	if entry.Category != Conflict || entry.Action != ActionKeep {
		t.Fatalf("entry = %s/%s, want conflict/keep", entry.Category, entry.Action)
	}
	if !strings.Contains(entry.Reason, "already exists locally") {
		t.Fatalf("reason = %q", entry.Reason)
	}
}

func TestClassifyDeletedFileWithLocalEditIsConflict(t *testing.T) {
	report := classifyFixture(t,
		map[string]string{"file.txt": "old"},
		map[string]string{"file.txt": "local"},
		map[string]string{},
	)
	entry := onlyEntry(t, report)
	if entry.Category != Conflict || entry.Action != ActionKeep {
		t.Fatalf("entry = %s/%s, want conflict/keep", entry.Category, entry.Action)
	}
	if !strings.Contains(entry.Reason, "local copy was modified") {
		t.Fatalf("reason = %q", entry.Reason)
	}
}

func TestClassifyMergedConfigPreservesLocalEdits(t *testing.T) {
	projectRoot := t.TempDir()
	targetRoot := t.TempDir()
	writeFile(t, projectRoot, "configs/app.yaml", "server:\n  port: 9000\n")
	writeFile(t, targetRoot, "configs/app.yaml", "server:\n  port: 9000\nnew:\n  enabled: true\n")
	doc := manifest.Document{
		GeneratedFiles:      []string{"configs/app.yaml"},
		GeneratedFileSHA256: map[string]string{"configs/app.yaml": hashContent("server:\n  port: 8080\n")},
	}
	report, err := Classify(Options{
		ProjectRoot: projectRoot,
		Previous:    doc,
		TargetRoot:  targetRoot,
		TargetFiles: []string{"configs/app.yaml"},
		Merged:      []string{"configs/app.yaml"},
	})
	if err != nil {
		t.Fatal(err)
	}
	entry := onlyEntry(t, report)
	if entry.Category != Modified || entry.Action != ActionInstall {
		t.Fatalf("entry = %s/%s, want modified/install", entry.Category, entry.Action)
	}
	if !strings.Contains(entry.Reason, "preserving the local file") {
		t.Fatalf("reason = %q", entry.Reason)
	}
	if report.HasConflicts() {
		t.Fatal("merged config must not be a conflict")
	}
}

func TestClassifyMergedConfigMissingLocallyIsConflict(t *testing.T) {
	projectRoot := t.TempDir()
	targetRoot := t.TempDir()
	writeFile(t, targetRoot, "configs/app.yaml", "server:\n  port: 8080\n")
	doc := manifest.Document{
		GeneratedFiles:      []string{"configs/app.yaml"},
		GeneratedFileSHA256: map[string]string{"configs/app.yaml": hashContent("server:\n  port: 8080\n")},
	}
	report, err := Classify(Options{
		ProjectRoot: projectRoot,
		Previous:    doc,
		TargetRoot:  targetRoot,
		TargetFiles: []string{"configs/app.yaml"},
		Merged:      []string{"configs/app.yaml"},
	})
	if err != nil {
		t.Fatal(err)
	}
	entry := onlyEntry(t, report)
	if entry.Category != Conflict || entry.Action != ActionKeep {
		t.Fatalf("entry = %s/%s, want conflict/keep", entry.Category, entry.Action)
	}
}

func TestClassifyIsStableAcrossInputOrder(t *testing.T) {
	baseline := map[string]string{
		"z.txt": "same",
		"a.txt": "old",
		"m.txt": "old",
	}
	local := map[string]string{
		"z.txt": "same",
		"a.txt": "local",
		"m.txt": "old",
	}
	target := map[string]string{
		"z.txt": "same",
		"a.txt": "target",
		"m.txt": "new",
		"b.txt": "added",
	}
	first := classifyFixture(t, baseline, local, target)
	second := classifyFixture(t, baseline, local, target)
	if !slices.Equal(entryKeys(first.Entries), entryKeys(second.Entries)) {
		t.Fatalf("entries are not stable:\n%v\n%v", entryKeys(first.Entries), entryKeys(second.Entries))
	}
	want := []string{"a.txt", "m.txt", "b.txt", "z.txt"}
	if got := entryKeys(first.Entries); !slices.Equal(got, want) {
		t.Fatalf("order = %v, want %v", got, want)
	}
}

func TestClassifyIgnoresTargetFileInputOrder(t *testing.T) {
	projectRoot := t.TempDir()
	targetRoot := t.TempDir()
	writeFile(t, targetRoot, "b.txt", "b")
	writeFile(t, targetRoot, "a.txt", "a")
	doc := manifest.Document{
		GeneratedFiles:      []string{},
		GeneratedFileSHA256: map[string]string{},
	}
	first, err := Classify(Options{
		ProjectRoot: projectRoot, Previous: doc, TargetRoot: targetRoot,
		TargetFiles: []string{"b.txt", "a.txt"},
	})
	if err != nil {
		t.Fatal(err)
	}
	second, err := Classify(Options{
		ProjectRoot: projectRoot, Previous: doc, TargetRoot: targetRoot,
		TargetFiles: []string{"a.txt", "b.txt"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(entryKeys(first.Entries), entryKeys(second.Entries)) {
		t.Fatalf("input order changed report: %v vs %v", entryKeys(first.Entries), entryKeys(second.Entries))
	}
}

func classifyFixture(t *testing.T, baseline, local, target map[string]string) Report {
	t.Helper()
	projectRoot := t.TempDir()
	targetRoot := t.TempDir()
	for name, content := range local {
		writeFile(t, projectRoot, name, content)
	}
	for name, content := range target {
		writeFile(t, targetRoot, name, content)
	}
	doc := manifest.Document{
		GeneratedFiles:      slices.Sorted(maps.Keys(baseline)),
		GeneratedFileSHA256: make(map[string]string, len(baseline)),
	}
	for name, content := range baseline {
		doc.GeneratedFileSHA256[name] = hashContent(content)
	}
	report, err := Classify(Options{
		ProjectRoot: projectRoot,
		Previous:    doc,
		TargetRoot:  targetRoot,
		TargetFiles: slices.Sorted(maps.Keys(target)),
	})
	if err != nil {
		t.Fatal(err)
	}
	return report
}

func onlyEntry(t *testing.T, report Report) Entry {
	t.Helper()
	if len(report.Entries) != 1 {
		t.Fatalf("entries = %d, want 1: %#v", len(report.Entries), report.Entries)
	}
	return report.Entries[0]
}

func writeFile(t *testing.T, root, rel, content string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func hashContent(content string) string {
	sum := sha256.Sum256([]byte(content))
	return "sha256:" + hex.EncodeToString(sum[:])
}

func entryKeys(entries []Entry) []string {
	out := make([]string, len(entries))
	for i, entry := range entries {
		out[i] = entry.Path
	}
	return out
}
