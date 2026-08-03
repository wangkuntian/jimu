# Jimu 统一 CRUD 与模块生成器实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**目标：** 统一 User、Role、Permission、Audit 的 API 契约，生成无需手工修复的完整 CRUD 模块，并让 OpenAPI 覆盖全部现有路由。

**架构：** 保持现有 Clean Architecture 分层：Handler 只处理 HTTP 输入输出，Application Service 负责用例与稳定错误，Repository 负责 Gorm 查询和持久化错误分类。生成器复用同一套 DTO、分页、状态码和错误约定，先在内存渲染并格式化全部文件，再安全写入磁盘。

**技术栈：** Go 1.26.5、Gin、Gorm、go-sql-driver/mysql、Goose、swaggo/swag、标准库 `go/format` 与 `testing`。

## 全局约束

- 不增加新的运行时服务或外部基础设施。
- 不改变认证、授权、refresh session、限流、生命周期和部署拓扑。
- User、Role、Permission 提供完整 CRUD；Audit 只提供 List/Get。
- Create 返回 `201`，Delete 返回空响应体 `204`，其余成功响应返回 `200`。
- 所有列表统一使用 `page`、`page_size`、`sort`、`order` 和共享分页响应。
- Domain 实体不得直接作为 API DTO；User API 不得暴露密码哈希或软删除字段。
- 参数错误为 `400`，资源不存在为 `404`，唯一键冲突为 `409`，未知内部错误返回脱敏 `500`。
- 生成模块默认字段为 `name`、`description`、时间和软删除字段，生成结果不得包含未实现代码。
- 所有行为修改必须先看到目标测试按预期失败，再写最小实现使其通过。

---

## 文件结构与职责

### 现有模块契约

- `internal/shared/errors/errors.go`：增加通用冲突错误码。
- `internal/shared/response/response.go`：增加 `Created` 和 `NoContent` helper。
- `internal/modules/user/application/dto.go`：User 请求/响应 DTO 与 entity 转换。
- `internal/modules/role/application/dto.go`：Role 请求/响应 DTO 与权限分配 DTO。
- `internal/modules/permission/application/dto.go`：Permission 请求/响应 DTO。
- `internal/modules/audit/application/dto.go`：Audit 只读响应 DTO。
- 四个模块的 `domain`、`application/service.go`、`infrastructure/mysql_repository.go`：统一 Repository 和 Service 契约。
- 四个模块的 `interfaces/handler.go`：统一参数校验、分页、状态码和响应 DTO。

### 模块生成器

- `tools/generator/module.go`：只保留生成流程、校验、渲染、格式化和安全写入。
- `tools/generator/templates.go`：保存完整模块源码模板。
- `tools/generator/module_test.go`：名称、冲突、migration、回滚和输出测试。
- `tools/generator/compile_test.go`：在临时仓库验证生成模块能够格式化、编译和测试。

### OpenAPI

- Auth、User、Role、Permission、Audit Handler：完整 Swagger 注解。
- `internal/contract/openapi.go`：仅用于文档的稳定响应 Schema。
- `docs/openapi/*`：由 `make swagger` 生成。
- `internal/contract/openapi_test.go`：读取生成 JSON，验证路径、方法、security 和分页参数。

---

### 任务 1：统一现有模块 CRUD 与 HTTP 契约

**文件：**

