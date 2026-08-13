package exporter

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
)

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
