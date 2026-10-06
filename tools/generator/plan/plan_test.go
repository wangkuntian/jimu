package plan

import (
	"strings"
	"testing"

	"jimu/tools/generator/manifest"
)

func validDocument() manifest.Document {
	return manifest.Document{
		SchemaVersion: 1,
		Framework:     manifest.Framework{Module: "jimu", Version: "v0.3.3"},
		Selection:     manifest.Selection{Shape: "app", Capabilities: []string{"user"}, Drivers: map[string][]string{}},
		Copy: []manifest.CopyAction{
			{Source: "z", Destination: "z", Include: []string{}, Exclude: []string{}},
			{Source: "a", Destination: "a", Include: []string{}, Exclude: []string{}},
		},
		Templates: []manifest.TemplateAction{{Source: "tpl", Destination: "generated.go", Kind: "go", Data: map[string]string{}}},
		Merges:    []manifest.MergeAction{{Source: "configs/app.yaml", Destination: "configs/app.yaml", Strategy: "sections", Sections: []string{}}},
		Rewrites:  []manifest.RewriteAction{{Kind: "module", From: "jimu", To: "example.com/app", Files: []string{}}},
		Assets:    []manifest.AssetAction{{Source: "deploy", Destination: "deploy", Include: []string{}, Exclude: []string{}}},
		Prune:     []manifest.PruneRule{{Kind: "test-import", Path: "**/*_test.go", Reason: "fixture"}},
		Report:    manifest.ReportSpec{Name: "app", Capabilities: []string{"user"}},
		GeneratedFiles: []string{
			"z.go", "a.go",
		},
	}
}

func TestBuildNormalizesActionOrderAndGeneratedFiles(t *testing.T) {
	doc := validDocument()
	got, err := Build(doc)
	if err != nil {
		t.Fatal(err)
	}
	if got.Copy[0].Destination != "a" || got.Copy[1].Destination != "z" {
		t.Fatalf("copy order = %#v, want a then z", got.Copy)
	}
	if len(got.GeneratedFiles) != 2 || got.GeneratedFiles[0] != "a.go" || got.GeneratedFiles[1] != "z.go" {
		t.Fatalf("generated files = %v, want sorted unique files", got.GeneratedFiles)
	}
	if got.Selection.Capabilities[0] != "user" || len(got.Copy) != 2 {
		t.Fatal("selection and copy set were not kept as separate fields")
	}
}

func TestBuildRejectsConflictingDestinations(t *testing.T) {
	doc := validDocument()
	doc.Templates = append(doc.Templates, manifest.TemplateAction{Source: "other", Destination: "generated.go", Kind: "go", Data: map[string]string{}})
	if _, err := Build(doc); err == nil || !strings.Contains(err.Error(), "duplicate destination") {
		t.Fatalf("Build() error = %v, want duplicate destination", err)
	}
}

func TestBuildAllowsExplicitConfigMergeDestinations(t *testing.T) {
	doc := validDocument()
	doc.Merges = append(doc.Merges, manifest.MergeAction{Source: "defaults.yaml", Destination: "configs/app.yaml", Strategy: "sections", Sections: []string{}})
	if _, err := Build(doc); err != nil {
		t.Fatalf("Build() rejected explicit merge destination: %v", err)
	}
}
