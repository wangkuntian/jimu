package breach

import "jimu/internal/contract"

// Descriptor 声明泄露口令检查能力的静态描述。
// breach 为外部服务依赖的库形态能力：无 HTTP 路由、无迁移、无 Module 实例，
// 仅进 catalog 清单并经 contract.BreachChecker 端口注入 auth。
var Descriptor = contract.Descriptor{
	Name:  "breach",
	Mount: contract.MountProtected,
}
