package scaffold

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"jimu/internal/templatesource"
	"jimu/tools/generator"

	"github.com/spf13/cobra"
)

var (
	errScaffoldSelectors        = errors.New("--template, --profile and --with are mutually exclusive")
	errFrameworkInternalProfile = errors.New("is framework-internal; use --template or --with")
	errMinimalProfileRemoved    = errors.New("is framework-internal; use --template=minimal")
	errShapeRequiresWith        = errors.New("--shape is only valid with --with")
)

func printScaffoldPlan(w io.Writer, res *generator.Result) {
	plan := fmt.Sprintf("dry-run: %s\n  module       %s\n  template     %s\n  support      %s\n  shape        %s\n  requested    %s\n  resolved     %s\n  auto-added   %s\n  capabilities %s (装配集)\n  copy set     %s (复制集：装配集 + 编译闭包 + 迁移携带)\n  drivers      %s\n  services     %s\n  migrations   %s\n  status       %s\n  files        %d (复制 + 渲染的上界；未减去按 import/资产/组成可满足性裁剪的测试文件)\n  assets       %d\n",
		res.Dir, res.Module, res.Template, res.Support, res.Shape,
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
  --template=<name>     官方预设：minimal（默认）、auth 或 full
  --with=<cap>[:<drv>]  专家模式（experimental），按能力名解析硬依赖与驱动

预设是裁剪起点：minimal 只含最小内核，auth 额外带用户/权限/认证，full 是全部能力。
生成后仍可用 capability add 继续组合能力。--profile 只用于拒绝框架内部形态。

生成顺序：复制/渲染 → go mod tidy → 自检（生成项目内 go build ./... 与
生成项目测试都必须绿）→ 原子换上 <dir>。产物先写 <dir>.tmp-<rand>，
任一步失败即整体回滚，绝不留下半成品；--no-tidy 只跳过 tidy，自检仍会跑。`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var opts generator.NewOptions
			opts.Dir = args[0]
			opts.Template, _ = cmd.Flags().GetString("template")
			opts.Profile, _ = cmd.Flags().GetString("profile")
			opts.With, _ = cmd.Flags().GetString("with")
			opts.Shape, _ = cmd.Flags().GetString("shape")
			opts.Module, _ = cmd.Flags().GetString("module")
			opts.DBDriver, _ = cmd.Flags().GetString("db")
			opts.NoTidy, _ = cmd.Flags().GetBool("no-tidy")
			opts.DryRun, _ = cmd.Flags().GetBool("dry-run")
			opts.Force, _ = cmd.Flags().GetBool("force")
			opts.Report, _ = cmd.Flags().GetBool("report")
			if opts.Template != "" && (opts.Profile != "" || opts.With != "") {
				return errScaffoldSelectors
			}
			if opts.Profile != "" {
				if opts.With != "" {
					return errScaffoldSelectors
				}
				if opts.Profile == "minimal" {
					return fmt.Errorf("profile %q %w", opts.Profile, errMinimalProfileRemoved)
				}
				return fmt.Errorf("profile %q %w", opts.Profile, errFrameworkInternalProfile)
			}
			if cmd.Flags().Changed("shape") && opts.Template != "" {
				return errShapeRequiresWith
			}
			source, cleanup, err := resolveSource(cmd, resolve)
			if err != nil {
				return err
			}
			defer func() { _ = cleanup() }()
			opts.SourceRoot = source.Root
			opts.Framework = source.Framework
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
	cmd.Flags().String("template", "", "official preset: minimal (default), auth or full (mutually exclusive with --profile/--with)")
	cmd.Flags().String("profile", "", "framework-internal profile selector; rejected by the CLI (presets are selected with --template)")
	cmd.Flags().String("with", "", "comma-separated capabilities, optionally <cap>:<driver> (mutually exclusive with --profile)")
	cmd.Flags().String("shape", "", "shape name for --with (default \"app\")")
	cmd.Flags().String("module", "", "Go module path (default: derived from <dir>)")
	cmd.Flags().String("db", "mysql", "database driver compiled into the project: mysql or postgres")
	cmd.Flags().Bool("no-tidy", false, "skip go mod tidy after generation (default: tidy runs and a failing tidy aborts)")
	cmd.Flags().Bool("dry-run", false, "print the plan without writing anything")
	cmd.Flags().Bool("force", false, "overwrite an existing generated project (requires .jimu/manifest.json)")
	cmd.Flags().Bool("report", false, "write <dir>/docs/profiles/generated-report.md (file count / code lines / direct deps / migrations / tables / routes)")
	addSourceFlag(cmd)
	return cmd
}
