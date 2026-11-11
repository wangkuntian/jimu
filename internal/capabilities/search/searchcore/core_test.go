package searchcore

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestToRowsValidatesDocument(t *testing.T) {
	_, err := ToRows([]Document{{DocID: 1}})
	assert.ErrorContains(t, err, "type is empty")

	_, err = ToRows([]Document{{Type: "user"}})
	assert.ErrorContains(t, err, "id is empty")

	rows, err := ToRows([]Document{{TenantID: 7, Type: "user", DocID: 9, Title: "t", Body: "b"}})
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, uint64(7), rows[0].TenantID)
	assert.Equal(t, "user", rows[0].DocType)
	assert.Equal(t, uint64(9), rows[0].DocID)
}

func TestNormalizeLimit(t *testing.T) {
	assert.Equal(t, DefaultLimit, NormalizeLimit(0))
	assert.Equal(t, DefaultLimit, NormalizeLimit(-5))
	assert.Equal(t, 3, NormalizeLimit(3))
}
