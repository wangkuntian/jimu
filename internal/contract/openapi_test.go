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
		"/users/{id}/roles":       {"post"},
		"/roles":                  {"get", "post"},
		"/roles/{id}":             {"get", "put", "delete"},
		"/roles/{id}/permissions": {"get", "post"},
		"/roles/options":          {"get"},
		"/permissions":            {"get", "post"},
		"/permissions/{id}":       {"get", "put", "delete"},
		"/audits":                 {"get"},
		"/audits/{id}":            {"get"},
		"/tenants":                {"get", "post"},
		// 管理端运维端点：必须声明鉴权（未列入 public 集合，故下方的 security 断言覆盖它们）
		"/admin/error-codes":    {"get"},
		"/admin/ratelimit/auth": {"get"},
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
			if method == "get" && path == "/roles" {
				assertHasQueryParam(t, operation, "tenant_id")
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
	assertDefinitionPropertyType(t, spec, "application.TenantResponse", "plan_id", "string")

	plan := definitionProperties(t, spec, "application.PlanResponse")
	assertDefinitionPropertyType(t, spec, "application.PlanResponse", "id", "string")
	if _, ok := plan["id"]; !ok {
		t.Fatal("application.PlanResponse 缺少字段 id")
	}

	assertDefinitionPropertyType(t, spec, "application.AssignPlanRequest", "plan_id", "string")
	assertPathParamType(t, spec, "/tenant-plans/{id}", "put", "id", "string")
	assertPathParamType(t, spec, "/tenant-plans/{id}", "delete", "id", "string")
}

func assertDefinitionPropertyType(t *testing.T, spec map[string]any, definition, property, want string) {
	t.Helper()
	properties := definitionProperties(t, spec, definition)
	field, ok := properties[property].(map[string]any)
	if !ok {
		t.Fatalf("%s.%s 不是对象", definition, property)
	}
	if got := field["type"]; got != want {
		t.Fatalf("%s.%s type = %v, want %s", definition, property, got, want)
	}
}

// TestOpenAPIAuthEndpointsCarryTypedData 登录/刷新/注册/me/config 的成功响应必须是具体类型：
// 前端据此生成 TokenPair、注册响应、MeView 与 AuthConfigResponse 的字段类型，而不是 unknown。
func TestOpenAPIAuthEndpointsCarryTypedData(t *testing.T) {
	spec := readOpenAPI(t)

	assertRefResponse(t, spec, "/auth/login", "post", 200, "#/definitions/contract.TokenPair")
	assertRefResponse(t, spec, "/auth/refresh", "post", 200, "#/definitions/contract.TokenPair")
	assertRefResponse(t, spec, "/auth/register", "post", 200, "#/definitions/application.RegisterResponse")
	assertRefResponse(t, spec, "/auth/me", "get", 200, "#/definitions/application.MeView")
	assertRefResponse(t, spec, "/auth/config", "get", 200, "#/definitions/interfaces.AuthConfigResponse")

	me := definitionProperties(t, spec, "application.MeView")
	for _, field := range []string{"user", "roles", "permissions"} {
		if _, ok := me[field]; !ok {
			t.Fatalf("application.MeView 缺少字段 %s", field)
		}
	}

	authConfig := definitionProperties(t, spec, "interfaces.AuthConfigResponse")
	for _, field := range []string{"public_registration", "provisioning_enabled"} {
		if _, ok := authConfig[field]; !ok {
			t.Fatalf("interfaces.AuthConfigResponse 缺少字段 %s", field)
		}
	}

	register := definitionProperties(t, spec, "application.RegisterResponse")
	if _, ok := register["user"]; !ok {
		t.Fatal("application.RegisterResponse 缺少 user 字段")
	}
	if _, ok := register["tenant"]; !ok {
		t.Fatal("application.RegisterResponse 缺少 tenant 字段（开通式注册时返回）")
	}

	user := definitionProperties(t, spec, "application.RegisterUserView")
	for _, field := range []string{"id", "username", "email", "phone", "status", "tenant_id"} {
		if _, ok := user[field]; !ok {
			t.Fatalf("application.RegisterUserView 缺少字段 %s", field)
		}
	}
}

