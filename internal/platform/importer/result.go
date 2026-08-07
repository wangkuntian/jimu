package importer

import "time"

// ImportResult contains the outcome of an import operation.
type ImportResult struct {
	TotalRows   int           `json:"total_rows"`
	SuccessRows int           `json:"success_rows"`
	ErrorRows   int           `json:"error_rows"`
	Errors      []ImportError `json:"errors,omitempty"`
	Duration    string        `json:"duration"`
}

// ImportError describes a single validation/import error for a specific row and field.
type ImportError struct {
	Row     int    `json:"row"`
	Field   string `json:"field"`
	Message string `json:"message"`
	Value   string `json:"value"`
}

// NewImportResult creates an empty result with zeroed counters.
func NewImportResult(total int) *ImportResult {
	return &ImportResult{
		TotalRows: total,
		Errors:    make([]ImportError, 0),
	}
}

// AddError appends an error and increments the error row counter.
func (r *ImportResult) AddError(row int, field, message, value string) {
	r.Errors = append(r.Errors, ImportError{
		Row:     row,
		Field:   field,
		Message: message,
		Value:   value,
	})
	r.ErrorRows++
}

// Finalize computes success_rows and sets the duration string.
func (r *ImportResult) Finalize(start time.Time) {
	r.SuccessRows = r.TotalRows - r.ErrorRows
	r.Duration = time.Since(start).String()
}
