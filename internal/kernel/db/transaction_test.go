package db

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestTransaction_Commit(t *testing.T) {
	db, mock := newMockGormDB(t)
	mock.ExpectBegin()
	mock.ExpectCommit()

	err := Transaction(db, func(tx *gorm.DB) error { return nil })
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestTransaction_Rollback(t *testing.T) {
	db, mock := newMockGormDB(t)
	mock.ExpectBegin()
	mock.ExpectRollback()

	err := Transaction(db, func(tx *gorm.DB) error { return errors.New("nope") })
	require.ErrorContains(t, err, "nope")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestWithTx_Commit(t *testing.T) {
	db, mock := newMockGormDB(t)
	mock.ExpectBegin()
	mock.ExpectCommit()

	err := WithTx(db, func(tx *gorm.DB) error { return nil })
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestWithTx_Rollback(t *testing.T) {
	db, mock := newMockGormDB(t)
	mock.ExpectBegin()
	mock.ExpectRollback()

	sentinel := errors.New("rollback me")
	err := WithTx(db, func(tx *gorm.DB) error { return sentinel })
	assert.ErrorIs(t, err, sentinel)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestWithTx_BeginError(t *testing.T) {
	db, mock := newMockGormDB(t)
	mock.ExpectBegin().WillReturnError(errors.New("begin failed"))

	err := WithTx(db, func(tx *gorm.DB) error { return nil })
	require.ErrorContains(t, err, "begin failed")
}

func TestWithTx_PanicRollsBack(t *testing.T) {
	db, mock := newMockGormDB(t)
	mock.ExpectBegin()
	mock.ExpectRollback()

	require.PanicsWithValue(t, "boom", func() {
		_ = WithTx(db, func(tx *gorm.DB) error {
			panic("boom")
		})
	})
}
