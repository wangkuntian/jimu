package manifest

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func validDocument() Document {
	return Document{
		SchemaVersion: CurrentSchemaVersion,
		Framework:     Framework{Module: "jimu", Version: "v0.3.3", Commit: "744c7a3"},
		Selection:     Selection{Shape: "app", Profile: "minimal", Capabilities: []string{"tenant", "user"}},
		Copy:          []CopyAction{{Source: "internal/kernel", Destination: "internal/kernel", Include: []string{}, Exclude: []string{}}},
		Report:        ReportSpec{Name: "minimal", Capabilities: []string{"tenant", "user"}},
	}
}

func TestValidateRejectsInvalidDocuments(t *testing.T) {
	tests := []struct {
		name string
		edit func(*Document)
		want string
	}{
		{
			name: "missing schema version",
			edit: func(doc *Document) { doc.SchemaVersion = 0 },
			want: "schema_version",
		},
		{
			name: "unknown schema version",
			edit: func(doc *Document) { doc.SchemaVersion = CurrentSchemaVersion + 1 },
			want: "schema version",
		},
		{
			name: "missing framework module",
			edit: func(doc *Document) { doc.Framework.Module = "" },
			want: "framework.module",
		},
		{
			name: "absolute copy source",
			edit: func(doc *Document) { doc.Copy[0].Source = "/tmp/framework" },
			want: "absolute",
		},
		{
			name: "parent traversal in destination",
			edit: func(doc *Document) { doc.Copy[0].Destination = "internal/../outside" },
			want: "..",
		},
		{
			name: "duplicate destinations",
			edit: func(doc *Document) {
				doc.Copy = append(doc.Copy, CopyAction{Source: "internal/config", Destination: "internal/kernel"})
			},
			want: "duplicate destination",
		},
		{
			name: "invalid digest",
			edit: func(doc *Document) { doc.Digest = "sha256:not-a-digest" },
			want: "digest",
		},
		{
			name: "invalid protected middleware mode",
			edit: func(doc *Document) {
				doc.Capabilities = []Capability{{Name: "auth", ProtectedHTTPMiddleware: "sometimes"}}
			},
			want: "protected middleware",
		},
		{
			name: "duplicate migration schema dependency",
			edit: func(doc *Document) {
				doc.Capabilities = []Capability{{Name: "user", MigrationSchemaDeps: []string{"tenant", "tenant"}}}
			},
			want: "duplicate",
		},
		{
			name: "self migration schema dependency",
			edit: func(doc *Document) {
				doc.Capabilities = []Capability{{Name: "user", MigrationSchemaDeps: []string{"user"}}}
			},
			want: "itself",
		},
		{
			name: "duplicate runtime service",
			edit: func(doc *Document) {
				doc.Capabilities = []Capability{{Name: "auth", RuntimeServices: []string{"mysql", "mysql"}}}
			},
			want: "duplicate",
		},
		{
			name: "invalid support level",
			edit: func(doc *Document) {
				doc.Selection.Support = "preview"
			},
			want: "support",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := validDocument()
			tt.edit(&doc)
			err := Validate(doc)
			if err == nil || !strings.Contains(strings.ToLower(err.Error()), strings.ToLower(tt.want)) {
				t.Fatalf("Validate() error = %v, want substring %q", err, tt.want)
			}
		})
	}
}

func TestLoadRejectsUnknownJSONFields(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "manifest.json")
	content := `{"schema_version":2,"framework":{"module":"jimu","unknown":true}}`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(path); err == nil || !strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("Load() error = %v, want unknown field error", err)
	}
}

func TestCanonicalJSONIsStableForMapInsertionOrder(t *testing.T) {
	first := validDocument()
	first.Selection.Drivers = map[string][]string{}
	first.Selection.Drivers["queue"] = []string{"rabbitmq", "redis"}
	first.Selection.Drivers["user"] = []string{"builtin"}

	second := validDocument()
	second.Selection.Drivers = map[string][]string{}
	second.Selection.Drivers["user"] = []string{"builtin"}
	second.Selection.Drivers["queue"] = []string{"rabbitmq", "redis"}

	one, err := CanonicalJSON(first)
	if err != nil {
		t.Fatal(err)
	}
	two, err := CanonicalJSON(second)
	if err != nil {
		t.Fatal(err)
	}
	if string(one) != string(two) {
		t.Fatalf("canonical JSON differs for equivalent maps:\n%s\n%s", one, two)
	}
}

func TestDigestIgnoresExistingDigest(t *testing.T) {
	first := validDocument()
	digest, err := Digest(first)
	if err != nil {
		t.Fatal(err)
	}
	first.Digest = digest
	second := first
	second.Digest = "sha256:" + strings.Repeat("0", 64)
	other, err := Digest(second)
	if err != nil {
		t.Fatal(err)
	}
	if digest != other {
		t.Fatalf("Digest() changed when existing digest changed: %q != %q", digest, other)
	}
	if !strings.HasPrefix(digest, "sha256:") || len(digest) != len("sha256:")+64 {
		t.Fatalf("Digest() = %q, want sha256 plus 64 hex characters", digest)
	}
}

func TestWriteAndLoadRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".jimu", "manifest.json")
	doc := validDocument()
	if err := Write(path, doc); err != nil {
		t.Fatal(err)
	}
	got, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if got.Digest == "" {
		t.Fatal("Load() returned a manifest without digest")
	}
	if got.Digest != mustDigest(t, got) {
		t.Fatalf("loaded digest %q does not match document", got.Digest)
	}
}

func TestLoadRejectsMissingProvenanceFields(t *testing.T) {
	for _, tc := range []struct {
		name string
		path []string
	}{
		{"framework dirty", []string{"framework", "dirty"}},
		{"selection template version", []string{"selection", "template_version"}},
		{"generated file hashes", []string{"generated_file_sha256"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), ".jimu", "manifest.json")
			doc := validDocument()
			doc.GeneratedFiles = []string{"go.mod"}
			doc.GeneratedFileSHA256 = map[string]string{"go.mod": "sha256:" + strings.Repeat("a", 64)}
			if err := Write(path, doc); err != nil {
				t.Fatal(err)
			}
			content, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			var raw map[string]any
			if err := json.Unmarshal(content, &raw); err != nil {
				t.Fatal(err)
			}
			removeJSONPath(raw, tc.path)
			content, err = json.Marshal(raw)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, content, 0o644); err != nil {
				t.Fatal(err)
			}
			if _, err := Load(path); err == nil || !strings.Contains(err.Error(), tc.path[len(tc.path)-1]) {
				t.Fatalf("Load() error = %v, want missing %s", err, tc.path[len(tc.path)-1])
			}
		})
	}
}

func removeJSONPath(root map[string]any, path []string) {
	current := root
	for _, part := range path[:len(path)-1] {
		current = current[part].(map[string]any)
	}
	delete(current, path[len(path)-1])
}

func mustDigest(t *testing.T, doc Document) string {
	t.Helper()
	digest, err := Digest(doc)
	if err != nil {
		t.Fatal(err)
	}
	return digest
}
