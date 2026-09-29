package breach

import "jimu/internal/contract"

// PortName 泄露口令检查能力对外提供的端口名：contract.BreachChecker。
// 仅在 auth.breach_check_enabled 打开时注册（关闭时端口缺席，auth 降级跳过检查）。
const PortName = "breach"

// Descriptor 声明泄露口令检查能力的静态描述。
// breach 为外部服务依赖的库形态能力：无 HTTP 路由、无迁移、无 Module 实例，
// 仅进 catalog 清单并经 contract.BreachChecker 端口注入 auth。
var Descriptor = contract.Descriptor{
	Name:  "breach",
	Mount: contract.MountProtected,
}
