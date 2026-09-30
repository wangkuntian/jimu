package projectmetrics

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMeasureUsesNeutralReportSpec(t *testing.T) {
	root := t.TempDir()
	writeMetricFixture(t, root, "go.mod", "module example.com/app\n\ngo 1.26\n")
	writeMetricFixture(t, root, "cmd/server/main.go", "package main\n\nimport _ \"example.com/app/internal/feature\"\n\nfunc main() {}\n")
	writeMetricFixture(t, root, "internal/feature/feature.go", "package feature\n\nconst Enabled = true\n")
	spec := ReportSpec{
		Name:         "app",
		Capabilities: []string{"feature"},
		Routes:       3,
		Migrations:   2,
		Tables:       4,
	}
	metrics, err := Measure(root, "example.com/app", spec, nil)
	if err != nil {
		t.Fatal(err)
	}
	if metrics.Profile != "app" || metrics.Routes != 3 || metrics.Migrations != 2 || metrics.Tables != 4 {
		t.Fatalf("static metrics = %#v", metrics)
	}
	if metrics.Files != 2 || metrics.Lines != 8 {
		t.Fatalf("closure metrics = files:%d lines:%d", metrics.Files, metrics.Lines)
	}
	if len(metrics.Capabilities) != 1 || metrics.Capabilities[0] != "feature" {
		t.Fatalf("capabilities = %v", metrics.Capabilities)
	}
}

func writeMetricFixture(t *testing.T, root, rel, content string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
