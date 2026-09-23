// Command profileoverlay 为指定形态生成 overlay 配置：把
// internal/profiles/active/assembly.go 在**本次构建**中替换为「只选该形态」的版本，
// 不改动工作区里的任何文件。
//
//	go run ./tools/profileoverlay <profile>   # 写 .overlay/<profile>/{active.go,overlay.json}，打印 JSON 路径
//	go run ./tools/profileoverlay -list       # 打印全部形态名（每行一个）
//
// 用法示例（先赋值再构建）：形态名非法时本工具非零退出，但**命令替换的失败不会改变外层命令
// 的退出码** —— 直接写进 `-overlay=$(...)` 只会留下空值，go build 把空 overlay 当作「无
// overlay」而静默按提交态（full）构建。赋值语句的退出码就是命令替换的退出码，`&&` 因而能截断：
//
//	overlay=$(go run ./tools/profileoverlay minimal) && go build -overlay="$overlay" -o bin/server ./cmd/server
//	overlay=$(go run ./tools/profileoverlay minimal) && go test -overlay="$overlay" ./internal/profiles/...
//
// 替换内容模板与产物路径统一由共享包 tools/internal/profileoverlay 提供（门禁与报告同一份）。
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"jimu/internal/profiles/registry"
	"jimu/tools/internal/profileoverlay"
)

func main() {
	if len(os.Args) == 2 && os.Args[1] == "-list" {
		fmt.Println(strings.Join(registry.Names(), "\n"))
		return
	}
	if len(os.Args) != 2 {
		fail(fmt.Errorf("用法：profileoverlay <profile> | -list"))
	}
	root, err := repoRoot()
	if err != nil {
		fail(err)
	}
	jsonPath, err := write(root, os.Args[1])
	if err != nil {
		fail(err)
	}
	fmt.Println(jsonPath)
}

func fail(err error) { fmt.Fprintln(os.Stderr, "❌ profileoverlay:", err); os.Exit(1) }

// write 生成该形态的 overlay 产物并返回 overlay JSON 的绝对路径。模板与路径口径都来自
// 共享包 tools/internal/profileoverlay，本命令不再持有副本（漂移会让消费者断言失败）。
func write(root, profile string) (string, error) {
	return profileoverlay.WriteFiles(root, profile)
}

// repoRoot 从当前工作目录向上找 go.mod（make/Docker 都从仓库根调用，此处兼容子目录调用）。
func repoRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("找不到 go.mod（从 %s 向上）", dir)
		}
		dir = parent
	}
}
