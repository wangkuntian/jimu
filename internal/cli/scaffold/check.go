package scaffold

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"jimu/internal/templatesource"
	"jimu/tools/generator"

	"github.com/spf13/cobra"
)

var errCompositionCheckFailed = errors.New("check failed")

// NewCheckCommand builds the lifecycle check command. It shares the upgrade
// planner so fast diagnostics, deep checks, human output, and JSON report the
// same facts.
func NewCheckCommand(resolve templatesource.Resolver) *cobra.Command {
	cmd := &cobra.Command{
		Use:           "check [dir]",
		Short:         "Check a generated project's lifecycle and upgrade state",
		SilenceUsage:  true,
		SilenceErrors: true,
		Long: `Check a generated project without modifying it.

默认 fast 模式检查 manifest schema/digest、能力组合、受管文件漂移、当前与目标
框架版本、可用升级、待处理 conflict 和下一步命令。受管文件漂移、可用升级或
conflict 只产生 warning；非法 manifest/组合产生非零退出。

--deep 在 fast 检查后用项目私有临时副本串行执行配置校验、go build ./... 和
go test ./...，不运行 Make 目标或业务脚本。检查过程不会更新 manifest、文件或
配置。--json 输出 schema version 1，并复用同一份检查事实。`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			dir := "."
			if len(args) > 0 {
				dir = args[0]
			}
			deep, _ := cmd.Flags().GetBool("deep")
			jsonOutput, _ := cmd.Flags().GetBool("json")
			source, cleanup, err := resolveSource(cmd, resolve)
			if err != nil {
				return err
			}
			defer func() { _ = cleanup() }()
			result, err := generator.CheckProject(generator.CheckOptions{
				Dir: dir, From: source.Root, Framework: source.Framework, Deep: deep,
			})
			if err != nil {
				return err
			}
			if jsonOutput {
				if err := printCheckJSON(cmd.OutOrStdout(), result); err != nil {
					return err
				}
			} else {
				printCheckResult(cmd.OutOrStdout(), result)
			}
			if result.HasErrors() {
				return checkFailureError(result)
			}
			return nil
		},
	}
	cmd.Flags().Bool("deep", false, "run config validation, go build ./..., and go test ./... in a temporary project copy")
	cmd.Flags().Bool("json", false, "print deterministic machine-readable lifecycle results")
	addSourceFlag(cmd)
	return cmd
}

func checkFailureError(result *generator.CheckResult) error {
	for _, diagnostic := range result.Diagnostics {
		if diagnostic.Severity != generator.CheckSeverityError {
			continue
		}
		if diagnostic.Path != "" {
			return fmt.Errorf("%w: %s at %s: %s", errCompositionCheckFailed, diagnostic.Code, diagnostic.Path, diagnostic.Message)
		}
		return fmt.Errorf("%w: %s: %s", errCompositionCheckFailed, diagnostic.Code, diagnostic.Message)
	}
	return errCompositionCheckFailed
}

func printCheckResult(w io.Writer, result *generator.CheckResult) {
	switch result.Status() {
	case generator.CheckStatusPassed:
		_, _ = fmt.Fprintf(w, "check passed: %s (valid, support=%s, capabilities=%d)\n",
			result.Dir, result.Support, len(result.Capabilities))
	case generator.CheckStatusWarnings:
		_, _ = fmt.Fprintf(w, "check passed with warnings: %s (valid, support=%s, capabilities=%d)\n",
			result.Dir, result.Support, len(result.Capabilities))
	default:
		_, _ = fmt.Fprintf(w, "check failed: %s (invalid, support=%s, capabilities=%d)\n",
			result.Dir, result.Support, len(result.Capabilities))
	}
	if result.Upgrade != nil {
		_, _ = fmt.Fprintf(w, "  framework    %s -> %s\n",
			checkVersionLabel(result.Upgrade.CurrentVersion), checkVersionLabel(result.Upgrade.TargetVersion))
		_, _ = fmt.Fprintf(w, "  upgrade      %s\n", result.Upgrade.Status)
	}
	_, _ = fmt.Fprintf(w, "  managed      %d file(s)\n", len(result.ManagedFiles))
	for _, diagnostic := range result.Diagnostics {
		_, _ = fmt.Fprintf(w, "  [%s/%s]", diagnostic.Severity, diagnostic.Code)
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
	if result.Deep != nil && result.Deep.Enabled {
		_, _ = fmt.Fprintf(w, "  deep         %t\n", result.Deep.Passed)
		for _, step := range result.Deep.Steps {
			_, _ = fmt.Fprintf(w, "    %-8s %s\n", step.Status, step.Command)
		}
	}
	for _, action := range result.NextActions {
		_, _ = fmt.Fprintf(w, "  next         %s\n", action)
	}
}

func printCheckJSON(w io.Writer, result *generator.CheckResult) error {
	document := struct {
		SchemaVersion int                         `json:"schema_version"`
		Status        generator.CheckStatus       `json:"status"`
		Validity      string                      `json:"validity"`
		Support       string                      `json:"support"`
		Capabilities  []string                    `json:"capabilities"`
		Summary       map[string]int              `json:"summary"`
		Diagnostics   []generator.CheckDiagnostic `json:"diagnostics"`
		ManagedFiles  []generator.ManagedFile     `json:"managed_files"`
		Upgrade       *generator.UpgradeCheck     `json:"upgrade"`
		Deep          *generator.DeepCheck        `json:"deep"`
		NextActions   []string                    `json:"next_actions"`
	}{
		SchemaVersion: 1,
		Status:        result.Status(),
		Validity:      result.Validity,
		Support:       result.Support,
		Capabilities:  result.Capabilities,
		Summary: map[string]int{
			"errors":            checkDiagnosticCount(result.Diagnostics, generator.CheckSeverityError),
			"warnings":          checkDiagnosticCount(result.Diagnostics, generator.CheckSeverityWarning),
			"managed_files":     len(result.ManagedFiles),
			"upgrade_available": boolInt(result.Upgrade != nil && result.Upgrade.Available),
			"conflicts":         upgradeConflictCount(result.Upgrade),
		},
		Diagnostics:  result.Diagnostics,
		ManagedFiles: result.ManagedFiles,
		Upgrade:      result.Upgrade,
		Deep:         result.Deep,
		NextActions:  result.NextActions,
	}
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(document); err != nil {
		return fmt.Errorf("encode check report: %w", err)
	}
	return nil
}

func checkDiagnosticCount(diagnostics []generator.CheckDiagnostic, severity generator.CheckSeverity) int {
	count := 0
	for _, diagnostic := range diagnostics {
		if diagnostic.Severity == severity {
			count++
		}
	}
	return count
}

func upgradeConflictCount(upgrade *generator.UpgradeCheck) int {
	if upgrade == nil {
		return 0
	}
	return upgrade.Conflicts
}

func boolInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

func checkVersionLabel(version string) string {
	if version == "" {
		return "unknown"
	}
	return version
}
