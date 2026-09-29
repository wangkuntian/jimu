package encryption

import "jimu/internal/contract"

// PortName 字段级加密能力对外提供的端口名：*Cipher。
const PortName = "encryption"

// Descriptor 声明字段级加密能力的静态描述（非 catalog 条目：profile 显式列出）。
// 无 HTTP 路由、无迁移、无 Module 实例，只把 *Cipher 暴露为端口供其它能力消费。
var Descriptor = contract.Descriptor{
	Name:  "encryption",
	Mount: contract.MountProtected,
}
