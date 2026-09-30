package workspace

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"jimu/tools/generator/manifest"
)

func TestCreateWritesManifestAndRejectsNonEmptyTarget(t *testing.T) {
	source := t.TempDir()
	writeFile(t, source, "input.txt", "generated\n")
	writeFile(t, source, "config.yaml", "base:\n  enabled: true\nnew:\n  value: source\n")
	doc := fixtureDocument("generated.txt", "config.yaml")
	target := filepath.Join(t.TempDir(), "project")
	result, err := Create(doc, CreateOptions{Target: target, SourceRoot: source, Module: "example.com/app", NoTidy: true, NoSelfCheck: true})
	if err != nil {
		t.Fatal(err)
	}
	if result.FileCount == 0 {
		t.Fatal("Create() returned no generated files")
	}
	if _, err := LoadProjectManifest(target); err != nil {
		t.Fatal(err)
	}
	writeFile(t, target, "user.txt", "keep\n")
	if _, err := Create(doc, CreateOptions{Target: target, SourceRoot: source, Module: "example.com/app", NoTidy: true, NoSelfCheck: true}); err == nil || !strings.Contains(err.Error(), "not empty") {
		t.Fatalf("Create() error = %v, want non-empty target rejection", err)
	}
}

func TestCreateForceReplacesExistingProject(t *testing.T) {
	source := t.TempDir()
	writeFile(t, source, "input.txt", "new\n")
	writeFile(t, source, "config.yaml", "base:\n  enabled: true\nnew:\n  value: source\n")
	doc := fixtureDocument("generated.txt", "config.yaml")
	target := filepath.Join(t.TempDir(), "project")
	if err := os.MkdirAll(target, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, target, "old.txt", "old\n")
	if _, err := Create(doc, CreateOptions{Target: target, SourceRoot: source, Module: "example.com/app", Force: true, NoTidy: true, NoSelfCheck: true}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(target, "old.txt")); !os.IsNotExist(err) {
		t.Fatalf("old file still exists, stat error = %v", err)
	}
}

func TestCreateRenderFailureLeavesExistingTarget(t *testing.T) {
	source := t.TempDir()
	writeFile(t, source, "bad.go", "package ???\n")
	writeFile(t, source, "config.yaml", "base:\n  enabled: true\n")
	doc := fixtureDocument("bad.go", "config.yaml")
	doc.Copy[0].Source = "bad.go"
	target := filepath.Join(t.TempDir(), "project")
	if err := os.MkdirAll(target, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, target, "user.txt", "keep\n")
	if _, err := Create(doc, CreateOptions{Target: target, SourceRoot: source, Module: "example.com/app", Force: true, NoTidy: true, NoSelfCheck: true}); err == nil {
		t.Fatal("Create() succeeded with invalid generated Go source")
	}
	if got := readFile(t, target, "user.txt"); got != "keep\n" {
		t.Fatalf("existing target changed after render failure: %q", got)
	}
}

func TestUpdatePreservesUserFilesEditedConfigAndIsIdempotent(t *testing.T) {
	source := t.TempDir()
	writeFile(t, source, "input.txt", "generated\n")
	writeFile(t, source, "new.txt", "new\n")
	writeFile(t, source, "config.yaml", "base:\n  enabled: true\nnew:\n  value: source\n")
	doc := fixtureDocument("generated.txt", "config.yaml")
	target := filepath.Join(t.TempDir(), "project")
	if _, err := Create(doc, CreateOptions{Target: target, SourceRoot: source, Module: "example.com/app", NoTidy: true, NoSelfCheck: true}); err != nil {
		t.Fatal(err)
	}
	writeFile(t, target, "user.txt", "keep\n")
	writeFile(t, target, "config.yaml", "base:\n  enabled: false\nuser:\n  value: custom\n")
	next := doc
	next.Copy = append(next.Copy, manifest.CopyAction{Source: "new.txt", Destination: "new.txt"})
	next.GeneratedFiles = append(next.GeneratedFiles, "new.txt")
	if _, err := Update(doc, next, UpdateOptions{ProjectRoot: target, SourceRoot: source}); err != nil {
		t.Fatal(err)
	}
	if got := readFile(t, target, "user.txt"); got != "keep\n" {
		t.Fatalf("user file = %q", got)
	}
	if got := readFile(t, target, "config.yaml"); got != "base:\n  enabled: false\nuser:\n  value: custom\nnew:\n  value: source\n" {
		t.Fatalf("merged config = %q", got)
	}
	first := readFile(t, target, manifestRel)
	result, err := Update(next, next, UpdateOptions{ProjectRoot: target, SourceRoot: source})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Changed) != 0 || len(result.Removed) != 0 {
		t.Fatalf("idempotent update changed=%v removed=%v", result.Changed, result.Removed)
	}
	if second := readFile(t, target, manifestRel); second != first {
		t.Fatal("idempotent update changed manifest bytes")
	}
}

func TestLoadProjectManifestRejectsMissingAndBadDigest(t *testing.T) {
	root := t.TempDir()
	if _, err := LoadProjectManifest(root); err == nil || !strings.Contains(err.Error(), manifestRel) {
		t.Fatalf("missing manifest error = %v", err)
	}
	doc := fixtureDocument("generated.txt", "config.yaml")
	if err := manifest.Write(filepath.Join(root, manifestRel), doc); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, manifestRel)
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	content = []byte(strings.Replace(string(content), "generated.txt", "changed.txt", 1))
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadProjectManifest(root); err == nil || !strings.Contains(err.Error(), "digest mismatch") {
		t.Fatalf("bad digest error = %v", err)
	}
}

func fixtureDocument(generated, config string) manifest.Document {
	return manifest.Document{
		SchemaVersion:  1,
		Framework:      manifest.Framework{Module: "jimu"},
		Selection:      manifest.Selection{Shape: "app"},
		Copy:           []manifest.CopyAction{{Source: "input.txt", Destination: generated}},
		Merges:         []manifest.MergeAction{{Source: config, Destination: config, Strategy: "sections"}},
		Rewrites:       []manifest.RewriteAction{{Kind: "module", From: "jimu", To: "example.com/app"}},
		GeneratedFiles: []string{generated, config},
	}
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

func readFile(t *testing.T, root, rel string) string {
	t.Helper()
	content, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
	if err != nil {
		t.Fatal(err)
	}
	return string(content)
}
