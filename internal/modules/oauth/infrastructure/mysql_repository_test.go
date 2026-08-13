// internal/modules/oauth/infrastructure/mysql_repository_test.go
package infrastructure

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"jimu/internal/modules/oauth/domain"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func newMockGormDB(t *testing.T) (*sql.DB, sqlmock.Sqlmock, *gorm.DB) {
	t.Helper()
	mockDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = mockDB.Close() })
	dialector := mysql.New(mysql.Config{Conn: mockDB, SkipInitializeWithVersion: true})
	db, err := gorm.Open(dialector, &gorm.Config{})
	require.NoError(t, err)
	return mockDB, mock, db
}

func bindingRows() *sqlmock.Rows {
	return sqlmock.NewRows([]string{"id", "user_id", "provider", "subject", "created_at", "updated_at"}).
		AddRow(1, 42, "github", "sub123", time.Now(), time.Now())
}

func TestFindByProviderSubjectFound(t *testing.T) {
	_, mock, db := newMockGormDB(t)
	repo := NewMySQLBindingRepository(db)
	ctx := context.Background()

	mock.ExpectQuery("SELECT \\* FROM `user_oauth_bindings`").
		WithArgs("github", "sub123", 1).
		WillReturnRows(bindingRows())

	binding, err := repo.FindByProviderSubject(ctx, "github", "sub123")
	require.NoError(t, err)
	require.NotNil(t, binding)
	assert.Equal(t, uint64(42), binding.UserID)
	assert.Equal(t, "github", binding.Provider)
	assert.Equal(t, "sub123", binding.Subject)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestFindByProviderSubjectNotFound(t *testing.T) {
	_, mock, db := newMockGormDB(t)
	repo := NewMySQLBindingRepository(db)

	mock.ExpectQuery("SELECT \\* FROM `user_oauth_bindings`").
		WithArgs("wechat", "wx-1", 1).
		WillReturnError(gorm.ErrRecordNotFound)

	_, err := repo.FindByProviderSubject(context.Background(), "wechat", "wx-1")
	assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestFindByProviderSubjectDBError(t *testing.T) {
	_, mock, db := newMockGormDB(t)
	repo := NewMySQLBindingRepository(db)

	mock.ExpectQuery("SELECT \\* FROM `user_oauth_bindings`").
		WillReturnError(errors.New("connection lost"))

	_, err := repo.FindByProviderSubject(context.Background(), "github", "sub123")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "connection lost")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestCreateSetsBindingID(t *testing.T) {
	_, mock, db := newMockGormDB(t)
	repo := NewMySQLBindingRepository(db)
	binding := &domain.OAuthBinding{UserID: 42, Provider: "github", Subject: "sub123"}

	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO `user_oauth_bindings`").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	require.NoError(t, repo.Create(context.Background(), binding))
	assert.Equal(t, uint64(1), binding.ID)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestCreateError(t *testing.T) {
	_, mock, db := newMockGormDB(t)
	repo := NewMySQLBindingRepository(db)

	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO `user_oauth_bindings`").
		WillReturnError(errors.New("duplicate entry"))
	mock.ExpectRollback()

	err := repo.Create(context.Background(), &domain.OAuthBinding{UserID: 42, Provider: "github", Subject: "sub123"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "duplicate entry")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestNewMySQLBindingRepositorySatisfiesInterface(t *testing.T) {
	_, _, db := newMockGormDB(t)
	var repo domain.BindingRepository = NewMySQLBindingRepository(db)
	assert.NotNil(t, repo)
}
