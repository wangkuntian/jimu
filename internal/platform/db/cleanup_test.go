package db

import (
	"context"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// cleanupModel 无软删除字段，Delete 走硬删除，便于 sqlmock 断言
type cleanupModel struct {
	ID uint64
}

func (cleanupModel) TableName() string { return "items" }

type noTableNameModel struct {
	ID uint64
}

func TestDefaultCleanupConfig(t *testing.T) {
	cfg := DefaultCleanupConfig()
	assert.Equal(t, 90, cfg.RetentionDays)
	assert.Equal(t, 100, cfg.BatchSize)
}

func TestNewCleanupService_AppliesDefaults(t *testing.T) {
	db, _ := newMockGormDB(t)
	svc := NewCleanupService(db, CleanupConfig{})
	assert.Equal(t, 90, svc.config.RetentionDays)
	assert.Equal(t, 100, svc.config.BatchSize)
}

func TestCleanupService_RunBatches(t *testing.T) {
	db, mock := newMockGormDB(t)
	svc := NewCleanupService(db, CleanupConfig{
		RetentionDays: 30,
		BatchSize:     2,
		Tables:        []CleanupTable{{Model: cleanupModel{}}},
	})

	// gorm 默认给 Delete 包事务；3 批：2 + 1 + 0 → 停止
	for _, n := range []int{2, 1, 0} {
		mock.ExpectBegin()
		mock.ExpectExec("DELETE FROM `items`").WillReturnResult(sqlmock.NewResult(0, int64(n)))
		mock.ExpectCommit()
	}

	results, err := svc.Run(context.Background())
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "items", results[0].Table)
	assert.Equal(t, int64(3), results[0].Deleted)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestCleanupService_RunCustomDeletedColumn(t *testing.T) {
	db, mock := newMockGormDB(t)
	svc := NewCleanupService(db, CleanupConfig{
		RetentionDays: 90,
		BatchSize:     100,
		Tables:        []CleanupTable{{Model: cleanupModel{}, DeletedAt: "deleted_time"}},
	})
	mock.ExpectBegin()
	mock.ExpectExec("deleted_time IS NOT NULL").WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectCommit()

	results, err := svc.Run(context.Background())
	require.NoError(t, err)
	assert.Equal(t, int64(0), results[0].Deleted)
}

func TestCleanupService_RunError(t *testing.T) {
	db, mock := newMockGormDB(t)
	svc := NewCleanupService(db, CleanupConfig{
		RetentionDays: 90,
		BatchSize:     100,
		Tables:        []CleanupTable{{Model: cleanupModel{}}},
	})
	mock.ExpectBegin()
	mock.ExpectExec("DELETE FROM `items`").WillReturnError(errors.New("db down"))
	mock.ExpectRollback()

	results, err := svc.Run(context.Background())
	require.ErrorContains(t, err, "db down")
	assert.Empty(t, results)
}

func TestCleanupService_RunMultipleTables(t *testing.T) {
	db, mock := newMockGormDB(t)
	svc := NewCleanupService(db, CleanupConfig{
		RetentionDays: 90,
		BatchSize:     100,
		Tables: []CleanupTable{
			{Model: cleanupModel{}},
			{Model: cleanupModel{}, DeletedAt: "deleted_time"},
		},
	})
	mock.ExpectBegin()
	mock.ExpectExec("DELETE FROM `items`").WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectCommit()
	mock.ExpectBegin()
	mock.ExpectExec("deleted_time IS NOT NULL").WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectCommit()

	results, err := svc.Run(context.Background())
	require.NoError(t, err)
	require.Len(t, results, 2)
}

func TestTableName(t *testing.T) {
	assert.Equal(t, "items", tableName(cleanupModel{}))
	assert.Equal(t, "unknown", tableName(noTableNameModel{}))
	assert.Equal(t, "unknown", tableName(nil))
}
