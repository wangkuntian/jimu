package frameworkmanifest

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"jimu/tools/generator/manifest"
)

func TestExportMinimalIsDeterministic(t *testing.T) {
	root := repositoryRoot(t)
	req := Request{Root: root, Profile: "minimal", Module: "example.com/minimal"}
	first, err := Export(req)
	if err != nil {
		t.Fatal(err)
	}
	second, err := Export(req)
	if err != nil {
		t.Fatal(err)
	}
	one, err := manifest.CanonicalJSON(first)
	if err != nil {
		t.Fatal(err)
	}
	two, err := manifest.CanonicalJSON(second)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(one, two) {
		t.Fatal("repeated minimal exports are not byte-identical")
	}
	if first.Digest == "" || first.Digest != second.Digest {
		t.Fatalf("repeated exports have different digests: %q and %q", first.Digest, second.Digest)
	}
	assertNoAbsolutePath(t, one)
	if got := first.Selection.Profile; got != "minimal" {
		t.Fatalf("profile = %q, want minimal", got)
	}
	if len(first.Selection.Capabilities) == 0 {
		t.Fatal("minimal export has no selected capabilities")
	}
}

func TestExportWithUsesHardDependencyClosureAndDriverSelection(t *testing.T) {
	doc, err := Export(Request{Root: repositoryRoot(t), With: "user,access,queue", Shape: "app", Module: "example.com/with"})
	if err != nil {
		t.Fatal(err)
	}
	if doc.Selection.Profile != "" || doc.Selection.Shape != "app" {
		t.Fatalf("selection = %#v, want app shape without profile", doc.Selection)
	}
	for _, name := range []string{"user", "access", "queue"} {
		if !contains(doc.Selection.Capabilities, name) {
			t.Errorf("selection does not contain %q: %v", name, doc.Selection.Capabilities)
		}
	}
	foundTenant := false
	for _, capability := range doc.Capabilities {
		if capability.Name == "tenant" && capability.MigrationOnly {
			foundTenant = true
		}
	}
	if !foundTenant {
		t.Fatal("manifest does not record tenant as a migration-only capability")
	}
	if got := doc.Selection.Drivers["queue"]; len(got) != 1 || got[0] != "redis" {
		t.Fatalf("queue drivers = %v, want [redis]", got)
	}
	if len(doc.Copy) == 0 || len(doc.Templates) == 0 || len(doc.Merges) == 0 {
		t.Fatalf("manifest actions are incomplete: copy=%d templates=%d merges=%d", len(doc.Copy), len(doc.Templates), len(doc.Merges))
	}
}

func TestExportRejectsInvalidSelection(t *testing.T) {
	root := repositoryRoot(t)
	for name, req := range map[string]Request{
		"both profile and with":    {Root: root, Profile: "minimal", With: "user", Module: "example.com/x"},
		"neither profile nor with": {Root: root, Module: "example.com/x"},
		"invalid shape":            {Root: root, With: "user", Shape: "../bad", Module: "example.com/x"},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := Export(req); err == nil {
				t.Fatal("Export() succeeded, want validation error")
			}
		})
	}
}

func TestExportManifestValidates(t *testing.T) {
	doc, err := Export(Request{Root: repositoryRoot(t), Profile: "minimal", Module: "example.com/minimal"})
	if err != nil {
		t.Fatal(err)
	}
	if err := manifest.Validate(doc); err != nil {
		t.Fatalf("exported manifest is invalid: %v", err)
	}
	if got, err := manifest.Digest(doc); err != nil || got != doc.Digest {
		t.Fatalf("exported digest mismatch: got=%q err=%v want=%q", got, err, doc.Digest)
	}
}

func TestExportMatchesGoldens(t *testing.T) {
	root := repositoryRoot(t)
	cases := map[string]Request{
		"minimal":                {Root: root, Profile: "minimal", Module: "example.com/minimal"},
		"with_user_access_queue": {Root: root, With: "user,access,queue", Shape: "app", Module: "example.com/with"},
	}
	for name, req := range cases {
		t.Run(name, func(t *testing.T) {
			doc, err := Export(req)
			if err != nil {
				t.Fatal(err)
			}
			golden := goldenDocument(doc)
			want, err := json.MarshalIndent(golden, "", "  ")
			if err != nil {
				t.Fatal(err)
			}
			want = append(want, '\n')
			path := goldenPath(name)
			if os.Getenv("UPDATE_GOLDENS") == "1" {
				if err := os.WriteFile(path, want, 0o644); err != nil {
					t.Fatal(err)
				}
				return
			}
			got, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(got, want) {
				t.Fatalf("golden mismatch for %s", name)
			}
		})
	}
}

func goldenPath(name string) string {
	_, source, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(source), "testdata", name+".json")
}

func goldenDocument(doc manifest.Document) manifest.Document {
	doc.Framework.Commit = "<commit>"
	doc.Digest = ""
	digest, err := manifest.Digest(doc)
	if err != nil {
		panic(err)
	}
	doc.Digest = digest
	return doc
}

func repositoryRoot(t *testing.T) string {
	t.Helper()
	root, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root, err = filepath.Abs(filepath.Join(root, "../../.."))
	if err != nil {
		t.Fatal(err)
	}
	return root
}

func assertNoAbsolutePath(t *testing.T, data []byte) {
	t.Helper()
	text := string(data)
	if strings.Contains(text, repositoryMarker(t)) {
		t.Fatalf("manifest contains repository absolute path %q", repositoryMarker(t))
	}
	_ = text
}

func repositoryMarker(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs(filepath.Join(mustGetwd(t), "../../.."))
	if err != nil {
		t.Fatal(err)
	}
	return root
}

func mustGetwd(t *testing.T) string {
	t.Helper()
	root, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	return root
}
