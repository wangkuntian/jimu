package importer

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"github.com/xuri/excelize/v2"
)

// ExcelImporter parses Excel (.xlsx) files into row maps.
type ExcelImporter struct {
	Sheet string // optional sheet name; empty means the active sheet
	Sink  RowSink
}

// NewExcelImporter creates a new Excel importer targeting the active sheet.
func NewExcelImporter() *ExcelImporter {
	return &ExcelImporter{}
}

// NewExcelImporterWithSink creates an Excel importer with a row persistence callback.
func NewExcelImporterWithSink(sink RowSink) *ExcelImporter {
	return &ExcelImporter{Sink: sink}
}

// Parse reads an Excel file and returns rows as maps keyed by header names.
func (e *ExcelImporter) Parse(ctx context.Context, file io.Reader) ([]map[string]string, error) {
	data, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("read excel file: %w", err)
	}

	f, err := excelize.OpenReader(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("open excel file: %w", err)
	}
	defer func() { _ = f.Close() }()

	sheet := e.Sheet
	if sheet == "" {
		sheet = f.GetSheetName(f.GetActiveSheetIndex())
	}

	rows, err := f.GetRows(sheet)
	if err != nil {
		return nil, fmt.Errorf("read sheet %q: %w", sheet, err)
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
func (e *ExcelImporter) Validate(ctx context.Context, rows []map[string]string, rules ValidationRules) (*ImportResult, error) {
	return NewValidator().Validate(ctx, rows, rules)
}

// Import persists rows through the configured sink and reports per-row errors.
func (e *ExcelImporter) Import(ctx context.Context, rows []map[string]string) (*ImportResult, error) {
	return importRows(ctx, rows, e.Sink)
}
