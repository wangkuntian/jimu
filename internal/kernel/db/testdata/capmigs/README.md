# 能力迁移运行器集成测试夹具

供 internal/kernel/db/migration_integration_test.go 使用：两个最小测试能力
（user / auditsvc），各自带 migrations/{mysql,postgres} 目录，用于在真实
MySQL/Postgres 上验证 MigrateEnabled 的按能力独立版本表行为。

命名 capmig_ 前缀避免与真实业务表冲突；测试结束负责清理对应表与版本表。

## 不等深依赖对（zsvc / advc）

供 TestMigrationIntegration_UnequalDepths 使用，钉住 down 的清单反转序：

- `zsvc`：基础能力，2 条迁移，001 建 capmig_zitems、002 建 capmig_zshared；
- `advc`：依赖能力，1 条迁移，ALTER capmig_zshared（依赖 zsvc 002 的表）。

测试以 [zsvc, advc] 传入（真实清单同为"基础在前、依赖在后"的拓扑序）。
两能力名字 zsvc > advc：按名降序的旧 sort 实现会让 zsvc 002 的
DROP TABLE 先于 advc 001 Down 执行，第 1 轮 down 即报 "Table doesn't exist"；
正向序同样失败；只有清单反转序（advc 先回滚）三轮 down 全部成功。
