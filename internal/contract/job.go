package contract

// Job 是能力经装配接缝贡献的定时任务定义：由组合根在内建任务之外一并注册。
// ID 全仓唯一（重名报错），Name/Spec 语义与调度器注册接口一致。
type Job struct {
	ID   string
	Name string
	Spec string
	Run  func()
}
