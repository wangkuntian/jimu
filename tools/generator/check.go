package generator

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	"jimu/internal/capability"
	"jimu/tools/generator/frameworkmanifest"
	"jimu/tools/generator/manifest"
	"jimu/tools/generator/plan"
	"jimu/tools/generator/preflight"
	upgradeplan "jimu/tools/generator/upgrade"
)

// CheckStatus is the overall read-only lifecycle result.
type CheckStatus string

const (
	CheckStatusPassed   CheckStatus = "passed"
	CheckStatusWarnings CheckStatus = "warnings"
	CheckStatusFailed   CheckStatus = "failed"
)

// CheckSeverity separates non-blocking lifecycle warnings from blocking errors.
type CheckSeverity string

const (
	CheckSeverityWarning CheckSeverity = "warning"
	CheckSeverityError   CheckSeverity = "error"
)

// ManagedFileStatus describes one manifest-managed path.
type ManagedFileStatus string

const (
	ManagedFileClean      ManagedFileStatus = "clean"
	ManagedFileModified   ManagedFileStatus = "modified"
	ManagedFileMissing    ManagedFileStatus = "missing"
	ManagedFileNotRegular ManagedFileStatus = "not-regular"
	ManagedFileNoBaseline ManagedFileStatus = "baseline-missing"
)

// CheckDiagnostic is one deterministic lifecycle diagnostic.
type CheckDiagnostic struct {
	Severity   CheckSeverity `json:"severity"`
	Code       string        `json:"code"`
	Capability string        `json:"capability,omitempty"`
	Path       string        `json:"path,omitempty"`
	Message    string        `json:"message"`
	Fix        string        `json:"fix,omitempty"`
}

// ManagedFile reports the manifest baseline and current file state.
type ManagedFile struct {
	Path           string            `json:"path"`
	Status         ManagedFileStatus `json:"status"`
	BaselineSHA256 string            `json:"baseline_sha256,omitempty"`
	LocalSHA256    string            `json:"local_sha256,omitempty"`
}

// UpgradeCheck exposes the shared upgrade plan to lifecycle diagnostics.
type UpgradeCheck struct {
	Status                 string              `json:"status"`
	CurrentVersion         string              `json:"current_version"`
	TargetVersion          string              `json:"target_version"`
	CurrentTemplateVersion string              `json:"current_template_version"`
	TargetTemplateVersion  string              `json:"target_template_version"`
	Available              bool                `json:"available"`
	Conflicts              int                 `json:"conflicts"`
	Counts                 map[string]int      `json:"counts"`
	Entries                []upgradeplan.Entry `json:"entries"`
}

// DeepCheckStep is one serial, read-only deep validation command.
type DeepCheckStep struct {
	Name    string `json:"name"`
	Command string `json:"command"`
	Status  string `json:"status"`
	Output  string `json:"-"`
}

// DeepCheck reports configuration, build, and test validation results.
type DeepCheck struct {
	Enabled bool            `json:"enabled"`
	Passed  bool            `json:"passed"`
	Steps   []DeepCheckStep `json:"steps"`
}

// CheckOptions is the input to the lifecycle check.
type CheckOptions struct {
	Dir       string
	From      string
	Framework *manifest.Framework
	Deep      bool
}

// CheckResult is the shared human and machine-readable lifecycle result.
type CheckResult struct {
	Dir          string            `json:"-"`
	CommandDir   string            `json:"-"`
	Validity     string            `json:"validity"`
	Support      string            `json:"support"`
	Capabilities []string          `json:"capabilities"`
	Diagnostics  []CheckDiagnostic `json:"diagnostics"`
	ManagedFiles []ManagedFile     `json:"managed_files"`
	Upgrade      *UpgradeCheck     `json:"upgrade"`
	Deep         *DeepCheck        `json:"deep"`
	NextActions  []string          `json:"next_actions"`
}

