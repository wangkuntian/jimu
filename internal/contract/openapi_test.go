package contract

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestOpenAPIIncludesCRUDContract(t *testing.T) {
	spec := readOpenAPI(t)
	paths := spec["paths"].(map[string]any)
	required := map[string][]string{
		"/auth/login":             {"post"},
		"/auth/register":          {"post"},
		"/auth/config":            {"get"},
		"/auth/refresh":           {"post"},
		"/auth/logout":            {"post"},
		"/auth/logout-all":        {"post"},
		"/auth/me":                {"get"},
		"/users":                  {"get", "post"},
		"/users/{id}":             {"get", "put", "delete"},
		"/roles":                  {"get", "post"},
		"/roles/{id}":             {"get", "put", "delete"},
		"/roles/{id}/permissions": {"post"},
		"/permissions":            {"get", "post"},
		"/permissions/{id}":       {"get", "put", "delete"},
		"/audits":                 {"get"},
		"/audits/{id}":            {"get"},
		"/tenants":                {"get", "post"},
	}
	public := map[string]bool{
		"post /auth/login":    true,
		"post /auth/register": true,
		"get /auth/config":    true,
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
			if method == "get" && (path == "/users" || path == "/roles" || path == "/permissions" || path == "/audits" || path == "/tenants") {
				assertQueryParams(t, operation, path)
			}
			if method == "get" && path == "/tenants" {
				assertHasQueryParam(t, operation, "keyword")
				assertHasQueryParam(t, operation, "status")
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

// TestOpenAPIPathsCarryNoBasePathPrefix 钉住 swag 的 @Router 注解不得自带 basePath 前缀：
// 注解写 /api/v1/xxx 时，生成的路径键与 basePath 组合后会变成 /api/v1/api/v1/xxx，
// 客户端无法调用（v0.4.0 前 /admin/error-codes 与 /admin/ratelimit/auth 就是这样）。
func TestOpenAPIPathsCarryNoBasePathPrefix(t *testing.T) {
	spec := readOpenAPI(t)

	basePath, _ := spec["basePath"].(string)
	if basePath == "" {
		t.Fatal("swagger.json 缺少 basePath")
	}
	paths, _ := spec["paths"].(map[string]any)
	if len(paths) == 0 {
		t.Fatal("swagger.json 没有任何路径")
	}
	for path := range paths {
		if strings.HasPrefix(path, basePath+"/") {
			t.Errorf("路径 %q 自带 basePath 前缀 %q：@Router 注解不应写 /api/v1 前缀", path, basePath)
		}
	}
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

func assertHasQueryParam(t *testing.T, operation map[string]any, name string) {
	t.Helper()
	params, _ := operation["parameters"].([]any)
	for _, param := range params {
		item, _ := param.(map[string]any)
		if item["in"] == "query" && item["name"] == name {
			return
		}
	}
	t.Fatalf("缺少 query 参数 %s", name)
}

// TestOpenAPIListEndpointsCarryTypedData 钉住本轮前端消费的租户/套餐端点必须声明具体 DTO：
// data 为 []{} 或裸对象时前端只能手写视图模型，字段变化不会在编译期报错。
// 同时钉住本轮的改动边界：未纳入范围的端点继续声明 response.Body。
func TestOpenAPIListEndpointsCarryTypedData(t *testing.T) {
	spec := readOpenAPI(t)

	assertRefResponse(t, spec, "/tenants", "get", 200, "#/definitions/application.TenantPageResponse")
	assertRefResponse(t, spec, "/tenants", "post", 201, "#/definitions/application.TenantBodyResponse")
	assertRefResponse(t, spec, "/tenants/{id}", "get", 200, "#/definitions/application.TenantBodyResponse")
	assertRefResponse(t, spec, "/tenant-plans", "get", 200, "#/definitions/application.PlanPageResponse")

	assertRefResponse(t, spec, "/tenants/{id}", "put", 200, "#/definitions/response.Body")
	assertRefResponse(t, spec, "/tenants/{id}", "delete", 200, "#/definitions/response.Body")
	assertRefResponse(t, spec, "/tenants/usage", "get", 200, "#/definitions/response.Body")

	tenants := definitionProperties(t, spec, "application.TenantPageResponse")
	assertItemsRef(t, tenants, "data", "#/definitions/application.TenantResponse")

	tenant := definitionProperties(t, spec, "application.TenantResponse")
	for _, field := range []string{"id", "code", "name", "status", "plan_id", "created_at"} {
		if _, ok := tenant[field]; !ok {
			t.Fatalf("application.TenantResponse 缺少字段 %s", field)
		}
	}
}

func assertRefResponse(t *testing.T, spec map[string]any, path, method string, statusCode int, wantRef string) {
	t.Helper()
	operation := operationOf(t, spec, path, method)
	responses, _ := operation["responses"].(map[string]any)
	resp, _ := responses[strconv.Itoa(statusCode)].(map[string]any)
	schema, _ := resp["schema"].(map[string]any)
	if got, _ := schema["$ref"].(string); got != wantRef {
		t.Fatalf("%s %s 的 %d schema = %q, want %q", method, path, statusCode, got, wantRef)
	}
}

func definitionProperties(t *testing.T, spec map[string]any, name string) map[string]any {
	t.Helper()
	defs, _ := spec["definitions"].(map[string]any)
	def, ok := defs[name].(map[string]any)
	if !ok {
		t.Fatalf("缺少 definition %s", name)
	}
	props, _ := def["properties"].(map[string]any)
	return props
}

func assertItemsRef(t *testing.T, props map[string]any, field, wantRef string) {
	t.Helper()
	entry, ok := props[field].(map[string]any)
	if !ok {
		t.Fatalf("缺少字段 %s", field)
	}
	items, _ := entry["items"].(map[string]any)
	if got, _ := items["$ref"].(string); got != wantRef {
		t.Fatalf("字段 %s 的 items = %q, want %q", field, got, wantRef)
	}
}

func operationOf(t *testing.T, spec map[string]any, path, method string) map[string]any {
	t.Helper()
	paths, _ := spec["paths"].(map[string]any)
	item, ok := paths[path].(map[string]any)
	if !ok {
		t.Fatalf("缺少路径 %s", path)
	}
	operation, ok := item[method].(map[string]any)
	if !ok {
		t.Fatalf("缺少 %s %s", method, path)
	}
	return operation
}
