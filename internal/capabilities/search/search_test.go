package search

import (
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestNewRejectsNilAndUncompiledDialect(t *testing.T) {
	_, err := New(nil)
	assert.Error(t, err)

	db, dbErr := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, dbErr)
	_, err = New(db)
	assert.ErrorContains(t, err, "not compiled into this build")
}
