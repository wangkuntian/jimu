package ws

import "jimu/internal/contract"

// Descriptor 声明 WebSocket 能力的静态描述（非 catalog 条目：profile 显式列出）。
// ws 是供 console 等能力复用的客户端 Hub 库，装配期无自建实例与端口。
var Descriptor = contract.Descriptor{
	Name:  "ws",
	Mount: contract.MountProtected,
}
