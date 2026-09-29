package outbox

import (
	"embed"

	"jimu/internal/contract"
)

// migrationsFS 能力自带迁移（Task 3：能力迁移经 embed 进二进制）。
//
//go:embed migrations
var migrationsFS embed.FS

// Descriptor 能力静态描述：受保护挂载点，拥有 outbox_events 表，
// 可选依赖 queue（publisher=mq 时才需要，缺省时走 event_bus 降级），并声明 outbox 配置段。
var Descriptor = contract.Descriptor{
	Name:         "outbox",
	SoftRequires: []string{"queue"},
	Mount:        contract.MountProtected,
	Migrations:   migrationsFS,
	Owns:         []string{"outbox_events"},
	Configs: []contract.ConfigSpec{
		{Section: ConfigKey, New: func() any { return &Config{} }},
	},
}
