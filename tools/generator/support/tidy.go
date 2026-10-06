package support

// 本文件实现 ⑧ `go mod tidy`：生成项目的依赖从「本仓 go.mod 的超集」收缩成「本项目真实用到的
// 直接依赖」（层①让 go.mod 变小的那一步，设计 §11 的层②边界）。

// Tidy 在生成项目里跑 `go mod tidy`。两道环境约束：
//
//	GOFLAGS=-mod=mod —— tidy 需要改写 go.mod/go.sum；显式声明，避免使用者环境里的
//	                    GOFLAGS=-mod=readonly 让 tidy 静默失败；
//	GOWORK=off       —— 生成项目不在任何 go.work 里，避免被使用者的工作区联动
//	                    （照 tools/generator/compile_test.go 的既有做法）。
//
// 失败即错误（不吞、不降级为警告），由 `jimu new --no-tidy` 显式关闭。
func Tidy(dir string) error {
	return runGo(dir, []string{"GOFLAGS=-mod=mod", "GOWORK=off"}, "mod", "tidy")
}
