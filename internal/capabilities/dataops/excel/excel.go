// Package excel 提供 Excel (.xlsx) 导入/导出驱动：一个驱动包在自己的 init() 里同时
// 注册导入与导出两个方向。
//
// 注意：驱动**包名**（excel）与格式串（importer.FormatExcel / exporter.FormatExcel，
// 取值 "xlsx"）不是一回事 —— 格式串是注册表的键，包名只用于形态清单 drivers.go 的
// blank import。本包是 xuri/excelize 依赖的唯一来源。
package excel

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"jimu/internal/capabilities/dataops/exporter"
	"jimu/internal/capabilities/dataops/importer"

	"github.com/xuri/excelize/v2"
)

func init() {
	importer.Register(importer.FormatExcel, func() importer.Importer { return NewImporter() })
	exporter.Register(exporter.FormatExcel, func() exporter.Exporter { return NewExporter() })
}

// NewImporter / NewExporter 返回本驱动实现（注册表之外也可直接使用）。
func NewImporter() importer.Importer { return &ExcelImporter{} }
func NewExporter() exporter.Exporter { return &ExcelExporter{} }

// ExcelImporter parses Excel (.xlsx) files into row maps.
type ExcelImporter struct {
	Sheet string // optional sheet name; empty means the active sheet
	Sink  importer.RowSink
}

// NewExcelImporter creates a new Excel importer targeting the active sheet.
func NewExcelImporter() *ExcelImporter {
	return &ExcelImporter{}
}

// NewExcelImporterWithSink creates an Excel importer with a row persistence callback.
func NewExcelImporterWithSink(sink importer.RowSink) *ExcelImporter {
	return &ExcelImporter{Sink: sink}
}

// readSheet 打开 Excel 并读取目标工作表，panic 一并兜底为错误。
//
// excelize 解析构造过的文件时可能 panic（如 GO-2026-6452 的负共享字符串索引），
// 上游尚未发布修复版本；导入的是用户上传的文件，此处必须保证异常只表现为一个错误，
// 不把请求或后台任务的 goroutine 打挂。
func readSheet(file io.Reader, sheet string) (rows [][]string, err error) {
	defer func() {
		if rec := recover(); rec != nil {
			err = fmt.Errorf("parse excel: %v", rec)
		}
	}()

	data, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("read excel file: %w", err)
	}

	f, err := excelize.OpenReader(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("open excel file: %w", err)
	}
	defer func() { _ = f.Close() }()

	if sheet == "" {
		sheet = f.GetSheetName(f.GetActiveSheetIndex())
	}

	rows, err = f.GetRows(sheet)
	if err != nil {
		return nil, fmt.Errorf("read sheet %q: %w", sheet, err)
	}
	return rows, nil
}

// Parse reads an Excel file and returns rows as maps keyed by header names.
func (e *ExcelImporter) Parse(ctx context.Context, file io.Reader) ([]map[string]string, error) {
	sheet := e.Sheet
	rows, err := readSheet(file, sheet)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, fmt.Errorf("excel sheet %q is empty", sheet)
	}

	headers := rows[0]
	if len(headers) == 0 {
		return nil, fmt.Errorf("excel sheet %q has no headers", sheet)
	}

	var result []map[string]string
	for lineIdx := 1; lineIdx < len(rows); lineIdx++ {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		record := rows[lineIdx]
		row := make(map[string]string, len(headers))
		for i, h := range headers {
			if i < len(record) {
				row[h] = record[i]
			} else {
				row[h] = ""
			}
		}
		result = append(result, row)
	}

	return result, nil
}

// Validate delegates to the shared validation engine.
func (e *ExcelImporter) Validate(ctx context.Context, rows []map[string]string, rules importer.ValidationRules) (*importer.ImportResult, error) {
	return importer.NewValidator().Validate(ctx, rows, rules)
}

// Import persists rows through the configured sink and reports per-row errors.
func (e *ExcelImporter) Import(ctx context.Context, rows []map[string]string) (*importer.ImportResult, error) {
	return importer.ImportRows(ctx, rows, e.Sink)
}

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
