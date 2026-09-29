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
//
// 两个能力集都要打印并显式标注（它们不相等）：`capabilities` = **装配集**（会进 assembly、
// 挂路由），`copy set` = **复制集**（额外含编译闭包/迁移携带/内核编译期 domain 依赖的目录）。
func printScaffoldPlan(w io.Writer, res *generator.Result) {
	plan := fmt.Sprintf("dry-run: %s\n  module       %s\n  shape        %s\n  capabilities %s (装配集)\n  copy set     %s (复制集：装配集 + 编译闭包 + 迁移携带)\n  drivers      %s\n  files        %d (复制 + 渲染的上界；未减去按 import/资产/组成可满足性裁剪的测试文件)\n  assets       %d\n",
		res.Dir, res.Module, res.Shape,
		strings.Join(res.Capabilities, ", "), strings.Join(res.CopySet, ", "),
		strings.Join(res.Drivers, ", "), res.FileCount, len(res.Assets))
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

生成顺序：复制/渲染 → go mod tidy → 自检（生成项目内 go build ./... 与
go run ./tools/checkcapabilities 都必须绿）→ 原子换上 <dir>。产物先写 <dir>.tmp-<rand>，
任一步失败即整体回滚，绝不留下半成品；--no-tidy 只跳过 tidy，自检仍会跑。`,
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
	cmd.Flags().Bool("no-tidy", false, "skip go mod tidy after generation (default: tidy runs and a failing tidy aborts)")
	cmd.Flags().Bool("dry-run", false, "print the plan without writing anything")
	cmd.Flags().Bool("force", false, "overwrite an existing generator product (requires its .jimu-generated marker)")
	cmd.Flags().Bool("report", false, "write <dir>/docs/profiles/generated-report.md (file count / code lines / direct deps / migrations / tables / routes)")
	return cmd
}
