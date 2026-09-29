package apikey

import (
	"embed"

	"jimu/internal/contract"
)

// migrationsFS 能力自带迁移（Task 3：能力迁移经 embed 进二进制）。
//
//go:embed migrations
var migrationsFS embed.FS

// Descriptor 能力静态描述：受保护挂载点，拥有 api_keys 表，
// 可选依赖 tenant（缺省时不做租户配额校验）。
var Descriptor = contract.Descriptor{
	Name:         "apikey",
	SoftRequires: []string{"tenant"},
	Mount:        contract.MountProtected,
	Migrations:   migrationsFS,
	Owns:         []string{"api_keys"},
}