- 修改：`internal/shared/errors/errors.go`
- 修改：`internal/shared/response/response.go`
- 修改：`internal/shared/response/response_test.go`
- 修改：`internal/modules/user/domain/repository.go`
- 修改：`internal/modules/user/application/dto.go`
- 修改：`internal/modules/user/application/service.go`
- 修改：`internal/modules/user/infrastructure/mysql_repository.go`
- 修改：`internal/modules/user/interfaces/handler.go`
- 创建：`internal/modules/user/application/service_test.go`
- 扩展：`internal/modules/user/interfaces/handler_test.go`
- 修改：`internal/modules/role/domain/role.go`
- 修改：`internal/modules/role/domain/permission.go`
- 创建：`internal/modules/role/application/dto.go`
- 修改：`internal/modules/role/application/service.go`
- 修改：`internal/modules/role/infrastructure/mysql_repository.go`
- 修改：`internal/modules/role/interfaces/handler.go`
- 创建：`internal/modules/role/application/service_test.go`
- 创建：`internal/modules/role/interfaces/handler_test.go`
- 创建：`internal/modules/permission/application/dto.go`
- 修改：`internal/modules/permission/application/service.go`
- 修改：`internal/modules/permission/infrastructure/mysql_repository.go`
- 修改：`internal/modules/permission/interfaces/handler.go`
- 创建：`internal/modules/permission/application/service_test.go`
- 创建：`internal/modules/permission/interfaces/handler_test.go`
- 修改：`internal/modules/audit/domain/audit.go`
- 创建：`internal/modules/audit/application/dto.go`
- 修改：`internal/modules/audit/application/service.go`
- 修改：`internal/modules/audit/infrastructure/mysql_repository.go`
- 修改：`internal/modules/audit/interfaces/handler.go`
- 扩展：`internal/modules/audit/interfaces/handler_test.go`

**接口：**

- Repository List 统一接收 `offset, limit int, sort, order string`。
- Service List 统一接收 `pagination.Pagination` 并返回 DTO slice、total 和 error。
- `response.Created(c, data)` 输出 `201` 的统一 Body。
- `response.NoContent(c)` 输出 `204` 且 body 长度为 0。
- `errors.CodeConflict` 映射 HTTP `409`。

- [x] **步骤 1：先写共享响应 helper 的失败测试**

在 `internal/shared/response/response_test.go` 增加：

```go
func TestCreatedUsesStandardEnvelope(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("request_id", "rid-created")

	Created(c, gin.H{"id": uint64(7)})

	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusCreated)
	}
	var body Body
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Code != 0 || body.RequestID != "rid-created" {
		t.Fatalf("body = %+v", body)
	}
}

func TestNoContentWritesNoBody(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	NoContent(c)

	if w.Code != http.StatusNoContent || w.Body.Len() != 0 {
		t.Fatalf("status = %d body = %q", w.Code, w.Body.String())
	}
}
```

- [x] **步骤 2：运行测试并确认因 helper 不存在而失败**

运行：

```bash
GOCACHE=/private/tmp/jimu-go-build-cache go test ./internal/shared/response -run 'TestCreated|TestNoContent'
```

预期：编译失败，明确报告 `undefined: Created` 和 `undefined: NoContent`。

- [x] **步骤 3：实现共享状态码和响应 helper**

在 `internal/shared/errors/errors.go` 增加：

```go
CodeConflict = 1009
```

在 `internal/shared/response/response.go` 增加：

```go
func Created(c *gin.Context, data interface{}) {
	c.JSON(http.StatusCreated, Body{
		Code: 0, Message: "ok", Data: data, RequestID: requestID(c),
	})
}

func NoContent(c *gin.Context) {
	c.Status(http.StatusNoContent)
}
```

并在 `StatusForCode` 中将 `CodeConflict` 映射到 `http.StatusConflict`。

- [x] **步骤 4：运行共享响应测试并确认通过**

运行：

```bash
GOCACHE=/private/tmp/jimu-go-build-cache go test ./internal/shared/response ./internal/shared/errors
```

预期：两个 package 均 `ok`。

- [x] **步骤 5：为四个 Service 写失败测试并固定接口**

每个 Service 测试使用实现对应 Repository interface 的内存 fake，至少覆盖以下断言：

```go
func TestAuditServiceGetMapsNotFound(t *testing.T) {
	repo := &fakeAuditRepository{findErr: gorm.ErrRecordNotFound}
	service := NewAuditService(repo)

	_, err := service.Get(context.Background(), 9)
	var appErr *errors.AppError
	if !stderrors.As(err, &appErr) || appErr.Code != errors.CodeNotFound {
		t.Fatalf("err = %v", err)
	}
}

func TestUserResponseDoesNotContainPassword(t *testing.T) {
	got := ToUserResponse(domain.User{ID: 1, Username: "alice", Password: "hash", Status: 1})
	b, err := json.Marshal(got)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(b, []byte("hash")) || bytes.Contains(b, []byte("password")) {
		t.Fatalf("sensitive response: %s", b)
	}
}
```

