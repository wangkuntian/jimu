# 迁移编写与存量库 adopt

**何时读**：新增/修改迁移、给已有表加列、处理存量实例升级、或要动 `migrate`/`seed` 的行为时。

**权威口径**：README「[数据库迁移](../../../README.md#数据库迁移)」、「[数据种子](../../../README.md#数据种子)」、[docs/releases/v0.3.0.md](../../../docs/releases/v0.3.0.md) 的「说明」段。

## 步骤

1. **写进所属能力的目录**：`internal/capabilities/<name>/migrations/mysql/` 与 `.../postgres/`（两个方言都要写；门禁的归属判定只扫 mysql，但 PostgreSQL 迁移必须同步、表名与 mysql 一致）
2. **能力内编号**：取该目录当前最大编号 +1（脚手架 `jimu capability create` 自动完成；也可手工 `ls` 后递增）。**不要**用全局编号 —— 各能力各自独立编号，已迁移的能力沿用历史编号
3. **一条 ALTER 只属于一个能力**：它改变的表归谁，迁移就写谁的能力目录 —— 即使这条 ALTER 是别的能力需要的
4. **注释写清 up/down 与在 MySQL / PostgreSQL 下的差异**（SQL 片段最好在注释里给出）
5. **迁移集带上 schema 依赖**：迁移集 = 形态声明集 ∪ schema 依赖（`user`/`access` → `tenant`，实现见 `cmd/cli/activecaps.go` 的 `catalog.MigrationSchemaDeps`）。`users` / `roles.tenant_id` 只由 tenant 的迁移创建，因此含 `user`/`access` 的形态也会一并迁移 tenant 的建表/加列 —— 否则会建出**写不进去**的 schema（只影响迁移，装配集仍不含 tenant）
6. **`seed` 与迁移同步**：结构性种子在各形态都照常执行；改动种子同步 README「[数据种子](../../../README.md#数据种子)」

## 运行方式（都跟随**编译期形态**）

```bash
./bin/jimu migrate up                    # 执行全部待应用迁移
./bin/jimu migrate status                # 查看各能力版本表状态
./bin/jimu migrate down                  # 每能力回滚其最后一条迁移（按反向能力序）
./bin/jimu migrate redo                  # 每能力重做其最后一条迁移
./bin/jimu migrate adopt-capabilities    # 存量库：登记各能力版本表基线
PROFILE=minimal make migrate-status      # make 目标同样叠加形态 overlay（表数下降）
```

`capabilities.enabled`（层③）**不参与**：关闭能力不删表不删数据（刻意设计，避免误删生产数据）。

## 存量实例升级路径

旧二进制 `migrate up` 到旧世界最新 → 部署 v0.3.0 新二进制 → `jimu migrate adopt-capabilities` 登记各能力版本表基线 → 之后正常 `migrate up`。全局 `goose_db_version` 保留为历史记录，新运行器不再读写；全新数据库直接 `migrate up`。

## 已知限制（务必先看）

- **`down`/`redo` 是「每轮每能力回滚一步」的轮转模型**：按**反向能力序**迭代（依赖方的迁移先回滚，避免基表被先删）。各能力迁移深度不一时，回滚到底（drain-to-empty）可能因跨能力表依赖失败（如 `tenant` 005 Down 引用 `roles`）
- 日常运维只需回滚最近一步不受影响；**需要整库回滚到空时改用重建库**（CLI 暂无按能力过滤参数）
- **同一数据库不要混用形态做迁移**：各能力版本表 `goose_db_version_<capability>` 按当前形态的迁移集记录，混用会让版本表与实际 schema 错配（建议整库统一用 `full`）

## 验收命令与期望输出

```bash
go test ./internal/kernel/db/... -count=1     # 真实库集成用例；无库时按设计 SKIP
make migrate-status                           # 各能力版本表状态正常
PROFILE=minimal make migrate-status           # 轻形态只看得到该形态（+ schema 依赖）的表
make check-capabilities                       # ① 号断言：Owns ↔ 迁移「单表唯一归属、无孤儿表、无未声明建表」
```

## 常见红与定位

| 症状 | 定位 |
|---|---|
| ① 号断言：某表有 `CREATE TABLE` 但没有能力声明 `Owns` | 在拥有该表的能力 `Descriptor.Owns` 里补上，或把建表迁移搬回正确的能力 |
| ① 号断言：同一张表被两个能力声明 | 一张表只能有一个所有者；`ALTER TABLE ... ADD` 不算归属，只有 `CREATE TABLE` 参与判定 |
| 同一能力内重复版本号 | 重排该能力目录内的编号（各能力独立编号，只看自己目录） |
| 轻形态启动后写库报缺列（如 `tenant_id`） | 迁移集漏了 schema 依赖，检查 `cmd/cli/activecaps.go` 的 `catalog.MigrationSchemaDeps` |
| `migrate down` 中途失败 | 轮转模型的已知限制，见上文；改用重建库或逐能力检查版本表 |
