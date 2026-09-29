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

func TestEscapeLike(t *testing.T) {
	// % _ 与转义符本身都要转义，避免用户输入被当作通配符
	assert.Equal(t, "100!%", escapeLike("100%"))
	assert.Equal(t, "a!_b", escapeLike("a_b"))
	assert.Equal(t, "!!", escapeLike("!"))
	assert.Equal(t, "普通文本", escapeLike("普通文本"))
	assert.Equal(t, "%!%数据库!_%", likePattern("%数据库_"))
}

func TestItoa(t *testing.T) {
	assert.Equal(t, "0", itoa(0))
	assert.Equal(t, "2", itoa(2))
	assert.Equal(t, "42", itoa(42))
	assert.Equal(t, "-7", itoa(-7))
}
