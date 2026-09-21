package queue

import (
	"jimu/internal/capabilities/queue/application"
	"jimu/internal/capabilities/queue/domain"
	"jimu/internal/capabilities/queue/infrastructure"
	"jimu/internal/capabilities/queue/interfaces"
	"jimu/internal/contract"
	"jimu/internal/kernel/http/middleware"
	"jimu/internal/kernel/scheduler"

	"gorm.io/gorm"
)

// Module 队列能力的模块实例：注册任务队列与调度管理端点
// （/api/v1/admin/jobs* 与 /api/v1/admin/tasks*，表所有者 = queue）。
type Module struct {
	jobRepo  domain.JobRepository
	deadRepo domain.DeadLetterRepository
	sched    *scheduler.CronScheduler
}

// NewModule 创建队列模块（模块实例，区别于按类型建队列的 New）。
// sched 为空时任务端点仍注册，但调用返回错误。
func NewModule(db *gorm.DB, sched *scheduler.CronScheduler) *Module {
	return &Module{
		jobRepo:  infrastructure.NewMysqlJobRepository(db),
		deadRepo: infrastructure.NewMysqlDeadLetterRepository(db),
		sched:    sched,
	}
}

// Name 模块名（与 queue 能力同名，同属一个能力）。
func (m *Module) Name() string { return "queue" }

// Descriptor 复用 queue 能力描述。
func (m *Module) Descriptor() contract.Descriptor { return Descriptor }

// RegisterHTTP 注册任务队列与调度管理端点。
func (m *Module) RegisterHTTP(r contract.Router) {
	admin := r.Group("/api/v1/admin")
	admin.Use(middleware.AdminAuth())

	jobHandler := interfaces.NewAdminJobHandler(m.jobRepo, m.deadRepo)
	admin.GET("/jobs", jobHandler.List)
	admin.POST("/jobs", jobHandler.Submit)
	admin.GET("/jobs/:id", jobHandler.Get)
	admin.POST("/jobs/:id/retry", jobHandler.Retry)
	admin.GET("/jobs/dead-letters", jobHandler.ListDeadLetters)
	admin.POST("/jobs/dead-letters/:id/resolve", jobHandler.ResolveDeadLetter)

	taskHandler := interfaces.NewAdminTaskHandler(application.NewAdminTaskService(m.sched))
	admin.GET("/tasks", taskHandler.List)
	admin.POST("/tasks/:id/run", taskHandler.Trigger)
	admin.POST("/tasks/:id/toggle", taskHandler.Toggle)
	admin.GET("/tasks/:id/history", taskHandler.History)
}

func (m *Module) RegisterJobs(j contract.JobRegistry) {}

func (m *Module) RegisterEvents(e contract.EventBus) {}

var _ contract.Module = (*Module)(nil)
