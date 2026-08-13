package exporter

import (
	"bytes"
	"context"
	"testing"

	"jimu/internal/platform/importer"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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

	rows, err := importer.NewCSVImporter().Parse(context.Background(), &buf)
	require.NoError(t, err)
	assert.Equal(t, exportRows, rows)
}

// TestCSVExporterMissingKey 行内缺失键补空串，不产生错位
func TestCSVExporterMissingKey(t *testing.T) {
	var buf bytes.Buffer
	err := NewCSVExporter().Export(context.Background(), exportHeader,
		[]map[string]string{{"name": "solo"}}, &buf)
	require.NoError(t, err)

	rows, err := importer.NewCSVImporter().Parse(context.Background(), &buf)
	require.NoError(t, err)
	assert.Equal(t, []map[string]string{{"name": "solo", "age": "", "city": ""}}, rows)
}

// TestExcelExporterRoundTrip Excel 写出后能被 ExcelImporter 原样读回
func TestExcelExporterRoundTrip(t *testing.T) {
	var buf bytes.Buffer
	err := NewExcelExporter().Export(context.Background(), exportHeader, exportRows, &buf)
	require.NoError(t, err)

	rows, err := importer.NewExcelImporter().Parse(context.Background(), &buf)
	require.NoError(t, err)
	assert.Equal(t, exportRows, rows)
}

// TestRegistry 注册表按格式取用
func TestRegistry(t *testing.T) {
	r := NewRegistry()
	r.Register(FormatCSV, NewCSVExporter())
	r.Register(FormatExcel, NewExcelExporter())

	e, err := r.Get(FormatCSV)
	require.NoError(t, err)
	assert.IsType(t, &CSVExporter{}, e)

	_, err = r.Get("pdf")
	assert.Error(t, err)
}
