// Package exporter 提供批量数据导出能力，与 importer 反向对称：
// 表头 + 行数据（header → value）渲染为 CSV 或 Excel 文件。
//
// 导出侧目前没有生产消费方（端点另行接入）；本包的注册表与 importer 保持对称，
// 供形态测试钉住编译面、并为后续端点预留查表入口。
package exporter

import (
	"context"
	"fmt"
	"io"
	"sort"
	"strings"
)

// Format 支持的导出文件格式
type Format string

const (
	FormatCSV   Format = "csv"
	FormatExcel Format = "xlsx"
)

// Exporter 定义将表头与行数据渲染为指定格式写入 writer 的契约。
type Exporter interface {
	// Export 将 header 与 rows 渲染为文件写入 w；行内缺失的键补空串。
	Export(ctx context.Context, header []string, rows []map[string]string, w io.Writer) error
}

// ExporterFactory 构造一个格式实现；驱动包在 init() 中注册。
type ExporterFactory func() Exporter

// Registry 按格式管理导出器工厂。
type Registry struct {
	factories map[Format]ExporterFactory
}

// NewRegistry 创建空注册表。
func NewRegistry() *Registry {
	return &Registry{factories: make(map[Format]ExporterFactory)}
}

// Register 注册指定格式的导出器工厂；重复注册是编码错误，直接 panic
// （与 database/sql.Register 同形）。
func (r *Registry) Register(format Format, f ExporterFactory) {
	if _, dup := r.factories[format]; dup {
		panic("exporter: format already registered: " + string(format))
	}
	r.factories[format] = f
}

// Get 返回指定格式的导出器（每次构造新实例），不支持时返回错误。
func (r *Registry) Get(format Format) (Exporter, error) {
	f, ok := r.factories[format]
	if !ok {
		return nil, fmt.Errorf("unsupported export format: %s", format)
	}
	return f(), nil
}

// defaultRegistry 是进程级注册表：驱动包在 init() 中写入，启动后只读，故不加锁。
var defaultRegistry = NewRegistry()

// Register 把格式实现注册进进程级注册表，仅供驱动包 init() 调用。
// 注意：驱动**包名**（csv/excel）与格式串（csv/xlsx，即 Format 取值）不是一回事 ——
// 一个驱动包可能承载多个格式串。
func Register(format Format, f ExporterFactory) { defaultRegistry.Register(format, f) }

// Get 从进程级注册表取格式实现；格式未编译进本构建时明确报错，不静默回退。
func Get(format Format) (Exporter, error) {
	f, ok := defaultRegistry.factories[format]
	if !ok {
		return nil, fmt.Errorf("export format %q is not compiled into this build (compiled: %s)",
			format, formatsList(RegisteredFormats()))
	}
	return f(), nil
}

// Supported 报告本构建是否注册了该格式。
func Supported(format Format) bool {
	_, ok := defaultRegistry.factories[format]
	return ok
}

// RegisteredFormats 返回本构建已注册的导出格式（升序），用于 fail-closed 文案与诊断。
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
