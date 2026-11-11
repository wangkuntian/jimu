package cache

import (
	"fmt"

	"jimu/internal/assembly"
	"jimu/internal/contract"
)

// Wire 装配缓存能力：解码 cache 段、按驱动构造实例，并把 Cache 暴露为端口。
//
// Redis 客户端由容器在装配期注入（能力不自己连库）；memory 驱动忽略它。
//
// 选了 redis 驱动但本构建没有 Redis 客户端时降级到 memory 并告警：缓存是可选
// 加速层，不该因为缺少外部服务让整个进程起不来（与 auth 的存储降级同口径）。
func Wire(ctx *assembly.Context) (contract.Module, error) {
	cfg, err := Load(ctx.Sections())
	if err != nil {
		return nil, fmt.Errorf("init cache: %w", err)
	}
	var client any
	if c := ctx.Redis(); c != nil {
		client = c
	}
	driver := cfg.Driver
	if driver == "redis" && client == nil {
		ctx.Logger().Warnw("cache driver redis is selected but no redis client is compiled in; falling back to memory",
			"driver", driver)
		driver = "memory"
	}
	svc, err := New(driver, Options{Prefix: cfg.Prefix, Client: client})
	if err != nil {
		return nil, fmt.Errorf("init cache: %w", err)
	}
	if err := ctx.Provide(PortName, svc); err != nil {
		return nil, fmt.Errorf("provide cache port: %w", err)
	}
	return nil, nil
}
