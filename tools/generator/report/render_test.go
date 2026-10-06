package report

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"jimu/tools/internal/projectmetrics"
)

func TestRenderGeneratedReportUsesManifestFacts(t *testing.T) {
	content := Render(projectmetrics.Metrics{
		Routes:       3,
		Migrations:   2,
		Tables:       4,
		Files:        5,
		Lines:        42,
		HeavyDeps:    []string{"excelize"},
		Capabilities: []string{"user"},
	}, Metadata{
		Module:         "example.com/app",
		Shape:          "app",
		Capabilities:   []string{"user", "access"},
		MigrationOnly:  []string{"tenant"},
		Drivers:        []string{"queue=redis"},
		Assets:         []string{"deploy"},
		GeneratedFiles: 10,
		AssetFiles:     2,
	}, 7)
	for _, want := range []string{"example.com/app", "user, access", "tenant", "queue=redis", "| 路由数 | 3 |", "| go.mod 直接依赖 | 7 |"} {
		if !strings.Contains(content, want) {
			t.Fatalf("report does not contain %q:\n%s", want, content)
		}
	}
	if strings.Contains(content, ".jimu-generated") {
		t.Fatal("report contains removed framework/marker implementation details")
	}
}

func TestTreeCountsExcludesManifestAndReport(t *testing.T) {
	root := t.TempDir()
	writeReportFixture(t, root, "main.go", "package main\n")
	writeReportFixture(t, root, ".jimu/manifest.json", "{}\n")
	writeReportFixture(t, root, "docs/profiles/generated-report.md", "old\n")
	writeReportFixture(t, root, "deploy/app.yaml", "app\n")
	generated, assets, err := TreeCounts(root, []string{"deploy"})
	if err != nil {
		t.Fatal(err)
	}
	if generated != 2 || assets != 1 {
		t.Fatalf("counts = generated:%d assets:%d", generated, assets)
	}
}

func writeReportFixture(t *testing.T, root, rel, content string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