// Status summarizes whether the lifecycle check has blocking errors.
func (r *CheckResult) Status() CheckStatus {
	if r.HasErrors() {
		return CheckStatusFailed
	}
	if r.HasWarnings() {
		return CheckStatusWarnings
	}
	return CheckStatusPassed
}

// HasErrors reports whether any diagnostic blocks the check.
func (r *CheckResult) HasErrors() bool {
	return slices.ContainsFunc(r.Diagnostics, func(diagnostic CheckDiagnostic) bool {
		return diagnostic.Severity == CheckSeverityError
	})
}

// HasWarnings reports whether any diagnostic is non-blocking.
func (r *CheckResult) HasWarnings() bool {
	return slices.ContainsFunc(r.Diagnostics, func(diagnostic CheckDiagnostic) bool {
		return diagnostic.Severity == CheckSeverityWarning
	})
}

// CheckProject performs the fast manifest/composition/managed-file/upgrade check
// and optionally runs deep validation in a private project copy. The user's
// project is never written.
func CheckProject(opts CheckOptions) (*CheckResult, error) {
	dir, err := filepath.Abs(opts.Dir)
	if err != nil {
		return nil, fmt.Errorf("resolve project dir: %w", err)
	}
	commandDir := opts.Dir
	if commandDir == "" {
		commandDir = "."
	}
	result := &CheckResult{
		Dir:        dir,
		CommandDir: commandDir,
		Validity:   "valid",
		Upgrade: &UpgradeCheck{
			Status:        "unavailable",
			TargetVersion: frameworkVersion(opts.Framework),
			Counts:        emptyUpgradeCounts(),
			Entries:       []upgradeplan.Entry{},
		},
		Deep: &DeepCheck{
			Enabled: opts.Deep,
			Steps:   []DeepCheckStep{},
		},
	}
	doc, err := manifest.Load(filepath.Join(dir, ".jimu", "manifest.json"))
	if err != nil {
		result.Validity = "invalid"
		result.Diagnostics = append(result.Diagnostics, CheckDiagnostic{
			Severity: CheckSeverityError,
			Code:     "manifest_invalid",
			Path:     ".jimu/manifest.json",
			Message:  sanitizeCheckText(err.Error(), dir),
			Fix:      "restore a valid manifest or regenerate the project",
		})
		result.NextActions = checkNextActions(result)
		sortCheckResult(result)
		return result, nil
	}

	result.Support = doc.Selection.Support
	result.Capabilities = slices.Clone(doc.Selection.Capabilities)
	if doc.Composition != nil {
		result.Support = doc.Composition.Support
		result.Capabilities = slices.Clone(doc.Composition.Resolved)
	}
	compositionDiagnostics := preflight.DiagnoseWithOverlay(doc, frameworkmanifest.CompositionDescriptors())
	if _, err := plan.Build(doc); err != nil {
		compositionDiagnostics = append(compositionDiagnostics, capability.Diagnostic{
			Phase:   capability.PhasePlan,
			Code:    capability.CodeInvalidPlan,
			Path:    "plan",
			Message: err.Error(),
			Fix:     "regenerate the project or fix the manifest actions",
		})
	}
	for _, diagnostic := range compositionDiagnostics {
		result.Diagnostics = append(result.Diagnostics, CheckDiagnostic{
			Severity:   CheckSeverityError,
			Code:       string(diagnostic.Code),
			Capability: diagnostic.Capability,
			Path:       diagnostic.Path,
			Message:    diagnostic.Message,
			Fix:        diagnostic.Fix,
		})
	}
	result.ManagedFiles = inspectManagedFiles(dir, doc)
	for _, file := range result.ManagedFiles {
		switch file.Status {
		case ManagedFileModified, ManagedFileNotRegular:
			result.Diagnostics = append(result.Diagnostics, CheckDiagnostic{
				Severity: CheckSeverityWarning,
				Code:     "managed_file_modified",
				Path:     file.Path,
				Message:  "managed file differs from the manifest baseline",
				Fix:      "review the local change or restore the generated baseline",
			})
		case ManagedFileMissing:
			result.Diagnostics = append(result.Diagnostics, CheckDiagnostic{
				Severity: CheckSeverityWarning,
				Code:     "managed_file_missing",
				Path:     file.Path,
				Message:  "managed file is missing",
				Fix:      "restore the generated baseline or run jimu upgrade --check --diff",
			})
		case ManagedFileNoBaseline:
			result.Diagnostics = append(result.Diagnostics, CheckDiagnostic{
				Severity: CheckSeverityWarning,
				Code:     "managed_file_baseline_missing",
				Path:     file.Path,
				Message:  "managed file has no SHA-256 baseline",
				Fix:      "regenerate the project or repair .jimu/manifest.json",
			})
		}
	}
	if !result.HasErrors() {
		upgrade, err := inspectUpgrade(dir, opts)
		if err != nil {
			result.Diagnostics = append(result.Diagnostics, CheckDiagnostic{
				Severity: CheckSeverityError,
				Code:     "upgrade_plan_failed",
				Message:  sanitizeCheckText(err.Error(), dir),
				Fix:      "verify the template source and rerun jimu check",
			})
		} else {
			result.Upgrade = upgrade
			switch upgradeplan.Status(upgrade.Status) {
			case upgradeplan.StatusConflict:
				result.Diagnostics = append(result.Diagnostics, CheckDiagnostic{
					Severity: CheckSeverityWarning,
					Code:     "upgrade_conflict",
					Message:  fmt.Sprintf("%d upgrade conflict(s) require manual resolution", upgrade.Conflicts),
					Fix:      "run jimu upgrade --check --diff and resolve the reported paths",
				})
			case upgradeplan.StatusUpgradeAvailable:
				result.Diagnostics = append(result.Diagnostics, CheckDiagnostic{
					Severity: CheckSeverityWarning,
					Code:     "upgrade_available",
					Message:  fmt.Sprintf("upgrade available: %s -> %s", upgrade.CurrentVersion, upgrade.TargetVersion),
					Fix:      "run jimu upgrade --check --diff, then upgrade from a clean branch",
				})
			}
		}
	}
	if opts.Deep {
		deep := runDeepCheck(dir)
		result.Deep = &deep
		for _, step := range deep.Steps {
			if step.Status == "failed" {
				result.Diagnostics = append(result.Diagnostics, CheckDiagnostic{
					Severity: CheckSeverityError,
					Code:     "deep_" + step.Name + "_failed",
					Message:  step.Command + " failed",
					Fix:      "inspect the captured output and fix the project before retrying",
				})
			}
		}
	}
	result.NextActions = checkNextActions(result)
	sortCheckResult(result)
	return result, nil
}

