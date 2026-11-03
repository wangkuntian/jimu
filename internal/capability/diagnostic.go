package capability

import (
	"fmt"
	"slices"
	"strings"

	"jimu/internal/contract"
)

// Phase 是诊断所属的解析阶段。解析顺序固定为 Phase 从小到大；同一阶段内再按
// capability/path/code/message 排序，保证输出稳定。
type Phase int

const (
	// PhaseInput 校验请求本身：未知、重复或空能力名。
	PhaseInput Phase = iota + 1
	// PhaseClosure 补齐并校验硬依赖闭包。
	PhaseClosure
	// PhaseConflict 校验能力互斥声明。
	PhaseConflict
	// PhasePorts 校验 contract 端口的提供方/必需消费者。
	PhasePorts
	// PhaseDrivers 校验驱动选择。
	PhaseDrivers
	// PhaseMigrationSchema 校验迁移 schema 依赖的携带。
	PhaseMigrationSchema
	// PhaseTables 校验表所有权与迁移文件。
	PhaseTables
	// PhaseRuntimeConfig 校验运行时外部服务与配置段。
	PhaseRuntimeConfig
	// PhaseMiddleware 校验受保护路由的中间件提供者。
	PhaseMiddleware
	// PhasePlan 校验生成计划本身。
	PhasePlan
)

// DiagnosticCode 是稳定的机器可读错误码。
type DiagnosticCode string

const (
	CodeUnknownCapability                DiagnosticCode = "unknown_capability"
	CodeDuplicateCapability              DiagnosticCode = "duplicate_capability"
	CodeMissingDependency                DiagnosticCode = "missing_dependency"
	CodeConflict                         DiagnosticCode = "conflict"
	CodeDuplicateProvider                DiagnosticCode = "duplicate_provider"
	CodeMissingProvider                  DiagnosticCode = "missing_provider"
	CodeUnknownDriver                    DiagnosticCode = "unknown_driver"
	CodeMissingDriver                    DiagnosticCode = "missing_driver"
	CodeMissingMigrationSchemaDependency DiagnosticCode = "missing_migration_schema_dependency"
	CodeDuplicateTableOwner              DiagnosticCode = "duplicate_table_owner"
	CodeMissingMigration                 DiagnosticCode = "missing_migration"
	CodeDuplicateMigration               DiagnosticCode = "duplicate_migration"
	CodeUnknownRuntimeService            DiagnosticCode = "unknown_runtime_service"
	CodeMissingRuntimeServiceConfig      DiagnosticCode = "missing_runtime_service_config"
	CodeMiddlewareProviderMissing        DiagnosticCode = "middleware_provider_missing"
	CodeMiddlewareConflict               DiagnosticCode = "middleware_conflict"
	CodeInvalidPlan                      DiagnosticCode = "invalid_plan"
)

// Diagnostic 是一条结构化组合诊断。Path 指向 manifest 或 selection 中的字段，
// Fix 给出可执行的修复建议。
type Diagnostic struct {
	Code       DiagnosticCode `json:"code"`
	Capability string         `json:"capability,omitempty"`
	Path       string         `json:"path,omitempty"`
	Message    string         `json:"message"`
	Fix        string         `json:"fix,omitempty"`

	// Phase 只用于稳定排序，不进入 JSON。
	Phase Phase `json:"-"`
}

// Error 把一组诊断包装成 error。Error() 逐行输出 code、路径与修复建议。
type Error struct {
	Diagnostics []Diagnostic
}

func (e *Error) Error() string {
	if e == nil || len(e.Diagnostics) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("composition invalid")
	for _, d := range e.Diagnostics {
		fmt.Fprintf(&b, "\n  [%s]", d.Code)
		if d.Capability != "" {
			fmt.Fprintf(&b, " capability %q", d.Capability)
		}
		if d.Path != "" {
			fmt.Fprintf(&b, " at %s", d.Path)
		}
		fmt.Fprintf(&b, ": %s", d.Message)
		if d.Fix != "" {
			fmt.Fprintf(&b, " (fix: %s)", d.Fix)
		}
	}
	return b.String()
}

// SortDiagnostics 按固定的解析阶段与稳定键排序。
func SortDiagnostics(values []Diagnostic) {
	slices.SortStableFunc(values, func(a, b Diagnostic) int {
		if a.Phase != b.Phase {
			return int(a.Phase) - int(b.Phase)
		}
		if a.Capability != b.Capability {
			return strings.Compare(a.Capability, b.Capability)
		}
		if a.Path != b.Path {
			return strings.Compare(a.Path, b.Path)
		}
		if a.Code != b.Code {
			return strings.Compare(string(a.Code), string(b.Code))
		}
		return strings.Compare(a.Message, b.Message)
	})
}

// DiagnosticsError 在诊断非空时返回 error。
func DiagnosticsError(values []Diagnostic) error {
	if len(values) == 0 {
		return nil
	}
	SortDiagnostics(values)
	return &Error{Diagnostics: values}
}

// Validity 根据诊断计算合法性。
func Validity(values []Diagnostic) contract.Validity {
	if len(values) == 0 {
		return contract.ValidityValid
	}
	return contract.ValidityInvalid
}
