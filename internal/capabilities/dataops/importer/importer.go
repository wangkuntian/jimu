package importer

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"
	"time"
)

// Format represents the supported import file formats.
type Format string

const (
	FormatCSV   Format = "csv"
	FormatExcel Format = "xlsx"
)

// RowSink persists one parsed row and returns any row-specific error.
type RowSink func(ctx context.Context, row map[string]string) error

// ErrImportPersistenceNotConfigured indicates that no row persistence callback was provided.
var ErrImportPersistenceNotConfigured = errors.New("import persistence is not configured")

// Importer defines the contract for parsing and importing structured data.
type Importer interface {
	// Parse reads raw file content into a list of row maps (header → value).
	Parse(ctx context.Context, file io.Reader) ([]map[string]string, error)
	// Validate checks rows against rules without persisting anything.
	Validate(ctx context.Context, rows []map[string]string, rules ValidationRules) (*ImportResult, error)
	// Import persists rows through the configured row sink and reports per-row errors.
	Import(ctx context.Context, rows []map[string]string) (*ImportResult, error)
}

// ImportRows persists rows through the given row sink and reports per-row errors.
// 供各导入驱动复用（驱动包在各自 Format 下实现 Parse/Validate 后调用本函数）。
func ImportRows(ctx context.Context, rows []map[string]string, sink RowSink) (*ImportResult, error) {
	if sink == nil {
		return nil, ErrImportPersistenceNotConfigured
	}

	start := time.Now()
	result := NewImportResult(len(rows))
	for i, row := range rows {
		select {
		case <-ctx.Done():
			result.Finalize(start)
			return result, ctx.Err()
		default:
		}

		if err := sink(ctx, row); err != nil {
			result.AddError(i+1, "row", err.Error(), "")
		}
	}
	result.Finalize(start)
	return result, nil
}

// ImporterFactory 构造一个格式实现；驱动包在 init() 中注册。
type ImporterFactory func() Importer

// Registry 按格式管理导入器工厂。
type Registry struct {
	factories map[Format]ImporterFactory
}

// NewRegistry creates an empty registry.
func NewRegistry() *Registry {
	return &Registry{factories: make(map[Format]ImporterFactory)}
}

// Register adds the factory for the given format. 重复注册是编码错误，直接 panic
// （与 database/sql.Register 同形）。
func (r *Registry) Register(format Format, f ImporterFactory) {
	if _, dup := r.factories[format]; dup {
		panic("importer: format already registered: " + string(format))
	}
	r.factories[format] = f
}

// Get returns a new importer for the given format, or an error if unsupported.
func (r *Registry) Get(format Format) (Importer, error) {
	f, ok := r.factories[format]
	if !ok {
		return nil, fmt.Errorf("unsupported import format: %s", format)
	}
	return f(), nil
}

// defaultRegistry 是进程级注册表：驱动包在 init() 中写入，启动后只读，故不加锁。
var defaultRegistry = NewRegistry()

// Register 把格式实现注册进进程级注册表，仅供驱动包 init() 调用。
// 注意：驱动**包名**（csv/excel）与格式串（csv/xlsx，即 Format 取值）不是一回事 ——
// 一个驱动包可能承载多个格式串。
func Register(format Format, f ImporterFactory) { defaultRegistry.Register(format, f) }

// Get 从进程级注册表取格式实现；格式未编译进本构建时明确报错，不静默回退。
func Get(format Format) (Importer, error) {
	f, ok := defaultRegistry.factories[format]
	if !ok {
		return nil, fmt.Errorf("import format %q is not compiled into this build (compiled: %s)",
			format, formatsList(RegisteredFormats()))
	}
	return f(), nil
}

// Supported 报告本构建是否注册了该格式。
func Supported(format Format) bool {
	_, ok := defaultRegistry.factories[format]
	return ok
}

// RegisteredFormats 返回本构建已注册的导入格式（升序），用于 fail-closed 文案与诊断。
func RegisteredFormats() []Format {
	out := make([]Format, 0, len(defaultRegistry.factories))
	for f := range defaultRegistry.factories {
		out = append(out, f)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

// formatsList 渲染已注册格式清单；空集渲染为 none（错误文案与诊断共用）。
func formatsList(formats []Format) string {
	if len(formats) == 0 {
		return "none"
	}
	out := make([]string, len(formats))
	for i, f := range formats {
		out[i] = string(f)
	}
	return strings.Join(out, ", ")
}
