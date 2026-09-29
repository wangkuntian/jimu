package search

import (
	"context"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestNewRejectsNilAndUnsupportedDialect(t *testing.T) {
	_, err := New(nil)
	assert.Error(t, err)

	db, dbErr := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, dbErr)
	_, err = New(db)
	assert.ErrorContains(t, err, "unsupported dialect")
}

func TestToRowsValidatesDocument(t *testing.T) {
	_, err := toRows([]Document{{DocID: 1}})
	assert.ErrorContains(t, err, "type is required")

	_, err = toRows([]Document{{Type: "user"}})
	assert.ErrorContains(t, err, "id is required")

	rows, err := toRows([]Document{{TenantID: 7, Type: "user", DocID: 9, Title: "t", Body: "b"}})
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, uint64(7), rows[0].TenantID)
	assert.Equal(t, "user", rows[0].DocType)
	assert.Equal(t, uint64(9), rows[0].DocID)
}

func TestNormalizeLimit(t *testing.T) {
	assert.Equal(t, defaultLimit, normalizeLimit(0))
	assert.Equal(t, defaultLimit, normalizeLimit(-5))
	assert.Equal(t, 3, normalizeLimit(3))
}

func TestSearcherIgnoresEmptyInput(t *testing.T) {
	// 空文档切片与空查询直接返回，不触碰数据库（db 为 nil 也不会 panic）
	s := &mysqlSearcher{}
	assert.NoError(t, s.Index(context.Background()))
	assert.NoError(t, s.Delete(context.Background(), 1, "", 1))
	assert.NoError(t, s.Delete(context.Background(), 1, "user"))
	results, err := s.Search(context.Background(), 1, "", 10)
	assert.NoError(t, err)
	assert.Nil(t, results)
}
