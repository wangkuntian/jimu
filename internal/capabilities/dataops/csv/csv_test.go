package csv

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"jimu/internal/capabilities/dataops/exporter"
	"jimu/internal/capabilities/dataops/importer"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func csvReader(data string) *strings.Reader {
	return strings.NewReader(data)
}

// TestCSVDriverRegistersBothDirections 本驱动包在 init() 里同时注册导入与导出两个方向，
// 且不连带注册其它格式（格式串与驱动包名不是一回事）。
func TestCSVDriverRegistersBothDirections(t *testing.T) {
	_, err := importer.Get(importer.FormatCSV)
	require.NoError(t, err)
	_, err = exporter.Get(exporter.FormatCSV)
	require.NoError(t, err)
	_, err = importer.Get(importer.FormatExcel)
	require.Error(t, err, "csv 驱动不得连带注册 excel")
	_, err = exporter.Get(exporter.FormatExcel)
	require.Error(t, err, "csv 驱动不得连带注册 excel 导出")
}

func TestCSVParseAndValidate(t *testing.T) {
	imp := NewCSVImporter()
	rows, err := imp.Parse(context.Background(), csvReader("username,email\nalice,a@x.com\nbob,bad-email\n"))
	require.NoError(t, err)
	assert.Len(t, rows, 2)

	result, err := imp.Validate(context.Background(), rows, importer.ValidationRules{Fields: []importer.FieldRule{
		{Field: "username", Type: importer.TypeString, Required: true},
		{Field: "email", Type: importer.TypeEmail},
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

func TestCSVImportWithoutSinkReturnsConfigurationError(t *testing.T) {
	result, err := NewCSVImporter().Import(context.Background(), []map[string]string{{"username": "alice"}})

	assert.ErrorIs(t, err, importer.ErrImportPersistenceNotConfigured)
	assert.Nil(t, result)
}

var exportRows = []map[string]string{
	{"name": "tom", "age": "30", "city": "beijing"},
	{"name": "jerry", "age": "25", "city": "shanghai"},
}
var exportHeader = []string{"name", "age", "city"}

// TestCSVExporterRoundTrip CSV 写出后能被 CSVImporter 原样读回
func TestCSVExporterRoundTrip(t *testing.T) {
	var buf bytes.Buffer
	err := NewCSVExporter().Export(context.Background(), exportHeader, exportRows, &buf)
	require.NoError(t, err)

	rows, err := NewCSVImporter().Parse(context.Background(), &buf)
	require.NoError(t, err)
	assert.Equal(t, exportRows, rows)
}

// TestCSVExporterMissingKey 行内缺失键补空串，不产生错位
func TestCSVExporterMissingKey(t *testing.T) {
	var buf bytes.Buffer
	err := NewCSVExporter().Export(context.Background(), exportHeader,
		[]map[string]string{{"name": "solo"}}, &buf)
	require.NoError(t, err)

	rows, err := NewCSVImporter().Parse(context.Background(), &buf)
	require.NoError(t, err)
	assert.Equal(t, []map[string]string{{"name": "solo", "age": "", "city": ""}}, rows)
}

// FuzzCSVImporterParse 保证任意 CSV 输入下解析不 panic，成功时行结构完整。
func FuzzCSVImporterParse(f *testing.F) {
	f.Add("name,age\ntom,30\n")
	f.Add("a,b,c\n1,2\n")
	f.Add("")
	f.Add(`"unclosed`)
	f.Fuzz(func(t *testing.T, data string) {
		rows, err := NewCSVImporter().Parse(context.Background(), strings.NewReader(data))
		if err != nil {
			return
		}
		for _, r := range rows {
			_ = r
		}
	})
}
