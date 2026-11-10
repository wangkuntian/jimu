package cache

import "jimu/internal/contract"

// PortName 缓存能力对外提供的端口名。
const PortName = "cache"

// Descriptor 声明缓存能力的静态描述（非 catalog 条目，由 profile 显式列出）。
//
// 能力语义是「缓存」，驱动是实现：redis 提供跨实例共享，memory 只在单实例内有效。
// 无 HTTP 路由、无迁移、无 Module 实例，只把 Cache 暴露为端口供其它能力消费。
var Descriptor = contract.Descriptor{
	Name:     "cache",
	Mount:    contract.MountProtected,
	Drivers:  []string{"redis", "memory"},
	Configs:  []contract.ConfigSpec{{Section: ConfigKey, New: func() any { return &Config{} }}},
	Provides: []contract.ProvidedPort{{Name: PortName}},
	DriverSpecs: []contract.DriverSpec{
		{Name: "redis", RuntimeServices: []contract.RuntimeService{contract.RuntimeServiceRedis}},
		{Name: "memory"},
	},
}
