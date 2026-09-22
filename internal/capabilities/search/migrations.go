package search

import (
	"embed"

	"jimu/internal/contract"
)

// migrationsFS 能力自带迁移（Task 3：能力迁移经 embed 进二进制）。
//
//go:embed migrations
var migrationsFS embed.FS

// Descriptor 能力静态描述：受保护挂载点，拥有 search_documents 表。
var Descriptor = contract.Descriptor{
	Name:       "search",
	Mount:      contract.MountProtected,
	Migrations: migrationsFS,
	Owns:       []string{"search_documents"},
}
