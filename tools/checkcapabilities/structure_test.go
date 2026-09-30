package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const validImports = `package sample
import (
    "jimu/internal/assembly"
    "jimu/internal/contract"
)
`

const validCapability = validImports + `
var Descriptor = contract.Descriptor{Name: "sample"}
func Wire(*assembly.Context) (contract.Module, error) { return nil, nil }
`

func TestCapabilityStructure(t *testing.T) {
	tests := []struct {
		name  string
		files map[string]string
		want  string
	}{
		{"valid without layers", map[string]string{"capability.go": validCapability}, ""},
		{"missing descriptor", map[string]string{"wire.go": validImports + `
func Wire(*assembly.Context) (contract.Module, error) { return nil, nil }
`}, "Descriptor"},
		{"missing wire", map[string]string{"capability.go": validImports + `
var Descriptor = contract.Descriptor{Name: "sample"}
`}, "Wire"},
		{"duplicate descriptor", map[string]string{"a.go": validCapability, "b.go": validImports + `
var Descriptor = contract.Descriptor{}
`}, "Descriptor"},
		{"wrong signature", map[string]string{"capability.go": validImports + `
var Descriptor = contract.Descriptor{}
func Wire() error { return nil }
`}, "Wire"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module jimu\n"), 0o644); err != nil {
				t.Fatal(err)
			}
			dir := filepath.Join(root, "internal", "capabilities", "sample")
			if err := os.MkdirAll(dir, 0o755); err != nil {
				t.Fatal(err)
			}
			for name, content := range tt.files {
				if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			err := checkCapabilityStructure(root, []string{"sample"})
			if tt.want == "" {
				if err != nil {
					t.Fatal(err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tt.want) || !strings.Contains(err.Error(), "sample") {
				t.Fatalf("error = %v, want capability and %q", err, tt.want)
			}
		})
	}
}
