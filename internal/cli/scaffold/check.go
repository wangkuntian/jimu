package scaffold

import (
	"errors"
	"fmt"
	"io"

	"jimu/tools/generator"

	"github.com/spf13/cobra"
)

var errCompositionCheckFailed = errors.New("composition check failed")

func NewCheckCommand() *cobra.Command {
	return &cobra.Command{
		Use:           "check [dir]",
		Short:         "Check an existing generated project's manifest and composition",
		SilenceUsage:  true,
		SilenceErrors: true,
		Long: `Check an existing generated project's manifest and composition.

读取 <dir>/.jimu/manifest.json（默认 <dir> 为当前目录），按与 jimu new 相同的
组合规则检查硬依赖、冲突、端口、驱动、迁移 schema 依赖、表/迁移、运行时服务/
配置、受保护中间件与生成计划。命令只读，不写任何文件；合法项目退出 0，
非法项目退出非 0 并逐条输出 code/capability/path/message/fix。`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			dir := "."
			if len(args) > 0 {
				dir = args[0]
			}
			result, err := generator.CheckProject(generator.CheckOptions{Dir: dir})
			if err != nil {
				return err
			}
			if len(result.Diagnostics) > 0 {
				printCheckDiagnostics(cmd.OutOrStdout(), result)
				return errCompositionCheckFailed
			}
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "check passed: %s (valid, support=%s, capabilities=%d)\n",
				result.Dir, result.Support, len(result.Capabilities))
			return nil
		},
	}
}

func printCheckDiagnostics(w io.Writer, result *generator.CheckResult) {
	_, _ = fmt.Fprintf(w, "check failed: %s (invalid, support=%s, capabilities=%d)\n",
		result.Dir, result.Support, len(result.Capabilities))
	for _, diagnostic := range result.Diagnostics {
		_, _ = fmt.Fprintf(w, "  [%s]", diagnostic.Code)
		if diagnostic.Capability != "" {
			_, _ = fmt.Fprintf(w, " capability %q", diagnostic.Capability)
		}
		if diagnostic.Path != "" {
			_, _ = fmt.Fprintf(w, " at %s", diagnostic.Path)
		}
		_, _ = fmt.Fprintf(w, ": %s", diagnostic.Message)
		if diagnostic.Fix != "" {
			_, _ = fmt.Fprintf(w, " (fix: %s)", diagnostic.Fix)
		}
		_, _ = fmt.Fprintln(w)
	}
}
