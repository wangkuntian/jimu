package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCapabilityImports(t *testing.T) {
	tests := []struct {
		name, source, path, want string
	}{
		{"same capability", "auth", "jimu/internal/capabilities/auth/domain", ""},
		{"own driver", "storage/local", "jimu/internal/capabilities/storage", ""},
		{"catalog composition root", "catalog", "jimu/internal/capabilities/user", ""},
		{"cross capability", "auth", "jimu/internal/capabilities/user", "internal/capabilities/user"},
		{"cross capability in test", "auth", "jimu/internal/capabilities/user", "internal/capabilities/user"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module jimu\n"), 0o644); err != nil {
				t.Fatal(err)
			}
			dir := filepath.Join(root, "internal", "capabilities", tt.source)
			if err := os.MkdirAll(dir, 0o755); err != nil {
				t.Fatal(err)
			}
			fileName := "source.go"
			if tt.name == "cross capability in test" {
				fileName = "source_test.go"
			}
			content := "package sample\nimport _ \"" + tt.path + "\"\n"
			if err := os.WriteFile(filepath.Join(dir, fileName), []byte(content), 0o644); err != nil {
				t.Fatal(err)
			}
			err := checkCapabilityImports(root)
			if tt.want == "" {
				if err != nil {
					t.Fatal(err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tt.want) || !strings.Contains(err.Error(), fileName) {
				t.Fatalf("error = %v, want file and %q", err, tt.want)
			}
		})
	}
}

func TestCapabilityImportsReportsInvalidSource(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/custom\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(root, "internal", "capabilities", "auth")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(dir, "broken.go")
	if err := os.WriteFile(file, []byte("package auth\nimport (\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := checkCapabilityImports(root); err == nil || !strings.Contains(err.Error(), file) {
		t.Fatalf("error = %v, want source file", err)
	}
}
