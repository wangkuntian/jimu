package main

import (
	"errors"
	"fmt"
	"io"

	"jimu/tools/generator"
	upgradeplan "jimu/tools/generator/upgrade"

	"github.com/spf13/cobra"
)

var upgradeCmd = newUpgradeCmd()

var (
	errUpgradeModes = errors.New("--dry-run and --check are mutually exclusive")
)

func newUpgradeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "upgrade [dir]",
		Short:         "Explicitly upgrade a generated project to the current framework",
		SilenceUsage:  true,
		SilenceErrors: true,
		Long: `Explicitly upgrade a generated project to the current framework.

读取 <dir>/.jimu/manifest.json（默认 <dir> 为当前目录），把 manifest 中记录的
选择集按当前框架重新渲染到临时目录，然后对每个受管文件做三方比较：
manifest 基线哈希、项目当前内容、目标模板产物。

分类：
  untouched  本地与目标都等于 manifest 基线
  modified   目标模板变化，本地仍等于基线（可安全安装）
  conflict   本地内容已偏离基线（用户修改、删除或路径被占用），绝不覆盖
  new        目标模板新增且本地不存在的文件（可安装）
  deleted    目标模板不再生成的文件（保留本地副本，不自动删除）

默认执行显式升级事务：冲突在写盘前中止；受支持配置以现有段值为底合并；
变化的文件先备份再逐个安装，随后在项目里执行 tidy/build/test，任一失败即回滚。
成功后更新 .jimu/manifest.json；重复执行幂等。

--dry-run 打印完整报告；--check 与 --dry-run 相同但不接受冲突（有冲突退出非 0）。
两种模式都只读项目目录，目标产物只写入系统临时目录。--no-tidy 只跳过规划临
时副本中的 tidy；真正的升级验证始终执行 tidy/build/test。`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			dir := "."
			if len(args) > 0 {
				dir = args[0]
			}
			dryRun, _ := cmd.Flags().GetBool("dry-run")
			check, _ := cmd.Flags().GetBool("check")
			noTidy, _ := cmd.Flags().GetBool("no-tidy")
			if dryRun == check {
				if dryRun {
					return errUpgradeModes
				}
				applied, err := generator.ApplyUpgrade(generator.UpgradeOptions{
					Dir: dir, NoTidy: noTidy,
				})
				if applied != nil && (err == nil || applied.Report.HasConflicts()) {
					printUpgradeApply(cmd.OutOrStdout(), applied)
				}
				return err
			}
			result, err := generator.PlanUpgrade(generator.UpgradeOptions{
				Dir: dir, NoTidy: noTidy,
			})
			if err != nil {
				return err
			}
			mode := "dry-run"
			if check {
				mode = "check"
			}
			printUpgradeReport(cmd.OutOrStdout(), mode, result)
			if check && result.Report.HasConflicts() {
				return fmt.Errorf("%w: %d", generator.ErrUpgradeConflicts, result.Report.Count(upgradeplan.Conflict))
			}
			return nil
		},
	}
	cmd.Flags().Bool("dry-run", false, "print the full three-way upgrade report without writing anything")
	cmd.Flags().Bool("check", false, "print the report and exit non-zero when the plan has conflicts")
	cmd.Flags().Bool("no-tidy", false, "skip go mod tidy in the temporary planning copy")
	return cmd
}

func printUpgradeApply(w io.Writer, result *generator.UpgradeApplyResult) {
	if result.Report.HasConflicts() {
		printUpgradeReport(w, "upgrade", result.UpgradeResult)
		return
	}
	if len(result.Changed) == 0 && !result.ManifestUpdated {
		_, _ = fmt.Fprintf(w, "upgrade: %s\n  status       %s\n  no changes\n", result.Dir, result.Report.Status())
		return
	}
	_, _ = fmt.Fprintf(w, "upgrade: %s\n", result.Dir)
	_, _ = fmt.Fprintf(w, "  status       %s\n", result.Report.Status())
	_, _ = fmt.Fprintf(w, "  framework    %s -> %s\n",
		frameworkLabel(result.Previous.Framework.Version, result.Previous.Framework.Commit),
		frameworkLabel(result.Target.Framework.Version, result.Target.Framework.Commit))
	_, _ = fmt.Fprintf(w, "  changed      %d file(s)\n", len(result.Changed))
	_, _ = fmt.Fprintf(w, "  manifest     %s\n", updatedLabel(result.ManifestUpdated))
	_, _ = fmt.Fprintf(w, "  verified     %t\n", result.Verified)
	for _, rel := range result.Changed {
		_, _ = fmt.Fprintf(w, "  install      %s\n", rel)
	}
}

func updatedLabel(updated bool) string {
	if updated {
		return "updated"
	}
	return "unchanged"
}

func printUpgradeReport(w io.Writer, mode string, result *generator.UpgradeResult) {
	report := result.Report
	_, _ = fmt.Fprintf(w, "%s: %s\n", mode, result.Dir)
	_, _ = fmt.Fprintf(w, "  status       %s\n", report.Status())
	_, _ = fmt.Fprintf(w, "  framework    %s -> %s\n",
		frameworkLabel(result.Previous.Framework.Version, result.Previous.Framework.Commit),
		frameworkLabel(result.Target.Framework.Version, result.Target.Framework.Commit))
	_, _ = fmt.Fprintf(w, "  selection    template=%s shape=%s support=%s\n",
		selectionLabel(result.Target.Selection.Template),
		result.Target.Selection.Shape,
		result.Target.Composition.Support)
	_, _ = fmt.Fprintf(w, "  module       %s\n", result.Module)
	_, _ = fmt.Fprintf(w, "  composition  requested=%d resolved=%d auto-added=%d\n",
		len(result.Resolution.Requested), len(result.Resolution.Resolved), len(result.Resolution.AutoAdded))
	_, _ = fmt.Fprintf(w, "  summary      untouched %d, modified %d, conflict %d, new %d, deleted %d\n",
		report.Count(upgradeplan.Untouched),
		report.Count(upgradeplan.Modified),
		report.Count(upgradeplan.Conflict),
		report.Count(upgradeplan.New),
		report.Count(upgradeplan.Deleted))
	for _, entry := range report.Entries {
		_, _ = fmt.Fprintf(w, "  %-9s %s\n", entry.Category, entry.Path)
		_, _ = fmt.Fprintf(w, "            %s (action: %s)\n", entry.Reason, entry.Action)
	}
}

func frameworkLabel(version, commit string) string {
	if version == "" && commit == "" {
		return "unknown"
	}
	if commit == "" {
		return version
	}
	return fmt.Sprintf("%s@%s", version, commit)
}

func selectionLabel(template string) string {
	if template == "" {
		return "none"
	}
	return template
}

func init() {
	rootCmd.AddCommand(upgradeCmd)
}
