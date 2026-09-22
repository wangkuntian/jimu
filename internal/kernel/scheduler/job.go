package scheduler

import "fmt"

// Job 是能力经装配接缝贡献的定时任务定义：由组合根在内建任务之外一并注册。
// ID 全仓唯一（重名报错），Name/Spec 语义与 CronScheduler.AddNamedFunc 一致。
type Job struct {
	ID   string
	Name string
	Spec string
	Run  func()
}

// 调度器存储类型取值（原 queue.SchedulerConfig，P2.4 归位内核：调度器是内核件，
// 其配置段由 queue 能力声明，但类型不属任何能力）。
const (
	// ConfigKey 调度器配置段键（app.yaml）。
	ConfigKey = "scheduler"
	// StoreMemory 内存存储：任务定义不持久化。
	StoreMemory = "memory"
	// StoreMySQL MySQL 存储：任务定义持久化到 scheduled_jobs。
	StoreMySQL = "mysql"
)

var validStores = []string{StoreMemory, StoreMySQL}

// Config 调度器配置段（mapstructure 键名与下沉前一致）。
type Config struct {
	Store string `mapstructure:"store"` // 任务定义存储类型：memory, mysql
}

// ApplyDefaults 本配置段无配置层默认值（container 对非 mysql 一律按 memory 处理）。
func (c *Config) ApplyDefaults() {}

// Validate 校验调度器配置段。
func (c *Config) Validate() error {
	for _, s := range validStores {
		if c.Store == s {
			return nil
		}
	}
	return fmt.Errorf("scheduler.store: %q, must be one of %v", c.Store, validStores)
}