func inspectManagedFiles(root string, doc manifest.Document) []ManagedFile {
	files := make([]ManagedFile, 0, len(doc.GeneratedFiles))
	for _, rel := range slices.Clone(doc.GeneratedFiles) {
		baseline := doc.GeneratedFileSHA256[rel]
		file := ManagedFile{Path: rel, BaselineSHA256: baseline}
		path := filepath.Join(root, filepath.FromSlash(rel))
		info, err := os.Lstat(path)
		switch {
		case os.IsNotExist(err):
			file.Status = ManagedFileMissing
			files = append(files, file)
			continue
		case err != nil:
			file.Status = ManagedFileNotRegular
			files = append(files, file)
			continue
		case !info.Mode().IsRegular():
			file.Status = ManagedFileNotRegular
			files = append(files, file)
			continue
		}
		local, err := hashFile(path)
		if err != nil {
			file.Status = ManagedFileNotRegular
			files = append(files, file)
			continue
		}
		file.LocalSHA256 = local
		switch {
		case baseline == "":
			file.Status = ManagedFileNoBaseline
		case local == baseline:
			file.Status = ManagedFileClean
		default:
			file.Status = ManagedFileModified
		}
		files = append(files, file)
	}
	slices.SortFunc(files, func(a, b ManagedFile) int {
		return strings.Compare(a.Path, b.Path)
	})
	return files
}

