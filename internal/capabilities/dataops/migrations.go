package dataops

import (
	"embed"

	"jimu/internal/contract"
)

// migrationsFS 能力自带迁移（Task 3：能力迁移经 embed 进二进制）。
//
//go:embed migrations
var migrationsFS embed.FS

// Descriptor 能力静态描述：受保护挂载点，拥有 import_jobs 表。
var Descriptor = contract.Descriptor{
	Name:       "dataops",
	Mount:      contract.MountProtected,
	Migrations: migrationsFS,
	Owns:       []string{"import_jobs"},
}
