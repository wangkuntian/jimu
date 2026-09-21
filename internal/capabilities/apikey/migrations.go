package apikey

import (
	"embed"

	"jimu/internal/contract"
)

// migrationsFS 能力自带迁移（Task 3：能力迁移经 embed 进二进制）。
//
//go:embed migrations
var migrationsFS embed.FS

// Descriptor 能力静态描述：queue 等基础设施能力尚无独立挂载点，
// 此处仅携带迁移与身份信息，供迁移运行器与后续 catalog 扩展消费。
var Descriptor = contract.Descriptor{
	Name:       "apikey",
	Mount:      contract.MountProtected,
	Migrations: migrationsFS,
}
