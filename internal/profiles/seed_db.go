//go:build !jimu_nodb

package profiles

import (
	"jimu/internal/assembly"
	"jimu/internal/platform/seeding"
)

// StructuralSeed 是各形态共用的既有结构性种子：默认租户、free 套餐、超管角色、
// admin 用户，加上按**本形态解析集**聚合的权限点，并同步 Casbin 策略
// （seeding.RunSeedWithCasbin，与 `jimu seed` 同一实现，幂等）。
//
// 权限点取自 assembly.Context.Capabilities()（Run 解析出的装配集），而非全量清单：
// import catalog 会把全部 catalog 能力包拉进每个形态的依赖闭包，形态裁剪随之失效。
//
// 迁移自 P2.6 起跟随形态裁剪，但**带上 schema 依赖**（见 catalog.MigrationSchemaDeps）：
// 含 user/access 的形态会一并迁移 tenant 的建表/加列，因此 tenants/tenant_plans 表与
// users/roles.tenant_id 列在各形态都存在，种子照常可执行。
// 种子需要部署期凭据 ADMIN_PASSWORD_FILE（回退 ADMIN_PASSWORD；RunSeed 的前置条件），未提供时跳过并告警：
// 容器启动早于 CLI 迁移、compose 也不向 server 注入该变量，服务启动不应因缺少该变量而失败。
// 跳过时还会在**库中无任何账号**的情况下补一条 error 级提示（seeding.WarnIfNoAccounts）：
// 关闭公开注册又没种出 admin 的部署将没有任何可用登录，必须显式告知下一步。
func StructuralSeed(ctx *assembly.Context) error {
	if ctx.DB() == nil {
		ctx.Logger().Warnw("structural seed skipped", "missing", "database")
		return nil
	}
	if seeding.AdminPassword() == "" {
		ctx.Logger().Warnw("structural seed skipped", "missing", "ADMIN_PASSWORD or ADMIN_PASSWORD_FILE")
		seeding.WarnIfNoAccounts(ctx.DB(), ctx.Logger())
		return nil
	}
	return seeding.RunSeedWithCasbin(ctx.DB(), ctx.Capabilities())
}
