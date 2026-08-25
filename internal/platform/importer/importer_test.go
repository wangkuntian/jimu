package importer

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func csvReader(data string) *strings.Reader {
	return strings.NewReader(data)
}

func TestCSVParseAndValidate(t *testing.T) {
	imp := NewCSVImporter()
	rows, err := imp.Parse(context.Background(), csvReader("username,email\nalice,a@x.com\nbob,bad-email\n"))
	require.NoError(t, err)
	assert.Len(t, rows, 2)

	result, err := imp.Validate(context.Background(), rows, ValidationRules{Fields: []FieldRule{
		{Field: "username", Type: TypeString, Required: true},
		{Field: "email", Type: TypeEmail},
	}})
	require.NoError(t, err)
	assert.Equal(t, 2, result.TotalRows)
	assert.Equal(t, 1, result.ErrorRows)
	assert.Equal(t, 1, result.SuccessRows)
	assert.Equal(t, "must be a valid email address", result.Errors[0].Message)
}

func TestCSVParseEmptyFile(t *testing.T) {
	imp := NewCSVImporter()
	_, err := imp.Parse(context.Background(), csvReader(""))
	require.Error(t, err)
}

func TestValidateUnique(t *testing.T) {
	rows := []map[string]string{
		{"username": "alice"},
		{"username": "alice"},
	}
	result, err := NewValidator().Validate(context.Background(), rows, ValidationRules{Fields: []FieldRule{
		{Field: "username", Type: TypeString, Required: true, Unique: true},
	}})
	require.NoError(t, err)
	assert.Equal(t, 1, result.ErrorRows)
	assert.Equal(t, "duplicate value in import", result.Errors[0].Message)
}

func TestRegistryGetUnsupported(t *testing.T) {
	r := NewRegistry()
	_, err := r.Get(Format("yaml"))
	require.Error(t, err)
}

func TestCSVImportPersistsRowsThroughSink(t *testing.T) {
	var imported []string
	imp := NewCSVImporterWithSink(func(ctx context.Context, row map[string]string) error {
		imported = append(imported, row["username"])
		if row["username"] == "bob" {
			return errors.New("username already exists")
		}
		return nil
	})

	rows := []map[string]string{
		{"username": "alice"},
		{"username": "bob"},
		{"username": "carol"},
	}
	result, err := imp.Import(context.Background(), rows)

	require.NoError(t, err)
	assert.Equal(t, []string{"alice", "bob", "carol"}, imported)
	assert.Equal(t, 3, result.TotalRows)
	assert.Equal(t, 2, result.SuccessRows)
	assert.Equal(t, 1, result.ErrorRows)
	require.Len(t, result.Errors, 1)
	assert.Equal(t, 2, result.Errors[0].Row)
	assert.Equal(t, "username already exists", result.Errors[0].Message)
}

func TestExcelImportPersistsRowsThroughSink(t *testing.T) {
	var imported []string
	imp := NewExcelImporterWithSink(func(ctx context.Context, row map[string]string) error {
		imported = append(imported, row["username"])
		return nil
	})

	result, err := imp.Import(context.Background(), []map[string]string{{"username": "alice"}})

	require.NoError(t, err)
	assert.Equal(t, []string{"alice"}, imported)
	assert.Equal(t, 1, result.SuccessRows)
}

func TestImportWithoutSinkReturnsConfigurationError(t *testing.T) {
	result, err := NewCSVImporter().Import(context.Background(), []map[string]string{{"username": "alice"}})

	assert.ErrorIs(t, err, ErrImportPersistenceNotConfigured)
	assert.Nil(t, result)
}