// TestOpenAPIUserManagementCarriesTypedData 钉住用户管理页面的契约：
// 列表/详情返回租户与角色摘要，创建请求携带 tenant_id/role_ids，分配角色使用 role_ids。
func TestOpenAPIUserManagementCarriesTypedData(t *testing.T) {
	spec := readOpenAPI(t)

	assertRefResponse(t, spec, "/users", "get", 200, "#/definitions/application.UserPageResponse")
	assertRefResponse(t, spec, "/users", "post", 201, "#/definitions/application.UserBodyResponse")
	assertRefResponse(t, spec, "/users/{id}", "get", 200, "#/definitions/application.UserBodyResponse")
	assertRefResponse(t, spec, "/roles", "get", 200, "#/definitions/application.RolePageResponse")
	assertRefResponse(t, spec, "/roles", "post", 201, "#/definitions/application.RoleBodyResponse")

	user := definitionProperties(t, spec, "application.UserResponse")
	for _, field := range []string{"id", "username", "email", "phone", "status", "tenant", "roles", "created_at"} {
		if _, ok := user[field]; !ok {
			t.Fatalf("application.UserResponse 缺少字段 %s", field)
		}
	}
	if got := propertyType(user, "id"); got != "string" {
		t.Fatalf("application.UserResponse.id type = %q, want string", got)
	}
	tenant := definitionProperties(t, spec, "application.TenantRef")
	if got := propertyType(tenant, "id"); got != "string" {
		t.Fatalf("application.TenantRef.id type = %q, want string", got)
	}
	role := definitionProperties(t, spec, "application.RoleRef")
	if got := propertyType(role, "id"); got != "string" {
		t.Fatalf("application.RoleRef.id type = %q, want string", got)
	}

	create := definitionProperties(t, spec, "application.CreateUserRequest")
	for _, field := range []string{"tenant_id", "username", "password", "role_ids"} {
		if _, ok := create[field]; !ok {
			t.Fatalf("application.CreateUserRequest 缺少字段 %s", field)
		}
	}

	assign := definitionProperties(t, spec, "application.AssignUserRolesRequest")
	if _, ok := assign["role_ids"]; !ok {
		t.Fatal("application.AssignUserRolesRequest 缺少 role_ids")
	}

	roleResponse := definitionProperties(t, spec, "application.RoleResponse")
	if got := propertyType(roleResponse, "id"); got != "string" {
		t.Fatalf("application.RoleResponse.id type = %q, want string", got)
	}
	assertPathParamType(t, spec, "/users/{id}", "get", "id", "string")
	assertPathParamType(t, spec, "/users/{id}/roles", "post", "id", "string")
	assertPathParamType(t, spec, "/roles/{id}", "get", "id", "string")
	assertPathParamType(t, spec, "/tenants/{id}", "get", "id", "string")
}

