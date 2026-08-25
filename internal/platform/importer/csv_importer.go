package importer

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
)

// CSVImporter parses CSV files into row maps.
type CSVImporter struct {
	Sink RowSink
}

// NewCSVImporter creates a new CSV importer.
func NewCSVImporter() *CSVImporter {
	return &CSVImporter{}
}

// NewCSVImporterWithSink creates a CSV importer with a row persistence callback.
func NewCSVImporterWithSink(sink RowSink) *CSVImporter {
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
func (c *CSVImporter) Validate(ctx context.Context, rows []map[string]string, rules ValidationRules) (*ImportResult, error) {
	return NewValidator().Validate(ctx, rows, rules)
}

// Import persists rows through the configured sink and reports per-row errors.
func (c *CSVImporter) Import(ctx context.Context, rows []map[string]string) (*ImportResult, error) {
	return importRows(ctx, rows, c.Sink)
}
