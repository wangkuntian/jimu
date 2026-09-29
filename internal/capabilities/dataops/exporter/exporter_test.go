package exporter

import (
	"context"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeExporter 是注册表用例使用的最小实现（无状态）。
type fakeExporter struct{}

func (fakeExporter) Export(ctx context.Context, header []string, rows []map[string]string, w io.Writer) error {
	return nil
}

func TestRegistryGetUnsupported(t *testing.T) {
	r := NewRegistry()
	_, err := r.Get(Format("pdf"))
	require.Error(t, err)
}

// TestGetRejectsUncompiledFormat 核心包不得自注册任何格式：驱动由形态侧 blank import
// 选中并在 init() 注册，未编进本构建的格式必须 fail-closed 报错并附已编译清单。
func TestGetRejectsUncompiledFormat(t *testing.T) {
	assert.Empty(t, RegisteredFormats(), "核心包不得自注册任何格式")
	_, err := Get(FormatExcel)
	require.ErrorContains(t, err, `export format "xlsx" is not compiled into this build`)
	assert.False(t, Supported(FormatCSV))
}

// TestRegistryIsFactoryBased 注册表存工厂而非实例：每次 Get 构造新实例。
func TestRegistryIsFactoryBased(t *testing.T) {
	calls := 0
	r := NewRegistry()
	r.Register(FormatCSV, func() Exporter { calls++; return fakeExporter{} })
	_, err := r.Get(FormatCSV)
	require.NoError(t, err)
	_, err = r.Get(FormatCSV)
	require.NoError(t, err)
	assert.Equal(t, 2, calls, "每次 Get 构造一个新实例")
	_, err = r.Get(FormatCSV + ".unknown")
	require.ErrorContains(t, err, "unsupported export format")
}
