package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGeneratorBoundaryAllowsFrameworkManifest(t *testing.T) {
	root := generatorBoundaryFixture(t)
	writeGeneratorFixture(t, root, "tools/generator/frameworkmanifest/export.go", `package frameworkmanifest

import (
	_ "jimu/internal/capabilities/catalog"
	_ "jimu/internal/contract"
	_ "jimu/internal/profiles/registry"
)
`)
	if err := checkGeneratorBoundary(root); err != nil {
		t.Fatalf("frameworkmanifest imports should be allowed: %v", err)
	}
}

func TestGeneratorBoundaryRejectsCoreImports(t *testing.T) {
	root := generatorBoundaryFixture(t)
	writeGeneratorFixture(t, root, "tools/generator/manifest/model.go", `package manifest

import _ "jimu/internal/capabilities/catalog"
`)
	writeGeneratorFixture(t, root, "tools/generator/render/render_test.go", `package render

import _ "jimu/internal/contract"
`)
	writeGeneratorFixture(t, root, "tools/generator/plan/build.go", `package plan

import _ "jimu/internal/profiles/registry"
`)
	writeGeneratorFixture(t, root, "tools/generator/workspace/stage.go", `package workspace

import _ "jimu/internal/contract"
`)
	writeGeneratorFixture(t, root, "tools/generator/report/measure.go", `package report

import _ "jimu/internal/capabilities/catalog"
`)
	writeGeneratorFixture(t, root, "tools/generator/module/create.go", `package module

import _ "jimu/internal/profiles/full"
`)

	err := checkGeneratorBoundary(root)
	if err == nil {
		t.Fatal("core imports must fail the generator boundary")
	}
	message := err.Error()
	for _, want := range []string{
		"manifest/model.go imports jimu/internal/capabilities/catalog",
		"render/render_test.go imports jimu/internal/contract",
		"plan/build.go imports jimu/internal/profiles/registry",
		"workspace/stage.go imports jimu/internal/contract",
		"report/measure.go imports jimu/internal/capabilities/catalog",
		"module/create.go imports jimu/internal/profiles/full",
	} {
		if !strings.Contains(message, want) {
			t.Fatalf("error = %q, want %q", message, want)
		}
	}
}

func TestGeneratorBoundaryIgnoresFixtures(t *testing.T) {
	root := generatorBoundaryFixture(t)
	writeGeneratorFixture(t, root, "tools/generator/testdata/example.go", `package example

import _ "jimu/internal/profiles/full"
`)
	if err := checkGeneratorBoundary(root); err != nil {
		t.Fatalf("testdata fixtures should not be treated as generator code: %v", err)
	}
}

func TestGeneratorBoundaryAllowsGeneratedProject(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/generated\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := checkGeneratorBoundary(root); err != nil {
		t.Fatalf("generated projects without tools/generator should pass: %v", err)
	}
}

func generatorBoundaryFixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module jimu\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "tools", "generator"), 0o755); err != nil {
		t.Fatal(err)
	}
	return root
}

func writeGeneratorFixture(t *testing.T, root, rel, content string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
