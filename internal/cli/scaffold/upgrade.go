package scaffold

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"jimu/internal/templatesource"
	"jimu/tools/generator"
	upgradeplan "jimu/tools/generator/upgrade"

	"github.com/spf13/cobra"
)

var errUpgradeModes = errors.New("--dry-run and --check are mutually exclusive")
var errUpgradeDiagnosticModes = errors.New("--json and --diff require --dry-run or --check")

func NewUpgradeCommand(resolve templatesource.Resolver) *cobra.Command {
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
			jsonOutput, _ := cmd.Flags().GetBool("json")
			diffOutput, _ := cmd.Flags().GetBool("diff")
			if dryRun && check {
				return errUpgradeModes
			}
			if (jsonOutput || diffOutput) && !dryRun && !check {
				return errUpgradeDiagnosticModes
			}
			if dryRun == check {
				source, cleanup, err := resolveSource(cmd, resolve)
				if err != nil {
					return err
				}
				defer func() { _ = cleanup() }()
				applied, err := generator.ApplyUpgrade(generator.UpgradeOptions{
					Dir: dir, From: source.Root, Framework: source.Framework, NoTidy: noTidy,
				})
				if applied != nil && (err == nil || applied.Report.HasConflicts()) {
					printUpgradeApply(cmd.OutOrStdout(), applied)
				}
				return err
			}
			source, cleanup, err := resolveSource(cmd, resolve)
			if err != nil {
				return err
			}
			defer func() { _ = cleanup() }()
			result, err := generator.PlanUpgrade(generator.UpgradeOptions{
				Dir: dir, From: source.Root, Framework: source.Framework, NoTidy: noTidy,
				IncludeDiff: diffOutput,
			})
			if err != nil {
				return err
			}
			mode := "dry-run"
			if check {
				mode = "check"
			}
			if jsonOutput {
				if err := printUpgradeJSON(cmd.OutOrStdout(), mode, result); err != nil {
					return err
				}
			} else {
				printUpgradeReport(cmd.OutOrStdout(), mode, result, diffOutput)
			}
			if check && result.Report.HasConflicts() {
				return fmt.Errorf("%w: %d", generator.ErrUpgradeConflicts, result.Report.Count(upgradeplan.Conflict))
			}
			return nil
		},
	}
	cmd.Flags().Bool("dry-run", false, "print the full three-way upgrade report without writing anything")
	cmd.Flags().Bool("check", false, "print the report and exit non-zero when the plan has conflicts")
	cmd.Flags().Bool("json", false, "print a deterministic machine-readable report (requires --dry-run or --check)")
	cmd.Flags().Bool("diff", false, "include local-to-target text diffs for changed files")
	cmd.Flags().Bool("no-tidy", false, "skip go mod tidy in the temporary planning copy")
	addSourceFlag(cmd)
	return cmd
}

func printUpgradeApply(w io.Writer, result *generator.UpgradeApplyResult) {
	if result.Report.HasConflicts() {
		printUpgradeReport(w, "upgrade", result.UpgradeResult, false)
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

func printUpgradeReport(w io.Writer, mode string, result *generator.UpgradeResult, includeDiff bool) {
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
		if includeDiff {
			if entry.Diff != "" {
				_, _ = fmt.Fprint(w, entry.Diff)
			} else if entry.DiffUnavailableReason != "" {
				_, _ = fmt.Fprintf(w, "            diff unavailable: %s\n", entry.DiffUnavailableReason)
			}
		}
	}
}

type upgradeJSONReport struct {
	SchemaVersion string               `json:"schema_version"`
	Mode          string               `json:"mode"`
	Status        upgradeplan.Status   `json:"status"`
	Framework     upgradeJSONFramework `json:"framework"`
	Selection     upgradeJSONSelection `json:"selection"`
	Counts        map[string]int       `json:"counts"`
	Entries       []upgradeplan.Entry  `json:"entries"`
	NextActions   []string             `json:"next_actions"`
}

type upgradeJSONFramework struct {
	From upgradeJSONVersion `json:"from"`
	To   upgradeJSONVersion `json:"to"`
}

type upgradeJSONVersion struct {
	Version string `json:"version"`
	Commit  string `json:"commit"`
}

type upgradeJSONSelection struct {
	Template string `json:"template"`
	Shape    string `json:"shape"`
	Support  string `json:"support"`
}

func printUpgradeJSON(w io.Writer, mode string, result *generator.UpgradeResult) error {
	support := ""
	if result.Target.Composition != nil {
		support = result.Target.Composition.Support
	}
	document := upgradeJSONReport{
		SchemaVersion: "v1",
		Mode:          mode,
		Status:        result.Report.Status(),
		Framework: upgradeJSONFramework{
			From: upgradeJSONVersion{
				Version: result.Previous.Framework.Version,
				Commit:  result.Previous.Framework.Commit,
			},
			To: upgradeJSONVersion{
				Version: result.Target.Framework.Version,
				Commit:  result.Target.Framework.Commit,
			},
		},
		Selection: upgradeJSONSelection{
			Template: result.Target.Selection.Template,
			Shape:    result.Target.Selection.Shape,
			Support:  support,
		},
		Counts: map[string]int{
			"untouched": result.Report.Count(upgradeplan.Untouched),
			"modified":  result.Report.Count(upgradeplan.Modified),
			"conflict":  result.Report.Count(upgradeplan.Conflict),
			"new":       result.Report.Count(upgradeplan.New),
			"deleted":   result.Report.Count(upgradeplan.Deleted),
		},
		Entries: result.Report.Entries,
		NextActions: []string{
			upgradeNextAction(result),
		},
	}
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(document); err != nil {
		return fmt.Errorf("encode upgrade report: %w", err)
	}
	return nil
}

func upgradeNextAction(result *generator.UpgradeResult) string {
	switch result.Report.Status() {
	case upgradeplan.StatusConflict:
		return "resolve conflicts, then rerun jimu upgrade --check"
	case upgradeplan.StatusUpgradeAvailable:
		return "review the upgrade plan, then run jimu upgrade"
	default:
		return "no action required"
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
