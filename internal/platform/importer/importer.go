package importer

import (
	"context"
	"errors"
	"fmt"
	"io"
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

func importRows(ctx context.Context, rows []map[string]string, sink RowSink) (*ImportResult, error) {
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

// Registry holds importer instances by format.
type Registry struct {
	importers map[Format]Importer
}

// NewRegistry creates an empty registry.
func NewRegistry() *Registry {
	return &Registry{importers: make(map[Format]Importer)}
}

// Register adds an importer for the given format.
func (r *Registry) Register(format Format, imp Importer) {
	r.importers[format] = imp
}

// Get returns the importer for the given format, or an error if unsupported.
func (r *Registry) Get(format Format) (Importer, error) {
	imp, ok := r.importers[format]
	if !ok {
		return nil, fmt.Errorf("unsupported import format: %s", format)
	}
	return imp, nil
}
