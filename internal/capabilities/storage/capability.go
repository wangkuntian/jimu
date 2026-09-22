package storage

import "jimu/internal/contract"

// PortName 文件存储能力对外提供的端口名：Storage。
const PortName = "storage"

// Descriptor 声明文件存储能力的静态描述（非 catalog 条目：profile 显式列出）。
// 无 HTTP 路由、无迁移、无 Module 实例，只把 Storage 暴露为端口供 uploadsec 消费。
var Descriptor = contract.Descriptor{
	Name:    "storage",
	Mount:   contract.MountProtected,
	Drivers: []string{"local", "s3"},
}