func inspectUpgrade(dir string, opts CheckOptions) (*UpgradeCheck, error) {
	result, err := PlanUpgrade(UpgradeOptions{
		Dir: dir, From: opts.From, Framework: opts.Framework,
	})
	if err != nil {
		return nil, err
	}
	return &UpgradeCheck{
		Status:                 string(result.Report.Status()),
		CurrentVersion:         result.Previous.Framework.Version,
		TargetVersion:          result.Target.Framework.Version,
		CurrentTemplateVersion: result.Previous.Selection.TemplateVersion,
		TargetTemplateVersion:  result.Target.Selection.TemplateVersion,
		Available:              result.Report.Status() == upgradeplan.StatusUpgradeAvailable,
		Conflicts:              result.Report.Count(upgradeplan.Conflict),
		Counts: map[string]int{
			"untouched": result.Report.Count(upgradeplan.Untouched),
			"modified":  result.Report.Count(upgradeplan.Modified),
			"conflict":  result.Report.Count(upgradeplan.Conflict),
			"new":       result.Report.Count(upgradeplan.New),
			"deleted":   result.Report.Count(upgradeplan.Deleted),
		},
		Entries: result.Report.Entries,
	}, nil
}

func runDeepCheck(projectRoot string) DeepCheck {
	result := DeepCheck{Enabled: true, Steps: []DeepCheckStep{}}
	tempRoot, err := os.MkdirTemp("", "jimu-check-deep-")
	if err != nil {
		result.Steps = append(result.Steps, DeepCheckStep{
			Name: "snapshot", Command: "copy project", Status: "failed", Output: err.Error(),
		})
		return result
	}
	defer func() { _ = os.RemoveAll(tempRoot) }()
	if _, _, err := CopyTree(projectRoot, tempRoot, func(rel string, _ fs.DirEntry) bool {
		return rel != ".git" && rel != "bin"
	}); err != nil {
		result.Steps = append(result.Steps, DeepCheckStep{
			Name: "snapshot", Command: "copy project", Status: "failed", Output: err.Error(),
		})
		return result
	}
	configTest := filepath.Join(tempRoot, "internal", "config", "jimu_check_config_test.go")
	configTestSource := `package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestJimuCheckConfig(t *testing.T) {
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "configs", "app.yaml")); err == nil {
			if err := os.Chdir(dir); err != nil {
				t.Fatal(err)
			}
			break
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("project configs/app.yaml not found")
		}
		dir = parent
	}
	if _, err := Load(); err != nil {
		t.Fatal(err)
	}
}
`
	if err := os.WriteFile(configTest, []byte(configTestSource), 0o644); err != nil {
		result.Steps = append(result.Steps, DeepCheckStep{
			Name: "config", Command: "prepare config validation", Status: "failed", Output: err.Error(),
		})
	} else {
		result.Steps = append(result.Steps,
			runDeepStep(tempRoot, "config", "go", []string{"test", "./internal/config", "-run", "^TestJimuCheckConfig$", "-count=1"}),
		)
	}
	result.Steps = append(result.Steps,
		runDeepStep(tempRoot, "build", "go", []string{"build", "./..."}),
		runDeepStep(tempRoot, "test", "go", []string{"test", "./...", "-count=1"}),
	)
	result.Passed = len(result.Steps) == 3
	for _, step := range result.Steps {
		if step.Status != "passed" {
			result.Passed = false
		}
	}
	return result
}

