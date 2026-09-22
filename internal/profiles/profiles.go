// Package profiles 提供各形态（profile）共享的装配构件：构建版本默认值与结构性种子。
//
// 形态入口分两层：internal/profiles/<name> 声明该形态的能力清单（package 非 main），
// profiles/<name>/main.go 只是调用 assembly.Run 的薄入口；共享件只放在本包，
// 避免各形态互相 import。
package profiles

import (
	"os"

	"jimu/internal/app"
	"jimu/internal/assembly"
	"jimu/internal/capabilities/catalog"
)

// Version 是形态清单默认填入的构建版本；入口包可经 ldflags 注入后覆盖
// （cmd/server 与 profiles/full 沿用 `-X main.version`，Makefile/Dockerfile 不变）。
var Version = "dev"

// StructuralSeed 是各形态共用的既有结构性种子：默认租户、free 套餐、超管角色、
// admin 用户，加上按**全量清单**聚合的权限点，并同步 Casbin 策略
// （app.RunSeedWithCasbin，与 `jimu seed` 同一实现，幂等）。
//
// 形态裁剪不门控种子：CLI 的 migrate 按完整清单建表，被形态排除的能力表同样存在
// （profile 驱动的迁移裁剪是 P2.6/P2.8 工作，本阶段不实现）。种子需要部署期凭据
// ADMIN_PASSWORD（RunSeed 的前置条件），未提供时跳过并告警：容器启动早于 CLI 迁移、
// compose 也不向 server 注入该变量，服务启动不应因缺少该变量而失败。
func StructuralSeed(ctx *assembly.Context) error {
	if os.Getenv("ADMIN_PASSWORD") == "" {
		ctx.Logger().Warnw("structural seed skipped", "missing", "ADMIN_PASSWORD")
		return nil
	}
	return app.RunSeedWithCasbin(ctx.DB(), catalog.All())
}
