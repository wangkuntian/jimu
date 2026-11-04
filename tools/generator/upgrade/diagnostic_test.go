package upgrade

import (
	"slices"
	"strings"
	"testing"

	"jimu/tools/generator/manifest"
)

func TestClassifyContentDiffUsesLocalToTargetDirection(t *testing.T) {
	projectRoot := t.TempDir()
	targetRoot := t.TempDir()
	writeFile(t, projectRoot, "README.md", "local line\n")
	writeFile(t, targetRoot, "README.md", "target line\n")
	report, err := Classify(Options{
		ProjectRoot: projectRoot,
		Previous:    baselineDocument("README.md", "old line\n"),
		TargetRoot:  targetRoot,
		TargetFiles: []string{"README.md"},
		IncludeDiff: true,
	})
	if err != nil {
		t.Fatal(err)
	}

	entry := onlyEntry(t, report)
	if !entry.BaselineExists || !entry.LocalExists || !entry.TargetExists {
		t.Fatalf("side existence = baseline:%t local:%t target:%t", entry.BaselineExists, entry.LocalExists, entry.TargetExists)
	}
	if entry.BaselineContentAvailable {
		t.Fatal("manifest baseline stores hashes only; content must not be claimed available")
	}
	if !strings.Contains(entry.Diff, "--- local/README.md") ||
		!strings.Contains(entry.Diff, "+++ target/README.md") ||
		!strings.Contains(entry.Diff, "-local line") ||
		!strings.Contains(entry.Diff, "+target line") {
		t.Fatalf("unexpected diff:\n%s", entry.Diff)
	}
}

func TestClassifyContentDiffSkipsBinaryFiles(t *testing.T) {
	projectRoot := t.TempDir()
	targetRoot := t.TempDir()
	writeFile(t, projectRoot, "asset.bin", "local\x00payload")
	writeFile(t, targetRoot, "asset.bin", "target\x00payload")
	report, err := Classify(Options{
		ProjectRoot: projectRoot,
		Previous:    baselineDocument("asset.bin", "old\x00payload"),
		TargetRoot:  targetRoot,
		TargetFiles: []string{"asset.bin"},
		IncludeDiff: true,
	})
	if err != nil {
		t.Fatal(err)
	}

	entry := onlyEntry(t, report)
	if !entry.Binary {
		t.Fatal("NUL-containing file must be marked binary")
	}
	if entry.Diff != "" {
		t.Fatalf("binary file must not produce a text patch:\n%s", entry.Diff)
	}
	if entry.LocalSHA256 == "" || entry.TargetSHA256 == "" {
		t.Fatal("binary diff fallback must retain both hashes")
	}
}

func TestClassifyContentDiffOmitsUntouchedAndRecordsMissingSides(t *testing.T) {
	projectRoot := t.TempDir()
	targetRoot := t.TempDir()
	writeFile(t, projectRoot, "deleted.txt", "old\n")
	writeFile(t, targetRoot, "new.txt", "new\n")
	report, err := Classify(Options{
		ProjectRoot: projectRoot,
		Previous: manifest.Document{
			GeneratedFiles: []string{"deleted.txt"},
			GeneratedFileSHA256: map[string]string{
				"deleted.txt": hashContent("old\n"),
			},
		},
		TargetRoot:  targetRoot,
		TargetFiles: []string{"new.txt"},
		IncludeDiff: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := entryKeys(report.Entries); !slices.Equal(got, []string{"new.txt", "deleted.txt"}) {
		t.Fatalf("order = %v", got)
	}
	byPath := make(map[string]Entry, len(report.Entries))
	for _, entry := range report.Entries {
		byPath[entry.Path] = entry
	}
	deleted := byPath["deleted.txt"]
	if !deleted.LocalExists || deleted.TargetExists || !strings.Contains(deleted.Diff, "-old") {
		t.Fatalf("deleted diagnostics = %#v", deleted)
	}
	added := byPath["new.txt"]
	if added.LocalExists || !added.TargetExists || !strings.Contains(added.Diff, "+new") {
		t.Fatalf("new diagnostics = %#v", added)
	}
}

func baselineDocument(path, content string) manifest.Document {
	return manifest.Document{
		GeneratedFiles:      []string{path},
		GeneratedFileSHA256: map[string]string{path: hashContent(content)},
	}
}
