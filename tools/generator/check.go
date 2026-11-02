package generator

import (
	"fmt"
	"path/filepath"
	"slices"

	"jimu/internal/capability"
	"jimu/tools/generator/frameworkmanifest"
	"jimu/tools/generator/manifest"
	"jimu/tools/generator/plan"
	"jimu/tools/generator/preflight"
)

// CheckOptions 是 `jimu check` 的输入。
type CheckOptions struct {
	Dir string
}

// CheckResult 是一次只读项目检查的结果。
type CheckResult struct {
	Dir          string
	Validity     string
	Support      string
	Capabilities []string
	Diagnostics  []capability.Diagnostic
}

// CheckProject 读取已有项目的 .jimu/manifest.json，按与生成前相同的组合规则检查
// manifest 与组合一致性。它不写任何文件。
func CheckProject(opts CheckOptions) (*CheckResult, error) {
	dir := opts.Dir
	if dir == "" {
		dir = "."
	}
	absolute, err := filepath.Abs(dir)
	if err != nil {
		return nil, fmt.Errorf("resolve project dir: %w", err)
	}
	doc, err := manifest.Load(filepath.Join(absolute, ".jimu", "manifest.json"))
	if err != nil {
		return nil, err
	}
	diagnostics := preflight.DiagnoseWithOverlay(doc, frameworkmanifest.CompositionDescriptors())
	if _, err := plan.Build(doc); err != nil {
		diagnostics = append(diagnostics, capability.Diagnostic{
			Phase:   capability.PhasePlan,
			Code:    capability.CodeInvalidPlan,
			Path:    "plan",
			Message: err.Error(),
			Fix:     "regenerate the project or fix the manifest actions",
		})
	}
	capability.SortDiagnostics(diagnostics)
	validity := string(capability.Validity(diagnostics))
	support := doc.Selection.Support
	capabilities := slices.Clone(doc.Selection.Capabilities)
	if doc.Composition != nil {
		support = doc.Composition.Support
		capabilities = slices.Clone(doc.Composition.Resolved)
	}
	return &CheckResult{
		Dir:          absolute,
		Validity:     validity,
		Support:      support,
		Capabilities: capabilities,
		Diagnostics:  diagnostics,
	}, nil
}
