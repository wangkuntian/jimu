package contract

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestOpenAPIIncludesCRUDContract(t *testing.T) {
	spec := readOpenAPI(t)
	paths := spec["paths"].(map[string]any)
	required := map[string][]string{
		"/auth/login":             {"post"},
		"/auth/register":          {"post"},
		"/auth/refresh":           {"post"},
		"/auth/logout":            {"post"},
		"/auth/logout-all":        {"post"},
		"/users":                  {"get", "post"},
		"/users/{id}":             {"get", "put", "delete"},
		"/roles":                  {"get", "post"},
		"/roles/{id}":             {"get", "put", "delete"},
		"/roles/{id}/permissions": {"post"},
		"/permissions":            {"get", "post"},
		"/permissions/{id}":       {"get", "put", "delete"},
		"/audits":                 {"get"},
		"/audits/{id}":            {"get"},
	}
	public := map[string]bool{
		"post /auth/login":    true,
		"post /auth/register": true,
		"post /auth/refresh":  true,
	}
	for path, methods := range required {
		item, ok := paths[path].(map[string]any)
		if !ok {
			t.Fatalf("missing path %s", path)
		}
		for _, method := range methods {
			operation, ok := item[method].(map[string]any)
			if !ok {
				t.Fatalf("missing %s %s", method, path)
			}
			if method == "get" && (path == "/users" || path == "/roles" || path == "/permissions" || path == "/audits") {
				assertQueryParams(t, operation, path)
			}
			if !public[method+" "+path] {
				if _, ok := operation["security"]; !ok {
					t.Fatalf("missing security for %s %s", method, path)
				}
			}
		}
	}
}

func readOpenAPI(t *testing.T) map[string]any {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("..", "..", "docs", "openapi", "swagger.json"))
	if err != nil {
		t.Fatal(err)
	}
	var spec map[string]any
	if err := json.Unmarshal(b, &spec); err != nil {
		t.Fatal(err)
	}
	return spec
}

func assertQueryParams(t *testing.T, operation map[string]any, path string) {
	t.Helper()
	params, _ := operation["parameters"].([]any)
	seen := make(map[string]bool)
	for _, param := range params {
		item := param.(map[string]any)
		if item["in"] == "query" {
			seen[item["name"].(string)] = true
		}
	}
	for _, name := range []string{"page", "page_size", "sort", "order"} {
		if !seen[name] {
			t.Fatalf("%s missing query param %s", path, name)
		}
	}
}
