package app

// export_test.go 只把 outbox 事件桥接的内部函数暴露给 app_test（仅测试构建可见，
// 生产 API 不变）。package app 的内测 import outbox/queue 会在能力 wire.go 引入
// 驱动包后成环（app → outbox → assembly → app），故这些用例必须以外部测试包运行。
var (
	BridgeFn               = bridgeFn
	RegisterEventBusBridge = registerEventBusBridge
	RegisterOutboxWorkers  = registerOutboxWorkers
)
