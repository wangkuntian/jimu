package importer

import (
	"context"
	"errors"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeImporter 是注册表用例使用的最小实现（无状态）。
type fakeImporter struct{}

func (fakeImporter) Parse(ctx context.Context, file io.Reader) ([]map[string]string, error) {
	return nil, nil
}
func (fakeImporter) Validate(ctx context.Context, rows []map[string]string, rules ValidationRules) (*ImportResult, error) {
	return nil, nil
}
func (fakeImporter) Import(ctx context.Context, rows []map[string]string) (*ImportResult, error) {
	return nil, nil
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

// TestImportRowsPersistsRowsThroughSink 逐行落库的公共实现：驱动包复用 ImportRows。
func TestImportRowsPersistsRowsThroughSink(t *testing.T) {
	var imported []string
	sink := func(ctx context.Context, row map[string]string) error {
		imported = append(imported, row["username"])
		if row["username"] == "bob" {
			return errors.New("username already exists")
		}
		return nil
	}

	rows := []map[string]string{
		{"username": "alice"},
		{"username": "bob"},
		{"username": "carol"},
	}
	result, err := ImportRows(context.Background(), rows, sink)

	require.NoError(t, err)
	assert.Equal(t, []string{"alice", "bob", "carol"}, imported)
	assert.Equal(t, 3, result.TotalRows)
	assert.Equal(t, 2, result.SuccessRows)
	assert.Equal(t, 1, result.ErrorRows)
	require.Len(t, result.Errors, 1)
	assert.Equal(t, 2, result.Errors[0].Row)
	assert.Equal(t, "username already exists", result.Errors[0].Message)
}

func TestImportRowsWithoutSinkReturnsConfigurationError(t *testing.T) {
	result, err := ImportRows(context.Background(), []map[string]string{{"username": "alice"}}, nil)

	assert.ErrorIs(t, err, ErrImportPersistenceNotConfigured)
	assert.Nil(t, result)
}

// TestGetRejectsUncompiledFormat 核心包不得自注册任何格式：驱动由形态侧 blank import
// 选中并在 init() 注册，未编进本构建的格式必须 fail-closed 报错并附已编译清单。
func TestGetRejectsUncompiledFormat(t *testing.T) {
	assert.Empty(t, RegisteredFormats(), "核心包不得自注册任何格式")
	_, err := Get(FormatExcel)
	require.ErrorContains(t, err, `import format "xlsx" is not compiled into this build`)
	assert.False(t, Supported(FormatCSV))
}

// TestRegistryIsFactoryBased 注册表存工厂而非实例：每次 Get 构造新实例。
func TestRegistryIsFactoryBased(t *testing.T) {
	calls := 0
	r := NewRegistry()
	r.Register(FormatCSV, func() Importer { calls++; return fakeImporter{} })
	_, err := r.Get(FormatCSV)
	require.NoError(t, err)
	_, err = r.Get(FormatCSV)
	require.NoError(t, err)
	assert.Equal(t, 2, calls, "每次 Get 构造一个新实例（无状态实现，语义与下沉前一致）")
	_, err = r.Get(FormatCSV + ".unknown")
	require.ErrorContains(t, err, "unsupported import format")
}
