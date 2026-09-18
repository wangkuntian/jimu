// Package exporter 提供批量数据导出能力，与 importer 反向对称：
// 表头 + 行数据（header → value）渲染为 CSV 或 Excel 文件。
package exporter

import (
	"context"
	"fmt"
	"io"
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

// Registry 按格式管理导出器。
type Registry struct {
	exporters map[Format]Exporter
}

// NewRegistry 创建空注册表。
func NewRegistry() *Registry {
	return &Registry{exporters: make(map[Format]Exporter)}
}

// Register 注册指定格式的导出器。
func (r *Registry) Register(format Format, e Exporter) {
	r.exporters[format] = e
}

// Get 返回指定格式的导出器，不支持时返回错误。
func (r *Registry) Get(format Format) (Exporter, error) {
	e, ok := r.exporters[format]
	if !ok {
		return nil, fmt.Errorf("unsupported export format: %s", format)
	}
	return e, nil
}
