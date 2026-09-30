package ws

import "jimu/internal/contract"

// Descriptor 声明 WebSocket 能力的静态描述（非 catalog 条目：profile 显式列出）。
// ws 为 console 提供管理端实时连接端口，不自行注册 HTTP 路由。
var Descriptor = contract.Descriptor{
	Name:  "ws",
	Mount: contract.MountProtected,
}
