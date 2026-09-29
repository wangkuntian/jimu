package notification

import "jimu/internal/contract"

// PortName 通知能力对外提供的端口名：Dispatcher。
const PortName = "notification"

// Descriptor 声明通知能力的静态描述（非 catalog 条目：profile 显式列出）。
// 无 HTTP 路由、无迁移、无 Module 实例，只提供 Dispatcher/Hub 端口。
var Descriptor = contract.Descriptor{
	Name:  "notification",
	Mount: contract.MountProtected,
}
