package queue

import (
	"embed"

	"jimu/internal/contract"
)

// migrationsFS 能力自带迁移（Task 3：能力迁移经 embed 进二进制）。
//
//go:embed migrations
var migrationsFS embed.FS

// Descriptor 能力静态描述：受保护挂载点，拥有 jobs/job_history/dead_letters/scheduled_jobs 四表，
// 并声明 queue 与 scheduler 两个配置段。
var Descriptor = contract.Descriptor{
	Name:       "queue",
	Mount:      contract.MountProtected,
	Migrations: migrationsFS,
	Owns:       []string{"jobs", "job_history", "dead_letters", "scheduled_jobs"},
	Drivers:    []string{"redis", "kafka", "rabbitmq"},
	// 调度器实例由本能力用于作业调度（/admin/tasks*、job_history），其配置段随之归本能力
	Configs: []contract.ConfigSpec{
		{Section: ConfigKey, New: func() any { return &Config{} }},
		{Section: SchedulerConfigKey, New: func() any { return &SchedulerConfig{} }},
	},
}
