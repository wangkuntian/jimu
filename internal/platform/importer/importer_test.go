package importer

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xuri/excelize/v2"
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
	// 正常路径：构造一个含表头与两行数据的工作簿，验证 readSheet 重构后仍可解析
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
