package scaffold

import (
	"fmt"
	"io"
	"strings"

	"jimu/internal/cli/actions"
	"jimu/internal/cli/tui"
	"jimu/internal/templatesource"
	"jimu/tools/generator"

	"github.com/spf13/cobra"
)

func printScaffoldPlan(w io.Writer, res *generator.Result) {
	plan := fmt.Sprintf("dry-run: %s\n  module       %s\n  profile      %s\n  support      %s\n  shape        %s\n  requested    %s\n  resolved     %s\n  auto-added   %s\n  capabilities %s (装配集)\n  copy set     %s (复制集：装配集 + 编译闭包 + 迁移携带)\n  drivers      %s\n  services     %s\n  migrations   %s\n  status       %s\n  files        %d (复制 + 渲染的上界；未减去按 import/资产/组成可满足性裁剪的测试文件)\n  assets       %d\n",
		res.Dir, res.Module, selectionLabel(res.Template), res.Support, res.Shape,
		joinOrNone(res.Requested), joinOrNone(res.Resolved), joinOrNone(res.AutoAdded),
		strings.Join(res.Capabilities, ", "), strings.Join(res.CopySet, ", "),
		joinOrNone(res.Drivers), joinOrNone(res.Services), joinOrNone(res.Migrations),
		res.Validity+"/"+res.Support, res.FileCount, len(res.Assets))
	_, _ = io.WriteString(w, plan)
}

func joinOrNone(values []string) string {
	if len(values) == 0 {
		return "none"
	}
	return strings.Join(values, ", ")
}

func NewCommand(resolve templatesource.Resolver) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "new <dir>",
		Short: "Scaffold a new project from selected capabilities",
		Long: `Scaffold a new project from selected capabilities.

选择器（互斥）：
  --profile=<name>      官方预设：minimal（默认）、auth 或 full
  --with=<cap>[:<drv>]  专家模式（experimental），按能力名解析硬依赖与驱动

预设是裁剪起点：minimal 只含最小内核，auth 额外带用户/权限/认证，full 是全部能力。
生成后仍可用 capability add 继续组合能力。

生成顺序：复制/渲染 → go mod tidy → 自检（生成项目内 go build ./... 与
生成项目测试都必须绿）→ 原子换上 <dir>。产物先写 <dir>.tmp-<rand>，
任一步失败即整体回滚，绝不留下半成品；--no-tidy 只跳过 tidy，自检仍会跑。`,
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 && tuiAvailable(cmd) {
				return nil
			}
			return cobra.ExactArgs(1)(cmd, args)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return runTUI(cmd, tui.ScreenNew)
			}
			source, cleanup, err := resolveSource(cmd, resolve)
			if err != nil {
				return err
			}
			defer func() { _ = cleanup() }()
			input := actions.NewInput{
				Dir:        args[0],
				Profile:    flagString(cmd, "profile"),
				With:       flagString(cmd, "with"),
				Shape:      flagString(cmd, "shape"),
				Module:     flagString(cmd, "module"),
				DBDriver:   flagString(cmd, "db"),
				Tests:      flagString(cmd, "tests"),
				Tools:      flagString(cmd, "tools"),
				Deploy:     flagString(cmd, "deploy"),
				CLIUI:      flagString(cmd, "cli"),
				Runtime:    flagString(cmd, "runtime"),
				NoTidy:     flagBool(cmd, "no-tidy"),
				DryRun:     flagBool(cmd, "dry-run"),
				Force:      flagBool(cmd, "force"),
				Report:     flagBool(cmd, "report"),
				SourceRoot: source.Root,
				Framework:  source.Framework,
			}
			res, err := actions.RunNew(cmd.Context(), input)
			if err != nil {
				return err
			}
			if input.DryRun {
				printScaffoldPlan(cmd.OutOrStdout(), res)
			}
			return nil
		},
	}
	cmd.Flags().String("profile", "", "official preset: minimal (default), auth or full (mutually exclusive with --with)")
	cmd.Flags().String("with", "", "comma-separated capabilities, optionally <cap>:<driver> (mutually exclusive with --profile)")
	cmd.Flags().String("shape", "", "shape name for --with (default \"app\")")
	cmd.Flags().String("module", "", "Go module path (default: derived from <dir>)")
	cmd.Flags().String("db", "mysql", "database driver compiled into the project: mysql or postgres")
	cmd.Flags().String("tests", "", "test pruning: selected, all, or none (default: selected)")
	cmd.Flags().String("tools", "", "generated tools: auto, all, or none (default: auto)")
	cmd.Flags().String("deploy", "", "deploy components: compose,k8s,helm,observability, all, or none")
	cmd.Flags().String("cli", "", "project CLI UI: both or direct (default: profile-derived)")
	cmd.Flags().String("runtime", "", "kernel runtime: lean or standard (default: lean for minimal, standard otherwise)")
	cmd.Flags().Bool("no-tidy", false, "skip go mod tidy after generation (default: tidy runs and a failing tidy aborts)")
	cmd.Flags().Bool("dry-run", false, "print the plan without writing anything")
	cmd.Flags().Bool("force", false, "overwrite an existing generated project (requires .jimu/manifest.json)")
	cmd.Flags().Bool("report", false, "write <dir>/docs/profiles/generated-report.md (file count / code lines / direct deps / migrations / tables / routes)")
	addSourceFlag(cmd)
	return cmd
}
