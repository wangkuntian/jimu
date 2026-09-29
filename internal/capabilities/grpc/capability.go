package grpc

import "jimu/internal/contract"

// Descriptor 声明 gRPC 接入能力的静态描述（非 catalog 条目：profile 显式列出）。
// 无 HTTP 路由、无迁移、无 Module 实例；贡献一个 contract.Component（gRPC server）。
var Descriptor = contract.Descriptor{
	Name:  "grpc",
	Mount: contract.MountProtected,
}
