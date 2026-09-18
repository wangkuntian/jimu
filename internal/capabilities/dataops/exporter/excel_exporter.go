package exporter

import (
	"context"
	"fmt"
	"io"

	"github.com/xuri/excelize/v2"
)

// ExcelExporter 将行数据渲染为 Excel (.xlsx)。
type ExcelExporter struct {
	Sheet string // 可选；空时写入默认 "Sheet1"
}

// NewExcelExporter 创建 Excel 导出器。
func NewExcelExporter() *ExcelExporter {
	return &ExcelExporter{}
}

// Export 将表头与行数据写入第一个 sheet，单元格从 A1 起逐行填充。
func (e *ExcelExporter) Export(ctx context.Context, header []string, rows []map[string]string, w io.Writer) error {
	sheet := e.Sheet
	if sheet == "" {
		sheet = "Sheet1"
	}

	f := excelize.NewFile()
	defer func() { _ = f.Close() }()

	if e.Sheet != "" && e.Sheet != "Sheet1" {
		idx, err := f.NewSheet(sheet)
		if err != nil {
			return fmt.Errorf("create sheet %q: %w", sheet, err)
		}
		f.SetActiveSheet(idx)
	}

	lines := [][]string{header}
	for _, row := range rows {
		if err := ctx.Err(); err != nil {
			return err
		}
		record := make([]string, len(header))
		for i, h := range header {
			record[i] = row[h]
		}
		lines = append(lines, record)
	}
	for i, line := range lines {
		cell, _ := excelize.CoordinatesToCellName(1, i+1)
		if err := f.SetSheetRow(sheet, cell, &line); err != nil {
			return fmt.Errorf("write excel row %d: %w", i+1, err)
		}
	}

	if err := f.Write(w); err != nil {
		return fmt.Errorf("write excel file: %w", err)
	}
	return nil
}
