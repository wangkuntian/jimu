package main

import (
	"fmt"
	"io"
	"strings"

	"jimu/tools/generator"

	"github.com/spf13/cobra"
)

// capabilityCmd 是框架侧的脚手架命令组：往**已生成项目**里增量加入能力。它依赖
// tools/generator（生成项目不含该工具树），因此不进生成项目（见 kernelExcludes/RenderCLIMain）。
var capabilityCmd = newCapabilityCmd()

func newCapabilityCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "capability",
		Short: "Create framework capabilities and manage generated projects",
	}
	cmd.AddCommand(newCapabilityCreateCmd())
	cmd.AddCommand(newCapabilityAddCmd())
	return cmd
}

func newCapabilityCreateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "create <name>",
		Short: "Create a new capability skeleton in the framework repository",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return generator.GenerateModule(args[0])
		},
	}
}

// printAddPlan 打印 add 的计划/结果：--dry-run 逐行列出将新增/改动的文件（唯一不落盘的计划输出），
// 非 dry-run 只报一行摘要（与 new 的「成功时安静」一致，文件级证据在测试与报告中）。
func printAddPlan(w io.Writer, res *generator.Result, dryRun bool) {
	var b strings.Builder
	if len(res.Changed) == 0 {
		fmt.Fprintf(&b, "no changes: %s already matches the declared capability set\n", res.Dir)
		_, _ = io.WriteString(w, b.String())
		return
	}
	fmt.Fprintf(&b, "changed %d file(s) in %s\n  capabilities %s\n",
		len(res.Changed), res.Dir, strings.Join(res.Capabilities, ", "))
	if dryRun {
		for _, rel := range res.Changed {
			b.WriteString("  " + rel + "\n")
		}
	}
	_, _ = io.WriteString(w, b.String())
}

func newCapabilityAddCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add <name>",
		Short: "Add a capability to an existing generated project",
		Long: `Add a capability to an existing generated project.

读 <dir>/.jimu-generated（模块路径、声明集、驱动集、框架源根），按同一套渲染管线在暂存目录
重渲染后逐文件落盘：生成文件（DO NOT EDIT）被覆盖，configs/*.yaml 与 deploy/helm/values.yaml
以现有文件为底合并（保住手改的段值），用户自有文件原样保留。任一失败即回滚，不留半成品。

硬依赖必须已在项目里 —— 缺依赖直接报错并提示先加依赖，不会自动补全。
能力已在 catalog entries 里时用 --force 才允许重建。`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var opts generator.AddOptions
			opts.Name = args[0]
			opts.Dir, _ = cmd.Flags().GetString("dir")
			opts.From, _ = cmd.Flags().GetString("from")
			opts.Module, _ = cmd.Flags().GetString("module")
			opts.Force, _ = cmd.Flags().GetBool("force")
			opts.DryRun, _ = cmd.Flags().GetBool("dry-run")
			res, err := generator.AddCapability(opts)
			if err != nil {
				return err
			}
			printAddPlan(cmd.OutOrStdout(), res, opts.DryRun)
			return nil
		},
	}
	cmd.Flags().String("dir", ".", "generated project root (must contain .jimu-generated)")
	cmd.Flags().String("from", "", "framework source root (default: .jimu-generated sourceRoot)")
	cmd.Flags().String("module", "", "override the module path recorded in .jimu-generated")
	cmd.Flags().Bool("force", false, "rebuild the capability even if it is already present")
	cmd.Flags().Bool("dry-run", false, "print the files that would change without writing anything")
	return cmd
}
