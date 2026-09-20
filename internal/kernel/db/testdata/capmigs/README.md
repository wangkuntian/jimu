# 能力迁移运行器集成测试夹具

供 internal/kernel/db/migration_integration_test.go 使用：两个最小测试能力
（user / auditsvc），各自带 migrations/{mysql,postgres} 目录，用于在真实
MySQL/Postgres 上验证 MigrateEnabled 的按能力独立版本表行为。

命名 capmig_ 前缀避免与真实业务表冲突；测试结束负责清理对应表与版本表。
