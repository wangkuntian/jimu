package scaffold

import (
	"fmt"
	"io"
	"strings"

	"jimu/internal/templatesource"
	"jimu/tools/generator"

	"github.com/spf13/cobra"
)

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

func NewAddCommand(resolve templatesource.Resolver) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add <name>",
		Short: "Add a capability to an existing generated project",
		Long: `Add a capability to an existing generated project.

读取 <dir>/.jimu/manifest.json（模块路径、声明集、驱动集），按同一套渲染管线在暂存目录
重渲染后逐文件落盘：生成文件（DO NOT EDIT）被覆盖，configs/*.yaml 与 deploy/helm/values.yaml
以现有文件为底合并（保住手改的段值），用户自有文件原样保留。写前复用组合 resolver/preflight，
冲突、缺失依赖、端口、驱动、迁移/配置/中间件问题都在落盘前失败；落盘使用与 upgrade 相同的
manifest 事务，任一失败按原字节回滚，不留半成品。

硬依赖必须已在项目里 —— 缺依赖直接报错并提示先加依赖，不会自动补全。既有驱动选择与
auto-added 语义会保留，支持级别保持 experimental。
能力已在 catalog entries 里时用 --force 才允许重建。预设（minimal/auth/full）是裁剪起点，
可以继续追加能力；带驱动的能力用 --driver 指定实现。`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var opts generator.AddOptions
			opts.Name = args[0]
			opts.Driver, _ = cmd.Flags().GetString("driver")
			opts.Dir, _ = cmd.Flags().GetString("dir")
			opts.Module, _ = cmd.Flags().GetString("module")
			opts.Force, _ = cmd.Flags().GetBool("force")
			opts.DryRun, _ = cmd.Flags().GetBool("dry-run")
			source, cleanup, err := resolveSource(cmd, resolve)
			if err != nil {
				return err
			}
			defer func() { _ = cleanup() }()
			opts.From = source.Root
			opts.Framework = source.Framework
			res, err := generator.AddCapability(opts)
			if err != nil {
				return err
			}
			printAddPlan(cmd.OutOrStdout(), res, opts.DryRun)
			return nil
		},
	}
	cmd.Flags().String("dir", ".", "generated project root (must contain .jimu/manifest.json)")
	cmd.Flags().String("driver", "", "driver for the added capability (default: the first declared driver)")
	cmd.Flags().String("module", "", "override the module path recorded in .jimu/manifest.json")
	cmd.Flags().Bool("force", false, "rebuild the capability even if it is already present")
	cmd.Flags().Bool("dry-run", false, "print the files that would change without writing anything")
	addSourceFlag(cmd)
	return cmd
}

func NewCapabilityCommand(resolve templatesource.Resolver) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "capability",
		Short: "Manage generated project capabilities",
	}
	cmd.AddCommand(NewAddCommand(resolve))
	return cmd
}
