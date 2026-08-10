package queue

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNew_Redis(t *testing.T) {
	cfg := Config{Type: "redis"}
	q, err := New(cfg)
	assert.NoError(t, err)
	_, ok := q.(*RedisQueue)
	assert.True(t, ok)
}

func TestNew_InvalidType(t *testing.T) {
	cfg := Config{Type: "invalid"}
	_, err := New(cfg)
	assert.Error(t, err)
}