Role 和 Permission 测试还必须覆盖：Create 重复名称返回 `CodeConflict`、List 传递已规范化的 offset/limit/sort/order、Update 缺失资源返回 `404`、Delete Repository 错误被保留为脱敏内部错误。

- [x] **步骤 6：运行 Service 测试并确认因新接口和 DTO 不存在而失败**

运行：

```bash
GOCACHE=/private/tmp/jimu-go-build-cache go test ./internal/modules/user/application ./internal/modules/role/application ./internal/modules/permission/application ./internal/modules/audit/application
```

预期：新增测试编译失败或断言失败，原因仅限缺少已设计的 DTO、Get/List 签名或错误映射。

- [x] **步骤 7：实现 DTO、Repository interface 和 Service**

DTO 转换保持显式，例如：

```go
type UserResponse struct {
	ID        uint64    `json:"id"`
	Username  string    `json:"username"`
	Status    int8      `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func ToUserResponse(user domain.User) UserResponse {
	return UserResponse{
		ID: user.ID, Username: user.Username, Status: user.Status,
		CreatedAt: user.CreatedAt, UpdatedAt: user.UpdatedAt,
	}
}
```

List 签名统一为：

```go
List(ctx context.Context, offset, limit int, sort, order string) ([]Entity, int64, error)
```

Service 使用 `p.GetOffset()`、`p.GetLimit()`、`p.Sort` 和 `p.Order` 调用 Repository，并将 entity slice 转为 DTO slice。`gorm.ErrRecordNotFound` 映射相应 not-found code，MySQL 1062 重复键映射 `CodeConflict` 或现有 `CodeUserExists`，其他错误使用 `errors.Wrap(CodeInternalError, ..., err)`。

- [x] **步骤 8：实现 Repository 的分页、排序和错误传播**

每个 List 使用 Handler 已校验的 sort/order：

```go
func (r *mysqlRepository) List(ctx context.Context, offset, limit int, sort, order string) ([]domain.Role, int64, error) {
	var items []domain.Role
	var total int64
	db := r.db.WithContext(ctx).Model(&domain.Role{})
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := db.Order(sort + " " + order).Offset(offset).Limit(limit).Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}
```

Audit 增加：

```go
FindByID(ctx context.Context, id uint64) (*AuditLog, error)
```

不得在 Repository 中拼接未经 Handler allow-list 校验的外部字段。

- [x] **步骤 9：为 Handler 写失败测试**

四个模块覆盖：Create `201`、Delete `204` 空 body、非法 ID `400`、List 默认分页和非法 sort、Get `404`、User body 不包含密码。典型断言：

```go
func TestDeleteReturnsEmptyNoContent(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "1"}}

	handler.Delete(c)

	if w.Code != http.StatusNoContent || w.Body.Len() != 0 {
		t.Fatalf("status = %d body = %q", w.Code, w.Body.String())
	}
}
```

- [x] **步骤 10：运行 Handler 测试并确认旧状态码、匿名 DTO 和未实现 Audit Get 导致失败**

运行：

```bash
GOCACHE=/private/tmp/jimu-go-build-cache go test ./internal/modules/user/interfaces ./internal/modules/role/interfaces ./internal/modules/permission/interfaces ./internal/modules/audit/interfaces
```

预期：新增断言失败；Audit Get 仍返回旧的 `not implemented` body。

- [x] **步骤 11：实现统一 Handler**

每个 Handler 必须：

```go
var p pagination.Pagination
if err := c.ShouldBindQuery(&p); err != nil {
	response.Fail(c, errors.New(errors.CodeInvalidParam, err.Error()))
	return
}
if err := p.Normalize("id", "name", "created_at"); err != nil {
	response.Fail(c, errors.New(errors.CodeInvalidParam, err.Error()))
	return
}
```

Create 调用 `response.Created`，Delete 调用 `response.NoContent`。Audit Get 解析 ID、调用 `service.Get` 并返回 Audit DTO。Role、Permission 移除 Handler 内匿名 struct，改用 Application DTO。

- [x] **步骤 12：验证整个 CRUD 契约并提交**

运行：

```bash
gofmt -w internal/shared internal/modules/user internal/modules/role internal/modules/permission internal/modules/audit
GOCACHE=/private/tmp/jimu-go-build-cache go test ./internal/shared/... ./internal/modules/user/... ./internal/modules/role/... ./internal/modules/permission/... ./internal/modules/audit/...
GOCACHE=/private/tmp/jimu-go-build-cache go test ./internal/modules/auth/...
git diff --check
```

预期：全部通过，Auth 回归测试无变化。

提交：

```bash
git add internal/shared internal/modules/user internal/modules/role internal/modules/permission internal/modules/audit
git commit -m "feat: unify module crud contracts"
```

---

### 任务 2：重写完整且安全的模块生成器

**文件：**

- 修改：`tools/generator/module.go`
- 创建：`tools/generator/templates.go`
- 创建：`tools/generator/module_test.go`
- 创建：`tools/generator/compile_test.go`
- 修改：`cmd/cli/main.go`（仅在需要传递明确 cwd 时调整调用）

**接口：**

- 对外保留 `GenerateModule(name string) error`。
- 新增 `GenerateModuleAt(root, name string) error` 供 CLI 内部和测试使用。
- 模板数据包含 `Name`、`NameCamel`、`TableName`、`RouteName` 和 `MigrationNumber`。

- [x] **步骤 1：写名称和冲突保护失败测试**

在 `module_test.go` 增加表驱动测试：

```go
func TestGenerateModuleRejectsInvalidNames(t *testing.T) {
	for _, name := range []string{"", "Product", "order-item", "order__item", "order_", "type"} {
		t.Run(name, func(t *testing.T) {
			root := newTestRepository(t)
			if err := GenerateModuleAt(root, name); err == nil {
				t.Fatal("expected validation error")
			}
			assertNoGeneratedFiles(t, root)
		})
	}
}

