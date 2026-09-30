package render

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"jimu/tools/generator/frameworkmanifest"
	"jimu/tools/generator/manifest"
	"jimu/tools/generator/plan"
)

func TestExecuteManifestActions(t *testing.T) {
	source := t.TempDir()
	destination := t.TempDir()
	writeFixture(t, source, "src/input.txt", "source\n")
	writeFixture(t, source, "configs/app.yaml", "base:\n  enabled: true\nnew:\n  value: source\n")
	writeFixture(t, source, "template.txt", "module={{.module}}\n")
	writeFixture(t, source, "src/unsupported_test.go", "package src\nimport _ \"example.com/app/missing\"\n")

	doc := manifest.Document{
		SchemaVersion:  1,
		Framework:      manifest.Framework{Module: "jimu"},
		Selection:      manifest.Selection{Shape: "app", Capabilities: []string{"user"}, Drivers: map[string][]string{}},
		Copy:           []manifest.CopyAction{{Source: "src/input.txt", Destination: "copied/input.txt", Include: []string{}, Exclude: []string{}}},
		Templates:      []manifest.TemplateAction{{Source: "template.txt", Destination: "generated.txt", Kind: "text", Data: map[string]any{"module": "example.com/app"}}},
		Merges:         []manifest.MergeAction{{Source: "configs/app.yaml", Destination: "configs/app.yaml", Strategy: "sections", Sections: []string{}}},
		Rewrites:       []manifest.RewriteAction{{Kind: "module", From: "example.com/old", To: "example.com/app", Files: []string{"copied/input.txt"}}},
		Prune:          []manifest.PruneRule{{Kind: "test-import", Path: "**/*_test.go", Reason: "missing package"}},
		Report:         manifest.ReportSpec{Name: "app", Capabilities: []string{"user"}},
		GeneratedFiles: []string{"copied/input.txt", "generated.txt", "configs/app.yaml"},
	}
	executionPlan, err := plan.Build(doc)
	if err != nil {
		t.Fatal(err)
	}
	files, err := Execute(Context{SourceRoot: source, Destination: destination, Module: "example.com/app", Plan: executionPlan})
	if err != nil {
		t.Fatal(err)
	}
	if !contains(files, "copied/input.txt") || !contains(files, "generated.txt") {
		t.Fatalf("generated files = %v", files)
	}
	content, err := os.ReadFile(filepath.Join(destination, "generated.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "module=example.com/app\n" {
		t.Fatalf("template output = %q", content)
	}
	if _, err := os.Stat(filepath.Join(destination, "src/unsupported_test.go")); !os.IsNotExist(err) {
		t.Fatalf("unsupported test still exists, stat error = %v", err)
	}
}

func TestExecuteFrameworkManifestTemplates(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	doc, err := frameworkmanifest.Export(frameworkmanifest.Request{
		Root:    root,
		Profile: "minimal",
		Module:  "example.com/minimal",
	})
	if err != nil {
		t.Fatal(err)
	}
	executionPlan, err := plan.Build(doc)
	if err != nil {
		t.Fatal(err)
	}
	destination := t.TempDir()
	files, err := Execute(Context{SourceRoot: root, Destination: destination, Module: "example.com/minimal", Plan: executionPlan})
	if err != nil {
		t.Fatal(err)
	}
	if !contains(files, "internal/profiles/minimal/assembly.go") {
		t.Fatalf("generated files do not contain the selected assembly: %v", files)
	}
	goMod, err := os.ReadFile(filepath.Join(destination, "go.mod"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(goMod), "module example.com/minimal\n") {
		t.Fatalf("module directive = %q", goMod)
	}
	cli, err := os.ReadFile(filepath.Join(destination, "cmd/cli/main.go"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(cli), `"jimu/tools/generator"`) || strings.Contains(string(cli), "rootCmd.AddCommand(newCmd)") || strings.Contains(string(cli), "rootCmd.AddCommand(capabilityCmd)") {
		t.Fatalf("generated CLI still contains framework scaffold commands")
	}
	if err := filepath.WalkDir(destination, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || filepath.Ext(path) != ".go" {
			return nil
		}
		_, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
		return err
	}); err != nil {
		t.Fatalf("generated Go source is invalid: %v", err)
	}
}

func writeFixture(t *testing.T, root, rel, content string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
