// Package profiles 提供各形态（profile）共享的装配构件：构建版本默认值与结构性种子。
//
// 形态分两层：internal/profiles/<name> 声明该形态的能力清单（package 非 main），
// 唯一入口 cmd/server 经选点包 internal/profiles/active 取当前形态（提交态默认 full，
// 构建期由 tools/profileoverlay 的 overlay 切换）；共享件只放在本包，避免各形态互相 import。
package profiles

import (
	"os"

	"jimu/internal/app"
	"jimu/internal/assembly"
)

// Version 是形态清单默认填入的构建版本；唯一入口可经 ldflags 注入后覆盖
// （cmd/server 沿用 `-X main.version`，Makefile/Dockerfile 不变）。
var Version = "dev"

// StructuralSeed 是各形态共用的既有结构性种子：默认租户、free 套餐、超管角色、
// admin 用户，加上按**本形态解析集**聚合的权限点，并同步 Casbin 策略
// （app.RunSeedWithCasbin，与 `jimu seed` 同一实现，幂等）。
//
// 权限点取自 assembly.Context.Capabilities()（Run 解析出的装配集），而非全量清单：
// import catalog 会把 18 个能力的包全量拉进每个形态的依赖闭包，形态裁剪随之失效。
//
// 迁移自 P2.6 起跟随形态裁剪：结构种子需要 tenant 能力（tenants/tenant_plans 表与
// users/roles.tenant_id 列都由它的迁移产生），不含该能力的形态无法播种（ORM 模型始终写
// tenant_id 列）。此时告警跳过而不是让服务启动失败 —— 与缺少 ADMIN_PASSWORD 同一处置；
// 管理数据改用具备 tenant 的形态（full/saas）执行 `jimu seed`。
// 种子需要部署期凭据 ADMIN_PASSWORD（RunSeed 的前置条件），未提供时同样跳过并告警：
// 容器启动早于 CLI 迁移、compose 也不向 server 注入该变量，服务启动不应因缺少该变量而失败。
func StructuralSeed(ctx *assembly.Context) error {
	if os.Getenv("ADMIN_PASSWORD") == "" {
		ctx.Logger().Warnw("structural seed skipped", "missing", "ADMIN_PASSWORD")
		return nil
	}
	if !app.HasCapability(ctx.Capabilities(), "tenant") {
		// 解析集不含 tenant 有两种成因：该形态本就不含它，或被 capabilities.enabled 关闭
		// （后者表/列仍在，CLI seed 按编译形态判断仍可用）—— 日志不要把两种混为一谈。
		ctx.Logger().Warnw("structural seed skipped",
			"reason", "tenant capability not active",
			"hint", "shape does not include tenant, or it is disabled via capabilities.enabled")
		return nil
	}
	return app.RunSeedWithCasbin(ctx.DB(), ctx.Capabilities())
}
