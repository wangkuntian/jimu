package searchmysql

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSearcherIgnoresEmptyInput(t *testing.T) {
	s := &searcher{}
	assert.NoError(t, s.Index(context.Background()))
	assert.NoError(t, s.Delete(context.Background(), 1, "", 1))
	assert.NoError(t, s.Delete(context.Background(), 1, "user"))
	results, err := s.Search(context.Background(), 1, "", 10)
	assert.NoError(t, err)
	assert.Nil(t, results)
}
