package generator

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// 本文件实现 ⑨ 自检：生成项目在**落盘之前**（暂存目录里）必须能构建并通过自己的能力门禁 ——
// 这是「模板/复制口径 vs 真实框架结构」漂移的 fail-closed 捕获网（模板与本仓结构之间没有编译期
// 约束，漂移只能靠真构建发现）。

// SelfCheck 在生成项目 dir 里依次跑 `go build ./...` 与 `go run ./tools/checkcapabilities`，
// 两处都必须绿（任一失败即返回错误，调用方据此回滚，绝不把不能构建的项目留给使用者）。
//
// 用生成项目**自带**的门禁（tools/checkcapabilities 是复制进产物的）而不是本仓的：门禁校验的是
// 生成树的 Descriptor/驱动/资产一致性，必须是被检查的那棵树自己的判定。
func SelfCheck(dir string) error {
	if err := runGo(dir, []string{"GOWORK=off"}, "build", "./..."); err != nil {
		return err
	}
	return runGo(dir, []string{"GOWORK=off"}, "run", "./tools/checkcapabilities")
}

// runGo 在 dir 里跑一个 go 子命令。env 追加在继承的环境之后（同名变量以后者为准）。
//
// 输出被捕获后**成功才透出**（`go run ./tools/checkcapabilities` 的 7 条 ✅ 是给使用者的回执），
// 失败则并进错误信息 —— 构建失败时看不到编译器输出的错误最难排查。
//
// 用 CommandContext（`noctx` 门禁要求）：生成器没有可传播的请求上下文 —— Tidy/SelfCheck 的签名是
// 简报固定的 `(dir string) error`，而 NewProject 也不接收 ctx（CLI 一次性调用）。因此这里用
// context.Background()：语义等价于 exec.Command，只是不给子进程留下「无 context」的调用形态。
func runGo(dir string, env []string, args ...string) error {
	cmd := exec.CommandContext(context.Background(), "go", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), env...)
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("go %s: %w\n%s", strings.Join(args, " "), err, strings.TrimRight(buf.String(), "\n"))
	}
	if buf.Len() > 0 {
		_, _ = os.Stdout.Write(buf.Bytes())
	}
	return nil
}