func runDeepStep(root, name, command string, args []string) DeepCheckStep {
	ctx := context.Background()
	cmd := exec.CommandContext(ctx, command, args...)
	cmd.Dir = root
	cmd.Env = append(filteredCheckEnv(), "GOWORK=off", "JWT_SECRET="+strings.Repeat("x", 32))
	var output bytes.Buffer
	cmd.Stdout = &output
	cmd.Stderr = &output
	status := "passed"
	if err := cmd.Run(); err != nil {
		status = "failed"
		_, _ = fmt.Fprintf(&output, "\n%v", err)
	}
	text := strings.TrimSpace(sanitizeCheckText(output.String(), root))
	return DeepCheckStep{
		Name:    name,
		Command: command + " " + strings.Join(args, " "),
		Status:  status,
		Output:  text,
	}
}

func filteredCheckEnv() []string {
	blocked := map[string]bool{
		"GOWORK": true, "JWT_SECRET": true, "JWT_SECRET_FILE": true,
		"DB_HOST": true, "DB_PORT": true, "DB_USER": true, "DB_PASSWORD": true, "DB_PASSWORD_FILE": true, "DB_NAME": true,
		"REDIS_ADDR": true, "REDIS_PASSWORD": true, "REDIS_PASSWORD_FILE": true, "REDIS_DB": true,
	}
	env := make([]string, 0, len(os.Environ()))
	for _, value := range os.Environ() {
		key, _, _ := strings.Cut(value, "=")
		if !blocked[key] {
			env = append(env, value)
		}
	}
	return env
}

func checkNextActions(result *CheckResult) []string {
	dir := fmt.Sprintf("%q", result.CommandDir)
	actions := make([]string, 0, 3)
	if result.HasErrors() {
		actions = append(actions, "fix the blocking diagnostics and rerun jimu check")
	}
	if result.Upgrade != nil {
		switch upgradeplan.Status(result.Upgrade.Status) {
		case upgradeplan.StatusConflict:
			actions = append(actions, fmt.Sprintf("jimu upgrade %s --check --diff", dir))
		case upgradeplan.StatusUpgradeAvailable:
			actions = append(actions, fmt.Sprintf("jimu upgrade %s --check --diff", dir))
			actions = append(actions, fmt.Sprintf("jimu upgrade %s", dir))
		}
	}
	if result.Deep != nil && !result.Deep.Enabled {
		actions = append(actions, fmt.Sprintf("jimu check %s --deep", dir))
	}
	if len(actions) == 0 {
		actions = append(actions, "no action required")
	}
	return actions
}

func sortCheckResult(result *CheckResult) {
	sort.SliceStable(result.Diagnostics, func(i, j int) bool {
		if result.Diagnostics[i].Severity != result.Diagnostics[j].Severity {
			return result.Diagnostics[i].Severity == CheckSeverityError
		}
		if result.Diagnostics[i].Path != result.Diagnostics[j].Path {
			return result.Diagnostics[i].Path < result.Diagnostics[j].Path
		}
		if result.Diagnostics[i].Code != result.Diagnostics[j].Code {
			return result.Diagnostics[i].Code < result.Diagnostics[j].Code
		}
		return result.Diagnostics[i].Message < result.Diagnostics[j].Message
	})
	slices.Sort(result.NextActions)
	slices.Sort(result.Capabilities)
}

func emptyUpgradeCounts() map[string]int {
	return map[string]int{
		"untouched": 0,
		"modified":  0,
		"conflict":  0,
		"new":       0,
		"deleted":   0,
	}
}

func frameworkVersion(framework *manifest.Framework) string {
	if framework == nil {
		return "dev"
	}
	return framework.Version
}

func sanitizeCheckText(value, root string) string {
	value = strings.ReplaceAll(value, filepath.ToSlash(root), ".")
	value = strings.ReplaceAll(value, root, ".")
	return value
}

func hashFile(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer func() { _ = file.Close() }()
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}
	return "sha256:" + hex.EncodeToString(hash.Sum(nil)), nil
}
