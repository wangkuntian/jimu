package main

import (
	"fmt"
	"io"
	"strings"

	"jimu/tools/generator"

	"github.com/spf13/cobra"
)

// printScaffoldPlan 打印 --dry-run 的计划（不落盘）。先拼成一段再写：只用一个写调用，
// 不必逐行忽略 errcheck。
func printScaffoldPlan(w io.Writer, res *generator.Result) {
	plan := fmt.Sprintf("dry-run: %s\n  module       %s\n  shape        %s\n  capabilities %s\n  drivers      %s\n  files        %d (复制 + 渲染的上界；未减去按 import 可满足性裁剪的测试文件)\n  assets       %d\n",
		res.Dir, res.Module, res.Shape,
		strings.Join(res.Capabilities, ", "), strings.Join(res.Drivers, ", "), res.FileCount, len(res.Assets))
	_, _ = io.WriteString(w, plan)
}

var newCmd = newScaffoldCmd()

func newScaffoldCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "new <dir>",
		Short: "Scaffold a new project from selected capabilities",
		Long: `Scaffold a new project from selected capabilities.

能力集二选一：
  --profile=<name>      取该形态的清单（full/minimal/saas/enterprise/machine）
  --with=<cap>[:<drv>]  按能力名解析（硬依赖闭包 + 拓扑序），可用冒号指定驱动
                        （默认取该能力 Descriptor.Drivers 首项），如 --with=queue:kafka,user
                        能力名可取自 catalog 18 项与 7 个 Ungated 能力
                        （apidocs/storage/notification/retention/ws/grpc/encryption）

产物先写 <dir>.tmp-<rand> 再原子 rename；失败绝不留下半成品。`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var opts generator.NewOptions
			opts.Dir = args[0]
			opts.Profile, _ = cmd.Flags().GetString("profile")
			opts.With, _ = cmd.Flags().GetString("with")
			opts.Shape, _ = cmd.Flags().GetString("shape")
			opts.Module, _ = cmd.Flags().GetString("module")
			opts.NoTidy, _ = cmd.Flags().GetBool("no-tidy")
			opts.DryRun, _ = cmd.Flags().GetBool("dry-run")
			opts.Force, _ = cmd.Flags().GetBool("force")
			opts.Report, _ = cmd.Flags().GetBool("report")
			res, err := generator.NewProject(opts)
			if err != nil {
				return err
			}
			if opts.DryRun {
				printScaffoldPlan(cmd.OutOrStdout(), res)
			}
			return nil
		},
	}
	cmd.Flags().String("profile", "", "capability profile to scaffold from (mutually exclusive with --with)")
	cmd.Flags().String("with", "", "comma-separated capabilities, optionally <cap>:<driver> (mutually exclusive with --profile)")
	cmd.Flags().String("shape", "", "shape name for --with (default \"app\")")
	cmd.Flags().String("module", "", "Go module path (default: derived from <dir>)")
	cmd.Flags().Bool("no-tidy", false, "accepted (default behaviour; go mod tidy wiring lands in T8)")
	cmd.Flags().Bool("dry-run", false, "print the plan without writing anything")
	cmd.Flags().Bool("force", false, "overwrite an existing generator product (requires its .jimu-generated marker)")
	cmd.Flags().Bool("report", false, "write <dir>/docs/profiles/generated-report.md (not implemented yet)")
	return cmd
}
