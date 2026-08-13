package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHashKey(t *testing.T) {
	// 哈希确定性且长度固定（SHA-256 64 位十六进制）
	h1 := HashKey("jimu_abc")
	h2 := HashKey("jimu_abc")
	assert.Equal(t, h1, h2)
	assert.Len(t, h1, 64)
	assert.NotEqual(t, h1, HashKey("jimu_abd"))
	assert.NotEqual(t, h1, HashKey(""))
}

func TestAPIKeyTableName(t *testing.T) {
	assert.Equal(t, "api_keys", (APIKey{}).TableName())
}