func TestOpenAPIRoleManagementCarriesTypedData(t *testing.T) {
	spec := readOpenAPI(t)

	assertRefResponse(t, spec, "/permissions", "get", 200, "#/definitions/application.PermissionPageResponse")
	assertRefResponse(t, spec, "/roles/{id}/permissions", "get", 200, "#/definitions/application.PermissionListResponse")
	assertRefResponse(t, spec, "/roles/options", "get", 200, "#/definitions/application.RoleOptionListResponse")

	permissions := definitionProperties(t, spec, "application.PermissionPageResponse")
	assertItemsRef(t, permissions, "data", "#/definitions/application.PermissionResponse")
	permissionList := definitionProperties(t, spec, "application.PermissionListResponse")
	assertItemsRef(t, permissionList, "data", "#/definitions/application.PermissionResponse")

	permission := definitionProperties(t, spec, "application.PermissionResponse")
	for _, field := range []string{"id", "name", "resource", "action", "created_at"} {
		if _, ok := permission[field]; !ok {
			t.Fatalf("application.PermissionResponse 缺少字段 %s", field)
		}
	}
	if got := propertyType(permission, "id"); got != "string" {
		t.Fatalf("application.PermissionResponse.id type = %q, want string", got)
	}

	assign := definitionProperties(t, spec, "application.AssignPermissionsRequest")
	assertItemsType(t, assign, "permission_ids", "string")
	create := definitionProperties(t, spec, "application.CreateRoleRequest")
	if got := propertyType(create, "tenant_id"); got != "string" {
		t.Fatalf("application.CreateRoleRequest.tenant_id type = %q, want string", got)
	}
	update := definitionProperties(t, spec, "application.UpdateRoleRequest")
	if got := propertyType(update, "status"); got != "integer" {
		t.Fatalf("application.UpdateRoleRequest.status type = %q, want integer", got)
	}

	role := definitionProperties(t, spec, "application.RoleResponse")
	for _, field := range []string{
		"id", "tenant_id", "name", "description", "code", "scope", "is_system",
		"status", "created_at", "updated_at",
	} {
		if _, ok := role[field]; !ok {
			t.Fatalf("application.RoleResponse 缺少字段 %s", field)
		}
	}
	if got := propertyType(role, "id"); got != "string" {
		t.Fatalf("application.RoleResponse.id type = %q, want string", got)
	}
	if got := propertyType(role, "tenant_id"); got != "string" {
		t.Fatalf("application.RoleResponse.tenant_id type = %q, want string", got)
	}
	if got := propertyType(role, "is_system"); got != "boolean" {
		t.Fatalf("application.RoleResponse.is_system type = %q, want boolean", got)
	}

	optionList := definitionProperties(t, spec, "application.RoleOptionListResponse")
	assertItemsRef(t, optionList, "data", "#/definitions/application.RoleOptionResponse")
	option := definitionProperties(t, spec, "application.RoleOptionResponse")
	for _, field := range []string{"id", "code", "name", "scope", "is_system"} {
		if _, ok := option[field]; !ok {
			t.Fatalf("application.RoleOptionResponse 缺少字段 %s", field)
		}
	}
	if got := propertyType(option, "id"); got != "string" {
		t.Fatalf("application.RoleOptionResponse.id type = %q, want string", got)
	}

	assertPathParamType(t, spec, "/roles/{id}/permissions", "get", "id", "string")
	assertPathParamType(t, spec, "/roles/{id}/permissions", "post", "id", "string")
}

func TestOpenAPIAuditCarriesTenantScope(t *testing.T) {
	spec := readOpenAPI(t)
	assertRefResponse(t, spec, "/audits", "get", 200, "#/definitions/application.AuditLogPageResponse")
	assertRefResponse(t, spec, "/audits/{id}", "get", 200, "#/definitions/application.AuditLogBodyResponse")

	page := definitionProperties(t, spec, "application.AuditLogPageResponse")
	assertItemsRef(t, page, "data", "#/definitions/application.AuditLogResponse")
	body := definitionProperties(t, spec, "application.AuditLogBodyResponse")
	if _, ok := body["data"]; !ok {
		t.Fatal("application.AuditLogBodyResponse missing data")
	}
	audit := definitionProperties(t, spec, "application.AuditLogResponse")
	if got := propertyType(audit, "tenant_id"); got != "string" {
		t.Fatalf("application.AuditLogResponse.tenant_id type = %q, want string", got)
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

func propertyType(props map[string]any, field string) string {
	entry, _ := props[field].(map[string]any)
	got, _ := entry["type"].(string)
	return got
}

func assertPathParamType(t *testing.T, spec map[string]any, path, method, name, want string) {
	t.Helper()
	operation := operationOf(t, spec, path, method)
	params, _ := operation["parameters"].([]any)
	for _, param := range params {
		item, _ := param.(map[string]any)
		if item["in"] == "path" && item["name"] == name {
			if got, _ := item["type"].(string); got != want {
				t.Fatalf("%s %s path 参数 %s type = %q, want %q", method, path, name, got, want)
			}
			return
		}
	}
	t.Fatalf("%s %s 缺少 path 参数 %s", method, path, name)
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

func assertItemsType(t *testing.T, props map[string]any, field, wantType string) {
	t.Helper()
	entry, ok := props[field].(map[string]any)
	if !ok {
		t.Fatalf("缺少字段 %s", field)
	}
	items, _ := entry["items"].(map[string]any)
	if got, _ := items["type"].(string); got != wantType {
		t.Fatalf("字段 %s 的 items type = %q, want %q", field, got, wantType)
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
