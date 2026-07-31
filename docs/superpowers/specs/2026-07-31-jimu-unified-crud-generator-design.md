# Jimu 统一 CRUD 与模块生成器设计

## 1. 目标

补齐现有 CRUD 能力，并让 Jimu 生成的模块无需手工修复即可使用。本次改动统一 User、Role、Permission 和 Audit API，生成可编译的完整 CRUD 模块，并扩展 OpenAPI 以覆盖所有现有公共路由。

本阶段不增加基础设施或依赖。Transactional Outbox、Scheduler、OpenTelemetry 以及更广泛的 MariaDB/Redis 集成测试留在后续独立阶段完成。

## 2. 范围

实现包含三个相互关联的部分：

1. 统一现有 User、Role、Permission 和 Audit 模块契约。
2. 使用完整 CRUD 模块生成器替换不完整的模块模板。
3. 为 Auth 和所有 CRUD 模块生成并验证完整的 OpenAPI 契约。

现有认证、授权、refresh session、限流、生命周期行为和部署拓扑保持不变。

## 3. API 契约

### 3.1 模块操作

- User、Role 和 Permission 提供 Create、Get、List、Update 和 Delete。
- Audit 保持只读，仅提供 List 和 Get。
- 所有 Handler 使用具名请求和响应 DTO，不再使用匿名请求结构体。
- Domain 实体不直接序列化为 API 响应。
- User 响应不得暴露密码哈希、软删除元数据或其他内部字段。

### 3.2 HTTP 行为

- Create 返回 HTTP `201 Created`，并使用统一响应结构。
- Get、List 和 Update 返回 HTTP `200 OK`。
- Delete 返回 HTTP `204 No Content`，响应体为空。
- 无效路径参数、JSON 请求体、分页、排序字段和排序方向返回 HTTP `400 Bad Request`。
- 资源不存在返回 HTTP `404 Not Found`。
- 唯一名称冲突返回 HTTP `409 Conflict`。
- 未知 Repository 和基础设施错误返回现有的脱敏 HTTP `500 Internal Server Error` 响应。

### 3.3 列表契约

User、Role、Permission 和 Audit 列表接口统一接受 `page`、`page_size`、`sort` 和 `order` 查询参数。每个模块声明允许排序的字段列表。列表响应使用现有统一分页结构，并执行共享的最大页大小限制。

## 4. Application 与持久化设计

每个模块遵循现有 Clean Architecture 边界：

- Handler 校验传输层输入，并通过共享响应格式输出 Application 结果。
- Service 负责用例行为，并将 Repository 结果映射为稳定的应用错误。
- Repository 负责 Gorm 查询、分页、排序和持久化错误。

User、Role 和 Permission Repository 统一提供 `FindByID`、`List`、`Create`、`Update` 和 `Delete`。Audit 在保留批量创建和列表能力的同时增加 `FindByID`。`Count` 和数据查询错误都必须返回，不得静默忽略。

实现将 `gorm.ErrRecordNotFound` 映射为对应的资源不存在错误。重复键错误映射为稳定的冲突错误，不暴露 SQL 或驱动细节。其他错误保留 cause 供内部日志记录，对外只返回脱敏响应。

## 5. 生成模块契约

`jimu module create <name>` 默认生成包含以下内容的完整 CRUD 模块：

- Entity 字段：`id`、`name`、`description`、`created_at`、`updated_at` 和软删除元数据。
- `name` 必填、唯一，最大长度为 128 个字符。
- `description` 可选，最大长度为 255 个字符。
- 具名 Create、Update 和 Response DTO。
- Repository 接口和 MySQL 实现。
- 完整实现 Create、Get、List、Update 和 Delete 的 Application Service。
- 使用统一 API 契约的 HTTP Handler 和 Router。
- 与现有 `contract.Module` 接口兼容的模块装配。
- 同时包含 Up 和 Down 的 Goose migration。
- 聚焦的 Service 和 Handler 测试。

生成的源码不得包含 TODO 标记、空 DTO、未实现 Handler、无效 import 或被忽略的参数解析错误。

### 5.1 命名与 migration 规则

模块名使用小写单数 ASCII 标识符。模块名必须匹配 `^[a-z][a-z0-9_]*$`，不得包含连续下划线或以下划线结尾，也不得是 Go 关键字。snake_case 名称通过将每个片段首字母大写转换为导出的 Go 名称。

路由和表名使用模块名加 `s`。生成器扫描现有数字编号的 Goose migration 并选择下一个三位序号，因此当前仓库会生成 `007_create_<name>s.sql`。

### 5.2 写入安全

生成器内部接受仓库根目录参数以便测试，CLI 仍对当前仓库执行操作。写入前必须校验仓库根目录、模块名、全部目标路径和 migration 编号。任何目标已存在时，操作整体失败且不覆盖文件。

所有模板先在内存中完成渲染，Go 源码也在内存中格式化，之后才创建文件。如果文件系统写入失败，删除本次调用创建的文件和目录，并保留调用前已经存在的内容。

## 6. OpenAPI

Swagger 注解和生成文档覆盖：

- Auth：register、login、refresh、logout 和 logout-all。
- User：Create、Get、List、Update 和 Delete。
- Role：Create、Get、List、Update、Delete 和权限分配。
- Permission：Create、Get、List、Update 和 Delete。
- Audit：List 和 Get。

契约记录具名请求/响应 Schema、bearer authentication、分页与排序参数、成功状态和稳定错误响应。CI 重新生成 OpenAPI，当生成结果与已提交文件不一致时失败。

## 7. 测试策略

实现遵循测试驱动开发。

- Application 测试覆盖 CRUD 成功、资源不存在、冲突和 Repository 错误映射。
- Handler 测试覆盖校验、HTTP 状态、响应结构、分页、Delete 空响应体和敏感字段排除。
- Repository 测试使用聚焦的 Gorm 测试设施覆盖分页与排序查询，以及 `Count`/数据查询错误传播。
- Generator 测试覆盖合法与非法名称、snake_case 转换、冲突保护、migration 编号、写入失败回滚、完整输出、无 TODO 标记、Go 格式化和临时仓库编译。
- OpenAPI 测试验证必需路径、方法、bearer security、分页参数和生成文件稳定性。

最终验证先运行聚焦测试，再运行 `go test ./...`、目标 race 测试、`go vet ./...`、Server 与 CLI 构建、连续两次 Swagger 生成且无 diff、Compose 校验和 Docker 镜像构建。

## 8. 交付边界

虽然 API 契约按一个整体系统设计，统一重构仍按逻辑拆分提交：

1. 共享响应与模块 CRUD 契约。
2. 完整模块生成器及其测试。
3. OpenAPI 覆盖和契约验证。

破坏性 API 变更仅限于已确认的状态码、DTO 和分页统一。不得包含无关模块重构。

## 9. 验收标准

满足以下条件时，本阶段完成：

1. 每个已声明的 User、Role、Permission 和 Audit 路由都有真实实现。
2. API 响应不会序列化密码哈希或软删除元数据。
3. 所有列表路由使用共享分页与排序契约。
4. `jimu module create product` 创建完整、已格式化、可编译的 CRUD 模块和 migration，无需手工修改。
5. 非法或冲突的生成器调用不会改变仓库内容。
6. 生成的 OpenAPI 包含所有已声明路由，并且重复生成结果稳定。
7. 聚焦测试、全量测试、race、vet、build、Compose 和 Docker 验证全部通过。
