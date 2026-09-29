// Package csv 提供 CSV 导入/导出驱动：一个驱动包在自己的 init() 里同时注册导入与
// 导出两个方向。
//
// 注意：驱动**包名**（csv）与格式串（importer.FormatCSV / exporter.FormatCSV，取值
// "csv"）不是一回事 —— 格式串是注册表的键，包名只用于形态清单 drivers.go 的 blank import。
package csv

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"

	"jimu/internal/capabilities/dataops/exporter"
	"jimu/internal/capabilities/dataops/importer"
)

func init() {
	importer.Register(importer.FormatCSV, func() importer.Importer { return NewImporter() })
	exporter.Register(exporter.FormatCSV, func() exporter.Exporter { return NewExporter() })
}

// NewImporter / NewExporter 返回本驱动实现（注册表之外也可直接使用）。
func NewImporter() importer.Importer { return &CSVImporter{} }
func NewExporter() exporter.Exporter { return &CSVExporter{} }

// CSVImporter parses CSV files into row maps.
type CSVImporter struct {
	Sink importer.RowSink
}

// NewCSVImporter creates a new CSV importer.
func NewCSVImporter() *CSVImporter {
	return &CSVImporter{}
}

// NewCSVImporterWithSink creates a CSV importer with a row persistence callback.
func NewCSVImporterWithSink(sink importer.RowSink) *CSVImporter {
	return &CSVImporter{Sink: sink}
}

// Parse reads a CSV file and returns rows as maps keyed by header names.
func (c *CSVImporter) Parse(ctx context.Context, file io.Reader) ([]map[string]string, error) {
	reader := csv.NewReader(file)
	reader.FieldsPerRecord = -1 // allow variable columns; we validate later
	reader.TrimLeadingSpace = true

	headers, err := reader.Read()
	if err != nil {
		return nil, fmt.Errorf("read CSV header: %w", err)
	}
	if len(headers) == 0 {
		return nil, fmt.Errorf("CSV file has no headers")
	}

	var rows []map[string]string
	line := 2 // data starts at line 2 (1-based, after header)
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("read CSV line %d: %w", line, err)
		}

		row := make(map[string]string, len(headers))
		for i, h := range headers {
			if i < len(record) {
				row[h] = record[i]
			} else {
				row[h] = ""
			}
		}
		rows = append(rows, row)
		line++
	}

	return rows, nil
}

// Validate parses then validates the CSV content (Parse + Validate convenience).
func (c *CSVImporter) Validate(ctx context.Context, rows []map[string]string, rules importer.ValidationRules) (*importer.ImportResult, error) {
	return importer.NewValidator().Validate(ctx, rows, rules)
}

// Import persists rows through the configured sink and reports per-row errors.
func (c *CSVImporter) Import(ctx context.Context, rows []map[string]string) (*importer.ImportResult, error) {
	return importer.ImportRows(ctx, rows, c.Sink)
}

// CSVExporter 将行数据渲染为 CSV。
type CSVExporter struct{}

// NewCSVExporter 创建 CSV 导出器。
func NewCSVExporter() *CSVExporter {
	return &CSVExporter{}
}

// Export 按 header 顺序写表头与每行数据，缺失键补空串。
func (c *CSVExporter) Export(ctx context.Context, header []string, rows []map[string]string, w io.Writer) error {
	writer := csv.NewWriter(w)
	defer writer.Flush()

	if err := writer.Write(header); err != nil {
		return fmt.Errorf("write CSV header: %w", err)
	}
	for _, row := range rows {
		if err := ctx.Err(); err != nil {
			return err
		}
		record := make([]string, len(header))
		for i, h := range header {
			record[i] = row[h]
		}
		if err := writer.Write(record); err != nil {
			return fmt.Errorf("write CSV row: %w", err)
		}
	}
	return writer.Error()
}
