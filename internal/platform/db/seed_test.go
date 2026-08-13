package db

import (
	"errors"
	"fmt"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"jimu/internal/modules/role/domain"
	userdomain "jimu/internal/modules/user/domain"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/mysql"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var seedDBSeq uint64

// newSeedSqliteDB 打开独立的内存 sqlite。
// 用命名共享缓存 DSN：普通 :memory: 每连接独立库，RunSeed 事务与 casbin
// 的 AutoMigrate 可能落在不同连接上导致表不可见。
func newSeedSqliteDB(t *testing.T) *gorm.DB {
	t.Helper()
	seq := atomic.AddUint64(&seedDBSeq, 1)
	dsn := fmt.Sprintf("file:seed_%d_%d?mode=memory&cache=shared", time.Now().UnixNano(), seq)
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	t.Cleanup(func() {
		sqlDB, err := db.DB()
		if err == nil {
			_ = sqlDB.Close()
		}
	})
	return db
}

// repoRoot 项目根目录：migrations 位于根下
func repoRoot(t *testing.T) string {
	t.Helper()
	return filepath.Dir(MigrationDir())
}

// expectRunSeedQueries 为 RunSeed 全流程编排 sqlmock 预期：
// BEGIN → 22×(权限 SELECT+INSERT) → 角色 SELECT+INSERT → 22×(role_permissions
// count+INSERT) → 管理员 SELECT+INSERT → user_roles INSERT → COMMIT
func expectRunSeedQueries(mock sqlmock.Sqlmock) {
	mock.ExpectBegin()

	for range basePermissions() {
		mock.ExpectQuery("SELECT \\* FROM `permissions`").
			WillReturnRows(sqlmock.NewRows(nil))
		mock.ExpectExec("INSERT INTO `permissions`").
			WillReturnResult(sqlmock.NewResult(1, 1))
	}

	mock.ExpectQuery("SELECT \\* FROM `roles`").
		WillReturnRows(sqlmock.NewRows(nil))
	mock.ExpectExec("INSERT INTO `roles`").
		WillReturnResult(sqlmock.NewResult(1, 1))

	for range basePermissions() {
		mock.ExpectQuery("SELECT count\\(\\*\\) FROM `role_permissions`").
			WillReturnRows(sqlmock.NewRows([]string{"count(*)"}).AddRow(0))
		mock.ExpectExec("INSERT INTO role_permissions \\(role_id, permission_id\\)").
			WillReturnResult(sqlmock.NewResult(0, 1))
	}

	mock.ExpectQuery("SELECT \\* FROM `users`").
		WillReturnRows(sqlmock.NewRows(nil))
	mock.ExpectExec("INSERT INTO `users`").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec("INSERT INTO user_roles \\(user_id, role_id\\)").
		WillReturnResult(sqlmock.NewResult(0, 1))

	mock.ExpectCommit()
}

func TestRunSeed_HappyPath(t *testing.T) {
	t.Setenv("ADMIN_PASSWORD", "secret123")

	sqlDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = sqlDB.Close() }()

	db, err := gorm.Open(mysql.New(mysql.Config{Conn: sqlDB, SkipInitializeWithVersion: true}), &gorm.Config{})
	require.NoError(t, err)

	expectRunSeedQueries(mock)

	require.NoError(t, RunSeed(db))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestRunSeed_MissingAdminPassword(t *testing.T) {
	t.Setenv("ADMIN_PASSWORD", "")
	err := RunSeed(nil)
	require.ErrorContains(t, err, "ADMIN_PASSWORD is required")
}

func TestRunSeed_PermissionQueryError(t *testing.T) {
	t.Setenv("ADMIN_PASSWORD", "secret123")

	db, mock := newMockGormDB(t)
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT \\* FROM `permissions`").
		WillReturnError(errors.New("select boom"))
	mock.ExpectRollback()

	err := RunSeed(db)
	require.ErrorContains(t, err, "seed permission failed")
}

func TestRunSeed_RoleInsertError(t *testing.T) {
	t.Setenv("ADMIN_PASSWORD", "secret123")

	db, mock := newMockGormDB(t)
	mock.ExpectBegin()
	for range basePermissions() {
		mock.ExpectQuery("SELECT \\* FROM `permissions`").
			WillReturnRows(sqlmock.NewRows(nil))
		mock.ExpectExec("INSERT INTO `permissions`").
			WillReturnResult(sqlmock.NewResult(1, 1))
	}
	mock.ExpectQuery("SELECT \\* FROM `roles`").
		WillReturnRows(sqlmock.NewRows(nil))
	mock.ExpectExec("INSERT INTO `roles`").
		WillReturnError(errors.New("role boom"))
	mock.ExpectRollback()

	err := RunSeed(db)
	require.ErrorContains(t, err, "seed admin role failed")
}

func TestRunSeed_AssignPermissionError(t *testing.T) {
	t.Setenv("ADMIN_PASSWORD", "secret123")

	db, mock := newMockGormDB(t)
	mock.ExpectBegin()
	for range basePermissions() {
		mock.ExpectQuery("SELECT \\* FROM `permissions`").
			WillReturnRows(sqlmock.NewRows(nil))
		mock.ExpectExec("INSERT INTO `permissions`").
			WillReturnResult(sqlmock.NewResult(1, 1))
	}
	mock.ExpectQuery("SELECT \\* FROM `roles`").
		WillReturnRows(sqlmock.NewRows(nil))
	mock.ExpectExec("INSERT INTO `roles`").
		WillReturnResult(sqlmock.NewResult(1, 1))
	// 第一个 role_permissions 检查：count=0 → insert 失败
	mock.ExpectQuery("SELECT count\\(\\*\\) FROM `role_permissions`").
		WillReturnRows(sqlmock.NewRows([]string{"count(*)"}).AddRow(0))
	mock.ExpectExec("INSERT INTO role_permissions \\(role_id, permission_id\\)").
		WillReturnError(errors.New("assign boom"))
	mock.ExpectRollback()

	err := RunSeed(db)
	require.ErrorContains(t, err, "assign permission failed")
}

func TestRunSeed_AdminUserCreateError(t *testing.T) {
	t.Setenv("ADMIN_PASSWORD", "secret123")

	db, mock := newMockGormDB(t)
	mock.ExpectBegin()
	for range basePermissions() {
		mock.ExpectQuery("SELECT \\* FROM `permissions`").
			WillReturnRows(sqlmock.NewRows(nil))
		mock.ExpectExec("INSERT INTO `permissions`").
			WillReturnResult(sqlmock.NewResult(1, 1))
	}
	mock.ExpectQuery("SELECT \\* FROM `roles`").
		WillReturnRows(sqlmock.NewRows(nil))
	mock.ExpectExec("INSERT INTO `roles`").
		WillReturnResult(sqlmock.NewResult(1, 1))
	for range basePermissions() {
		mock.ExpectQuery("SELECT count\\(\\*\\) FROM `role_permissions`").
			WillReturnRows(sqlmock.NewRows([]string{"count(*)"}).AddRow(0))
		mock.ExpectExec("INSERT INTO role_permissions \\(role_id, permission_id\\)").
			WillReturnResult(sqlmock.NewResult(0, 1))
	}
	mock.ExpectQuery("SELECT \\* FROM `users`").
		WillReturnRows(sqlmock.NewRows(nil))
	mock.ExpectExec("INSERT INTO `users`").
		WillReturnError(errors.New("user boom"))
	mock.ExpectRollback()

	err := RunSeed(db)
	require.ErrorContains(t, err, "seed admin user failed")
}

func TestRunSeed_AssignAdminRoleError(t *testing.T) {
	t.Setenv("ADMIN_PASSWORD", "secret123")

	db, mock := newMockGormDB(t)
	mock.ExpectBegin()
	for range basePermissions() {
		mock.ExpectQuery("SELECT \\* FROM `permissions`").
			WillReturnRows(sqlmock.NewRows(nil))
		mock.ExpectExec("INSERT INTO `permissions`").
			WillReturnResult(sqlmock.NewResult(1, 1))
	}
	mock.ExpectQuery("SELECT \\* FROM `roles`").
		WillReturnRows(sqlmock.NewRows(nil))
	mock.ExpectExec("INSERT INTO `roles`").
		WillReturnResult(sqlmock.NewResult(1, 1))
	for range basePermissions() {
		mock.ExpectQuery("SELECT count\\(\\*\\) FROM `role_permissions`").
			WillReturnRows(sqlmock.NewRows([]string{"count(*)"}).AddRow(0))
		mock.ExpectExec("INSERT INTO role_permissions \\(role_id, permission_id\\)").
			WillReturnResult(sqlmock.NewResult(0, 1))
	}
	mock.ExpectQuery("SELECT \\* FROM `users`").
		WillReturnRows(sqlmock.NewRows(nil))
	mock.ExpectExec("INSERT INTO `users`").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec("INSERT INTO user_roles \\(user_id, role_id\\)").
		WillReturnError(errors.New("userrole boom"))
	mock.ExpectRollback()

	err := RunSeed(db)
	require.ErrorContains(t, err, "assign admin role failed")
}

func TestSeedCasbinPolicies_CreateEnforcerError(t *testing.T) {
	// sqlmock DB 上 gormadapter 建表失败 → NewEnforcer 报错
	db, _ := newMockGormDB(t)
	err := SeedCasbinPolicies(db)
	require.ErrorContains(t, err, "create enforcer")
}

func TestRunSeedWithCasbin_SeedError(t *testing.T) {
	t.Setenv("ADMIN_PASSWORD", "")
	err := RunSeedWithCasbin(nil)
	require.ErrorContains(t, err, "ADMIN_PASSWORD is required")
}

// TestRunSeedWithCasbin 用 sqlite 内存库端到端跑完整种子（含 casbin 策略）。
// 需要 cwd 切到项目根以定位 conf/rbac_model.conf。
func TestRunSeedWithCasbin(t *testing.T) {
	t.Chdir(repoRoot(t))
	t.Setenv("ADMIN_PASSWORD", "secret123")

	db := newSeedSqliteDB(t)
	require.NoError(t, db.AutoMigrate(&domain.Permission{}, &domain.Role{}, &userdomain.User{}))
	require.NoError(t, db.Exec("CREATE TABLE role_permissions (role_id INTEGER, permission_id INTEGER)").Error)
	require.NoError(t, db.Exec("CREATE TABLE user_roles (user_id INTEGER, role_id INTEGER)").Error)

	require.NoError(t, RunSeedWithCasbin(db))

	// 管理员已创建
	var admin userdomain.User
	require.NoError(t, db.Where("username = ?", "admin").First(&admin).Error)
	assert.Equal(t, int8(1), admin.Status)

	// 权限均已落库
	var permCount int64
	require.NoError(t, db.Table("permissions").Count(&permCount).Error)
	assert.Equal(t, int64(len(basePermissions())), permCount)

	// 角色-权限关系已落库
	var rpCount int64
	require.NoError(t, db.Table("role_permissions").Count(&rpCount).Error)
	assert.Equal(t, int64(len(basePermissions())), rpCount)

	// 幂等：再次执行不报错
	require.NoError(t, RunSeed(db))
}

func TestBasePermissionsCoverBusinessRoutes(t *testing.T) {
	required := []struct {
		resource string
		action   string
	}{
		{"/api/v1/users", "GET"},
		{"/api/v1/users", "POST"},
		{"/api/v1/users/*", "GET"},
		{"/api/v1/users/*", "PUT"},
		{"/api/v1/users/*", "DELETE"},
		{"/api/v1/roles", "GET"},
		{"/api/v1/roles", "POST"},
		{"/api/v1/roles/*", "GET"},
		{"/api/v1/roles/*", "PUT"},
		{"/api/v1/roles/*", "DELETE"},
		{"/api/v1/roles/*/permissions", "POST"},
		{"/api/v1/permissions", "GET"},
		{"/api/v1/permissions", "POST"},
		{"/api/v1/permissions/*", "GET"},
		{"/api/v1/permissions/*", "PUT"},
		{"/api/v1/permissions/*", "DELETE"},
		{"/api/v1/audits", "GET"},
		{"/api/v1/audits/*", "GET"},
	}
	got := make(map[string]bool)
	for _, permission := range basePermissions() {
		got[permission.Resource+" "+permission.Action] = true
	}
	for _, item := range required {
		if !got[item.resource+" "+item.action] {
			t.Fatalf("missing permission %s %s", item.action, item.resource)
		}
	}
}
