# 新增 / 删除一个能力

**何时读**：往本仓加一个新能力（或删掉一个），或给能力补权限点、配置段、迁移归属、资产声明时。

**权威口径**：README「[开发规范 › 新增能力 / 驱动](../../../README.md#新增能力--驱动)」、「[开发规范 › 模块结构](../../../README.md#模块结构)」、「[能力清单](../../../README.md#能力清单)」。

## 步骤

1. **建能力目录**：`internal/capabilities/<name>/`，按需分层（`domain/` / `application/` / `infrastructure/` / `interfaces/`）。用脚手架打底：`./bin/jimu capability create <name>`（只落典型 CRUD 骨架，**不改任何注册点**）
2. **导出静态 `Descriptor`**（`contract.Descriptor`，字段定义见 `internal/contract/capability.go`）：
   - `Name`：能力名，全仓唯一，必须与 `catalog` 清单里的名字一致
   - `Requires`：**硬依赖** —— 启用本能力必须同时启用这些能力，参与闭包补齐与拓扑序
   - `SoftRequires`：**软依赖** —— 目标缺失时本能力降级运行，不补齐、不参与拓扑序
   - `Owns`：本能力迁移 `CREATE` 的表名（**只认 `CREATE TABLE`**，`ALTER ... ADD` 不算归属）
   - `Migrations`：能力自带迁移的 `fs.FS`（根下有 `mysql/` 与 `postgres/`）
   - `Configs`：本能力拥有的配置段（`contract.ConfigSpec{Section, New}`），可多段
   - `Permissions`：能力拥有的权限点（种子时按启用集聚合写入）
   - `Mount`：路由挂载方式（零值等价 `MountProtected`，特权路由不会被裸挂到根路由）
   - `Drivers`：能力支持的**驱动包名**（有第三方驱动时才写，见 [profile-driver.md](profile-driver.md)）
   - `Assets`：能力拥有的非代码资产（仓库相对路径；有资产时才写）
3. **导出 `Wire` 自装配**，有 HTTP、任务、事件或生命周期职责时实现 `contract.Module`；仅提供端口或迁移的能力可返回 nil。能力内部只依赖 `contract` 端口；**禁止**跨能力 import（包括测试和驱动子包）
4. **在 `internal/capabilities/catalog` 登记**该能力
5. **在需要它的形态清单 `internal/profiles/<name>/assembly.go` 里加入**；非 catalog 条目按 `Ungated` 声明。唯一入口 `cmd/server` 只调 `assembly.Run(active.Assembly())`，**不要**去 `cmd/server` 里逐个装配能力
6. **迁移**：写进 `internal/capabilities/<name>/migrations/{mysql,postgres}/`，能力内编号取该目录当前最大 +1；细节见 [migration.md](migration.md)
7. **配置段**：新增配置段时同步 README「[配置项](../../../README.md#配置项)」与 `configs/*.yaml`（键一经发布不得随意改动）
8. **资产**：声明 `Assets` 后跑门禁确认归属唯一（见 [gates.md](gates.md)）
9. **删除能力**：从形态清单与 `catalog` 摘除 → 迁移与表按「运行时不删表」的既有取舍保留（README「[能力开关（v0.3.0）](../../../README.md#能力开关v030)」）→ 跑门禁确认没有悬空引用

## 验收命令与期望输出

```bash
make check-capabilities           # 必须 7 条 ✅（① 自描述与 Owns ↔ 迁移归属 ② 驱动 ③ 形态只 import 已声明驱动 ④ 唯一入口与选点包 ⑤ 资产归属 ⑥ 跨能力 import ⑦ 生成器边界）
make profiles-check               # 5 形态 overlay 构建 + golden 依赖闭包通过
go test ./internal/... -count=1   # 能力自带单测全绿
```

## 常见红与定位

| 症状 | 定位 |
|---|---|
| `check-capabilities` ① 号红：声明的表没有迁移创建 / 一张表被两个能力声明 | `Owns` 与实际 `CREATE TABLE` 逐值对齐；`ALTER TABLE ... ADD` 不参与归属 |
| `check-capabilities` ④ 号红：选点包 import 了不止一个形态 | 只改形态包，**不要**动 `internal/profiles/active` 与 `cmd/server` |
| 能力声明了但没生效 | 该能力不在当前形态清单里，或 `capabilities.enabled` 排除了它 —— 见 [runtime.md](runtime.md) |
| 启动报「多个受保护中间件提供者」 | 同一启用集只允许一个能力**真实**提供受保护链，见 [runtime.md](runtime.md) |
