package search

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestContainsCJK(t *testing.T) {
	tests := []struct {
		query string
		want  bool
	}{
		{"golang", false},
		{"structured logging", false},
		{"v1.2.3", false},
		{"数据库", true},
		{"golang 后端", true},
		{"検索エンジン", true},
		{"한국어", true},
		{"", false},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.want, containsCJK(tt.query), "query=%q", tt.query)
	}
}

func TestItoa(t *testing.T) {
	assert.Equal(t, "0", itoa(0))
	assert.Equal(t, "2", itoa(2))
	assert.Equal(t, "42", itoa(42))
	assert.Equal(t, "-7", itoa(-7))
}
