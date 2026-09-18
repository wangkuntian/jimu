package importer

import (
	"context"
	"fmt"
	"net/mail"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// FieldType describes the expected type of a field value.
type FieldType string

const (
	TypeString FieldType = "string"
	TypeInt    FieldType = "int"
	TypeFloat  FieldType = "float"
	TypeBool   FieldType = "bool"
	TypeDate   FieldType = "date"
	TypeEmail  FieldType = "email"
	TypePhone  FieldType = "phone"
)

// FieldRule defines validation constraints for a single field.
type FieldRule struct {
	Field    string    `json:"field"`
	Type     FieldType `json:"type"`
	Required bool      `json:"required"`
	Unique   bool      `json:"unique"` // uniqueness is checked across the batch
	Pattern  string    `json:"pattern,omitempty"`
	Min      *float64  `json:"min,omitempty"`
	Max      *float64  `json:"max,omitempty"`
}

// ValidationRules is the collection of field rules for an import operation.
type ValidationRules struct {
	Fields []FieldRule `json:"fields"`
}

// Validator performs row-level validation against rules.
type Validator struct {
	uniqueSets map[string]map[string]bool // field → set of seen values
}

// NewValidator creates a new Validator with initialized state.
func NewValidator() *Validator {
	return &Validator{uniqueSets: make(map[string]map[string]bool)}
}

// Validate checks every row against the rules and collects errors.
// A row with errors is counted as an error row but does not halt the batch.
func (v *Validator) Validate(ctx context.Context, rows []map[string]string, rules ValidationRules) (*ImportResult, error) {
	start := time.Now()
	result := NewImportResult(len(rows))

	// Build a quick lookup: field name → rule.
	ruleByField := make(map[string]FieldRule, len(rules.Fields))
	for _, r := range rules.Fields {
		ruleByField[r.Field] = r
	}

	for i, row := range rows {
		select {
		case <-ctx.Done():
			result.Finalize(start)
			return result, ctx.Err()
		default:
		}

		rowErr := false
		rowNum := i + 1 // 1-based row number for reporting

		for fieldName, rule := range ruleByField {
			value, exists := row[fieldName]

			// Required check.
			if rule.Required && (!exists || strings.TrimSpace(value) == "") {
				result.AddError(rowNum, fieldName, "field is required", value)
				rowErr = true
				continue
			}

			// Skip further checks if value is empty and not required.
			if !exists || strings.TrimSpace(value) == "" {
				continue
			}

			// Type check.
			if err := checkType(rule.Type, value); err != "" {
				result.AddError(rowNum, fieldName, err, value)
				rowErr = true
				continue
			}

			// Range check for numeric types.
			if rule.Type == TypeInt || rule.Type == TypeFloat {
				n, _ := strconv.ParseFloat(value, 64)
				if rule.Min != nil && n < *rule.Min {
					result.AddError(rowNum, fieldName, fmt.Sprintf("must be >= %v", *rule.Min), value)
					rowErr = true
				}
				if rule.Max != nil && n > *rule.Max {
					result.AddError(rowNum, fieldName, fmt.Sprintf("must be <= %v", *rule.Max), value)
					rowErr = true
				}
			}

			// Pattern check.
			if rule.Pattern != "" {
				re, err := regexp.Compile(rule.Pattern)
				if err == nil && !re.MatchString(value) {
					result.AddError(rowNum, fieldName, "does not match required pattern", value)
					rowErr = true
				}
			}

			// Uniqueness check within the batch.
			if rule.Unique {
				if v.uniqueSets[fieldName] == nil {
					v.uniqueSets[fieldName] = make(map[string]bool)
				}
				if v.uniqueSets[fieldName][value] {
					result.AddError(rowNum, fieldName, "duplicate value in import", value)
					rowErr = true
				} else {
					v.uniqueSets[fieldName][value] = true
				}
			}
		}

		_ = rowErr // reserved for future partial-success handling
	}

	result.Finalize(start)
	return result, nil
}

// checkType returns an empty string if the value matches the expected type.
func checkType(t FieldType, value string) string {
	switch t {
	case TypeString:
		return ""
	case TypeInt:
		if _, err := strconv.ParseInt(value, 10, 64); err != nil {
			return "must be an integer"
		}
	case TypeFloat:
		if _, err := strconv.ParseFloat(value, 64); err != nil {
			return "must be a number"
		}
	case TypeBool:
		switch strings.ToLower(value) {
		case "true", "false", "1", "0", "yes", "no":
			return ""
		default:
			return "must be a boolean (true/false/1/0/yes/no)"
		}
	case TypeDate:
		for _, layout := range []string{time.RFC3339, "2006-01-02", "2006/01/02", "2006-01-02 15:04:05"} {
			if _, err := time.Parse(layout, value); err == nil {
				return ""
			}
		}
		return "must be a valid date (YYYY-MM-DD)"
	case TypeEmail:
		if _, err := mail.ParseAddress(value); err != nil {
			return "must be a valid email address"
		}
	case TypePhone:
		// Simple international phone check: optional +, then 7-15 digits.
		re := regexp.MustCompile(`^\+?[0-9]{7,15}$`)
		if !re.MatchString(value) {
			return "must be a valid phone number (7-15 digits, optional +)"
		}
	default:
		return ""
	}
	return ""
}
