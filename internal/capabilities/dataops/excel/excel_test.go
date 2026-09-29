package excel

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"jimu/internal/capabilities/dataops/exporter"
	"jimu/internal/capabilities/dataops/importer"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xuri/excelize/v2"
)

// TestExcelDriverRegistersBothDirections 本驱动包在 init() 里同时注册导入与导出两个方向，
// 且不连带注册其它格式（格式串 "xlsx" 与驱动包名 excel 不是一回事）。
func TestExcelDriverRegistersBothDirections(t *testing.T) {
	_, err := importer.Get(importer.FormatExcel)
	require.NoError(t, err)
	_, err = exporter.Get(exporter.FormatExcel)
	require.NoError(t, err)
	_, err = importer.Get(importer.FormatCSV)
	require.Error(t, err, "excel 驱动不得连带注册 csv")
	_, err = exporter.Get(exporter.FormatCSV)
	require.Error(t, err, "excel 驱动不得连带注册 csv 导出")
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

func TestExcelParseMalformedInputReturnsErrorWithoutPanic(t *testing.T) {
	imp := NewExcelImporter()

	// 非法/构造过的 xlsx 输入：必须返回错误而不是 panic（excelize 的负共享字符串索引
	// 一类缺陷见 GO-2026-6452，导入器已做 recover 兜底）
	for _, data := range []string{
		"",
		"not-an-xlsx",
		"PK\x03\x04this-is-not-a-zip-record",
	} {
		var (
			rows []map[string]string
			err  error
		)
		require.NotPanics(t, func() {
			rows, err = imp.Parse(context.Background(), strings.NewReader(data))
		}, "input=%q", data)
		assert.Error(t, err, "input=%q", data)
		assert.Nil(t, rows)
	}
}

func TestExcelParseValidWorkbook(t *testing.T) {
	// 正常路径：构造一个含表头与两行数据的工作簿，验证 readSheet 搬入本包后仍可解析
	file := excelize.NewFile()
	sheet := file.GetSheetName(file.GetActiveSheetIndex())
	require.NoError(t, file.SetSheetRow(sheet, "A1", &[]interface{}{"username", "email"}))
	require.NoError(t, file.SetSheetRow(sheet, "A2", &[]interface{}{"alice", "alice@example.com"}))
	require.NoError(t, file.SetSheetRow(sheet, "A3", &[]interface{}{"bob", "bob@example.com"}))
	buf, err := file.WriteToBuffer()
	require.NoError(t, err)
	require.NoError(t, file.Close())

	rows, err := NewExcelImporter().Parse(context.Background(), bytes.NewReader(buf.Bytes()))
	require.NoError(t, err)
	require.Len(t, rows, 2)
	assert.Equal(t, map[string]string{"username": "alice", "email": "alice@example.com"}, rows[0])
	assert.Equal(t, map[string]string{"username": "bob", "email": "bob@example.com"}, rows[1])
}

var exportRows = []map[string]string{
	{"name": "tom", "age": "30", "city": "beijing"},
	{"name": "jerry", "age": "25", "city": "shanghai"},
}
var exportHeader = []string{"name", "age", "city"}

// TestExcelExporterRoundTrip Excel 写出后能被 ExcelImporter 原样读回
func TestExcelExporterRoundTrip(t *testing.T) {
	var buf bytes.Buffer
	err := NewExcelExporter().Export(context.Background(), exportHeader, exportRows, &buf)
	require.NoError(t, err)

	rows, err := NewExcelImporter().Parse(context.Background(), &buf)
	require.NoError(t, err)
	assert.Equal(t, exportRows, rows)
}