func TestGenerateModuleDoesNotOverwriteExistingTarget(t *testing.T) {
	root := newTestRepository(t)
	target := filepath.Join(root, "internal/modules/product/domain/entity.go")
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil { t.Fatal(err) }
	if err := os.WriteFile(target, []byte("keep"), 0o644); err != nil { t.Fatal(err) }

	if err := GenerateModuleAt(root, "product"); err == nil {
		t.Fatal("expected target conflict")
	}
	got, err := os.ReadFile(target)
	if err != nil || string(got) != "keep" {
		t.Fatalf("existing target changed: %q, %v", got, err)
	}
}
```

- [x] **步骤 2：运行测试并确认新入口不存在**

运行：

```bash
GOCACHE=/private/tmp/jimu-go-build-cache go test ./tools/generator -run 'TestGenerateModuleRejects|TestGenerateModuleDoesNotOverwrite'
```

预期：编译失败，报告 `undefined: GenerateModuleAt`。

- [x] **步骤 3：实现预检和模板数据**

实现：

```go
func GenerateModule(name string) error {
	root, err := os.Getwd()
	if err != nil { return fmt.Errorf("get working directory: %w", err) }
	return GenerateModuleAt(root, name)
}

func GenerateModuleAt(root, name string) error {
	data, targets, err := preflight(root, name)
	if err != nil { return err }
	files, err := renderAll(data, targets)
	if err != nil { return err }
	return writeAll(root, files)
}
```

名称校验使用正则、Go keyword set 和 snake_case 转换；仓库根目录必须包含 `go.mod`、`internal/modules` 和 `migrations`。预检枚举全部目标路径，任何一个存在立即失败。

- [x] **步骤 4：写完整输出与 migration 失败测试**

测试 `order_item` 生成 `OrderItem`，扫描 `001`、`006` 后生成 `007_create_order_items.sql`，并断言以下文件全部存在：Module、Entity、Repository、DTO、Service、MySQL Repository、Handler、Router、Service Test、Handler Test、migration。

```go
for _, rel := range requiredFiles("order_item", "007") {
	if _, err := os.Stat(filepath.Join(root, rel)); err != nil {
		t.Errorf("missing %s: %v", rel, err)
	}
}
```

遍历生成文件，断言不包含未实现标记和 `gin.Error{}`，并使用 `format.Source` 验证所有 `.go` 文件已格式化。

- [x] **步骤 5：运行完整输出测试并确认旧模板不满足契约**

运行：

```bash
GOCACHE=/private/tmp/jimu-go-build-cache go test ./tools/generator -run 'TestGenerateModuleCreatesCompleteCRUD'
```

预期：因缺少 migration、测试文件、默认字段和完整实现而失败。

- [x] **步骤 6：实现完整模板**

`templates.go` 中的生成模块必须遵循任务 1 的最终签名。默认 DTO：

```go
type CreateProductRequest struct {
	Name        string `json:"name" binding:"required,max=128"`
	Description string `json:"description" binding:"max=255"`
}

