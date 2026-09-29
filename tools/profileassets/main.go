// Command profileassets 打印某形态的资产清单，供构建脚本（APIdocs 条件化）与 P2.7 生成器消费。
//
//	go run ./tools/profileassets <profile>                 # 该形态的资产路径，一行一个
//	go run ./tools/profileassets -capabilities <profile>   # 该形态的能力名，一行一个
//
// 形态名来自 internal/profiles/registry；未知形态报错并列出可用形态。
package main

import (
	"fmt"
	"io"
	"os"
	"strings"

	"jimu/internal/profiles/registry"
	"jimu/tools/internal/profileassets"
)

func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "❌ profileassets:", err)
		os.Exit(1)
	}
}

// run 是可测的入口：解析参数、打印清单。
func run(args []string, out io.Writer) error {
	capabilities := false
	if len(args) > 0 && args[0] == "-capabilities" {
		capabilities = true
		args = args[1:]
	}
	if len(args) != 1 {
		return fmt.Errorf("用法：profileassets [-capabilities] <profile>")
	}
	profile := args[0]

	if capabilities {
		a, err := registry.Lookup(profile)
		if err != nil {
			return err
		}
		names := make([]string, 0, len(a.Capabilities))
		for _, c := range a.Capabilities {
			names = append(names, c.Descriptor.Name)
		}
		if _, err := fmt.Fprintln(out, strings.Join(names, "\n")); err != nil {
			return err
		}
		return nil
	}

	paths, err := profileassets.ForProfile(profile)
	if err != nil {
		return err
	}
	if _, err := fmt.Fprintln(out, strings.Join(paths, "\n")); err != nil {
		return err
	}
	return nil
}
