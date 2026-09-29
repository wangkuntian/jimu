package retention

import "jimu/internal/contract"

// Descriptor 声明数据保留能力的静态描述（非 catalog 条目：profile 显式列出）。
// 无 HTTP 路由、无迁移、无 Module 实例，只向组合根贡献 cleanup/retention 定时任务。
var Descriptor = contract.Descriptor{
	Name:  "retention",
	Mount: contract.MountProtected,
}