type UpdateProductRequest struct {
	Name        string `json:"name" binding:"required,max=128"`
	Description string `json:"description" binding:"max=255"`
}

type ProductResponse struct {
	ID          uint64    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
```

migration 必须创建唯一 `name`、`description`、时间和 `deleted_at` 索引，并提供对应 Down。生成测试使用内存 fake，不依赖真实 MariaDB 或 Redis。

- [x] **步骤 7：写写入失败回滚测试**

通过包内可替换的窄文件写入函数制造第 N 次写入失败，断言本次生成的文件和空目录全部清理，预先存在的 `migrations/001_base.sql` 保持不变。测试结束后恢复默认 writer，避免污染其他测试。

- [x] **步骤 8：运行回滚测试并确认失败**

运行：

```bash
GOCACHE=/private/tmp/jimu-go-build-cache go test ./tools/generator -run 'TestGenerateModuleRollsBackWriteFailure'
```

预期：在实现回滚前能够观察到残留文件，测试失败。

- [x] **步骤 9：实现原子式渲染和失败清理**

`renderAll` 在内存中完成所有 `template.Execute` 和 `format.Source`。`writeAll` 记录本次创建的文件和目录，任何写入失败时按反向顺序删除这些路径；不得删除预检前已经存在的目录或文件。

- [x] **步骤 10：验证生成模块编译并提交**

`compile_test.go` 在临时仓库写入最小 `go.mod`，复制生成模块依赖的 `internal/contract`、`internal/shared` 接口 stub，运行：

```go
cmd := exec.Command("go", "test", "./internal/modules/product/...")
cmd.Dir = root
cmd.Env = append(os.Environ(), "GOWORK=off", "GOCACHE=/private/tmp/jimu-go-build-cache")
output, err := cmd.CombinedOutput()
if err != nil {
	t.Fatalf("generated module does not compile: %v\n%s", err, output)
}
```

执行：

```bash
gofmt -w tools/generator cmd/cli/main.go
GOCACHE=/private/tmp/jimu-go-build-cache go test ./tools/generator ./cmd/cli
git diff --check
```

预期：生成器测试全部通过，`go test` 能编译新生成模块。

提交：

```bash
git add tools/generator cmd/cli/main.go
git commit -m "feat: generate complete crud modules"
```

---

### 任务 3：补齐 OpenAPI 契约并完成全量验证

**文件：**

- 修改：`internal/modules/auth/interfaces/handler.go`
- 修改：`internal/modules/user/interfaces/handler.go`
- 修改：`internal/modules/role/interfaces/handler.go`
- 修改：`internal/modules/permission/interfaces/handler.go`
- 修改：`internal/modules/audit/interfaces/handler.go`
- 创建：`internal/contract/openapi.go`
- 创建：`internal/contract/openapi_test.go`
- 生成：`docs/openapi/docs.go`
- 生成：`docs/openapi/swagger.json`
- 生成：`docs/openapi/swagger.yaml`
- 修改：`README.md`

**接口：**

- OpenAPI 使用具名成功、分页和错误 Schema。
- 所有受保护路由声明 `@Security BearerAuth`。
- List 路由声明四个统一查询参数。

- [ ] **步骤 1：写 OpenAPI 路径失败测试**

测试读取 `docs/openapi/swagger.json`，解码为结构化 map，并断言下列 path/method 全部存在：

```go
required := map[string][]string{
	"/auth/login": {"post"},
	"/auth/register": {"post"},
	"/auth/refresh": {"post"},
	"/auth/logout": {"post"},
	"/auth/logout-all": {"post"},
	"/users": {"get", "post"},
	"/users/{id}": {"get", "put", "delete"},
	"/roles": {"get", "post"},
	"/roles/{id}": {"get", "put", "delete"},
	"/roles/{id}/permissions": {"post"},
	"/permissions": {"get", "post"},
	"/permissions/{id}": {"get", "put", "delete"},
	"/audits": {"get"},
	"/audits/{id}": {"get"},
}
```

List operation 必须包含 `page`、`page_size`、`sort`、`order`；除公开 Auth 路由外的操作必须含 BearerAuth security。

- [ ] **步骤 2：运行测试并确认当前 OpenAPI 路径不完整**

运行：

```bash
GOCACHE=/private/tmp/jimu-go-build-cache go test ./internal/contract -run TestOpenAPI
```

预期：失败并报告缺少 `/users/{id}`、Role、Permission、Audit 等路径。

- [ ] **步骤 3：增加稳定文档 Schema 和 Swagger 注解**

`internal/contract/openapi.go` 定义仅用于文档的结构：

```go
type ErrorResponse struct {
	Code      int    `json:"code"`
	Message   string `json:"message"`
	RequestID string `json:"request_id,omitempty"`
}

type PageResponse struct {
	Code      int         `json:"code"`
	Message   string      `json:"message"`
	Data      interface{} `json:"data"`
	Total     int64       `json:"total"`
	Page      int         `json:"page"`
	PageSize  int         `json:"page_size"`
	RequestID string      `json:"request_id,omitempty"`
}
```

每个 Handler 注解明确 `@Param id path int true`、List 的四个 query 参数、请求 DTO、成功 status、`400/401/403/404/409/500` 错误和 BearerAuth。删除接口使用 `@Success 204`。

- [ ] **步骤 4：生成 OpenAPI 并运行契约测试**

运行：

```bash
make swagger
GOCACHE=/private/tmp/jimu-go-build-cache go test ./internal/contract -run TestOpenAPI
```

预期：OpenAPI 测试通过，JSON 中包含全部路径与方法。

- [ ] **步骤 5：验证 Swagger 生成稳定性**

运行：

```bash
make swagger
git diff --exit-code docs/openapi
```

预期：第二次生成无新增 diff。

- [ ] **步骤 6：更新 README 的 API 状态与生成器说明**

README 明确 Create `201`、Delete `204`、统一列表参数、Audit Get，以及生成器默认生成完整 CRUD、migration 和测试。删除与旧 `200` 或不完整骨架冲突的描述。

- [ ] **步骤 7：运行完整 Go 验证**

运行：

```bash
GOCACHE=/private/tmp/jimu-go-build-cache go test ./...
GOCACHE=/private/tmp/jimu-go-build-cache go test -race ./internal/shared/... ./internal/modules/user/... ./internal/modules/role/... ./internal/modules/permission/... ./internal/modules/audit/... ./tools/generator
GOCACHE=/private/tmp/jimu-go-build-cache go vet ./...
GOCACHE=/private/tmp/jimu-go-build-cache go build ./cmd/server ./cmd/cli
bash -n work/test_runtime_security.sh
bash -n work/smoke_api_contract.sh
docker compose config --quiet
git diff --check
```

预期：全部命令退出码为 0。

- [ ] **步骤 8：构建最终 Docker 镜像**

运行：

```bash
docker build -t jimu:unified-crud-test .
```

预期：镜像构建完成，Server 和 CLI 均在 builder stage 编译成功。

- [ ] **步骤 9：提交 OpenAPI 和文档**

```bash
git add internal/modules/auth/interfaces/handler.go \
  internal/modules/user/interfaces/handler.go \
  internal/modules/role/interfaces/handler.go \
  internal/modules/permission/interfaces/handler.go \
  internal/modules/audit/interfaces/handler.go \
  internal/contract/openapi.go internal/contract/openapi_test.go \
  docs/openapi README.md
git commit -m "docs: complete openapi crud contract"
```

- [ ] **步骤 10：检查最终提交边界和工作区状态**

运行：

```bash
git log -4 --oneline
git status --short
```

预期：设计、CRUD、Generator、OpenAPI 四个提交边界清晰，`git status --short` 无输出。
