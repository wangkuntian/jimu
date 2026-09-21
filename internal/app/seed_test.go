package app

import (
	"errors"
	"fmt"
	"path/filepath"
	"runtime"
	"sync/atomic"
	"testing"
	"time"

	roledomain "jimu/internal/capabilities/access/domain"
	"jimu/internal/capabilities/catalog"
	tenantdomain "jimu/internal/capabilities/tenant/domain"
	userdomain "jimu/internal/capabilities/user/domain"
	"jimu/internal/contract"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/mysql"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// newMockGormDB 用 sqlmock 构建 gorm.DB（跳过版本初始化，避免真实连接）
func newMockGormDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock) {
	t.Helper()
	sqlDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	dialector := mysql.New(mysql.Config{Conn: sqlDB, SkipInitializeWithVersion: true})
	gormDB, err := gorm.Open(dialector, &gorm.Config{})
	require.NoError(t, err)
	t.Cleanup(func() { _ = sqlDB.Close() })
	return gormDB, mock
}

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

// migrateSeedTables 建 RunSeed 触达的全部表（含 user_roles / role_permissions 原始 SQL 表）。
func migrateSeedTables(t *testing.T, db *gorm.DB) {
	t.Helper()
	require.NoError(t, db.AutoMigrate(&roledomain.Permission{}, &roledomain.Role{}, &userdomain.User{}, &tenantdomain.Tenant{}, &tenantdomain.Plan{}))
	require.NoError(t, db.Exec("CREATE TABLE role_permissions (role_id INTEGER, permission_id INTEGER)").Error)
	require.NoError(t, db.Exec("CREATE TABLE user_roles (user_id INTEGER, role_id INTEGER)").Error)
}

// seededPermissions 聚合全量清单的权限点（模拟 RunSeed 对 caps 的聚合）。
func seededPermissions() []contract.Permission {
	var out []contract.Permission
	for _, d := range catalog.All() {
		out = append(out, d.Permissions...)
	}
	return out
}

// repoRoot 项目根目录：按本文件源码路径向上两级定位，
// 不依赖工作目录（迁移已迁入能力目录，顶层 migrations/ 不复存在）。
func repoRoot(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	require.True(t, ok, "runtime.Caller failed")
	// internal/app/seed_test.go → 仓库根（向上两级）
	return filepath.Clean(filepath.Join(filepath.Dir(thisFile), "..", ".."))
}

// expectDefaultTenantQuery 编排默认租户 FirstOrCreate 的 SELECT+INSERT 预期
func expectDefaultTenantQuery(mock sqlmock.Sqlmock) {
	mock.ExpectQuery("SELECT \\* FROM `tenants`").
		WillReturnRows(sqlmock.NewRows(nil))
	mock.ExpectExec("INSERT INTO `tenants`").
		WillReturnResult(sqlmock.NewResult(1, 1))
}

// expectFreePlanQuery 编排内置套餐 FirstOrCreate 的 SELECT+INSERT 预期
func expectFreePlanQuery(mock sqlmock.Sqlmock) {
	mock.ExpectQuery("SELECT \\* FROM `tenant_plans`").
		WillReturnRows(sqlmock.NewRows(nil))
	mock.ExpectExec("INSERT INTO `tenant_plans`").
		WillReturnResult(sqlmock.NewResult(1, 1))
}

// expectRunSeedQueries 为 RunSeed 全流程编排 sqlmock 预期：
// BEGIN → 租户 SELECT+INSERT → 套餐 SELECT+INSERT → N×(权限 SELECT+INSERT) → 角色 SELECT+INSERT →
// N×(role_permissions count+INSERT) → 管理员 SELECT+INSERT → user_roles INSERT → COMMIT
func expectRunSeedQueries(mock sqlmock.Sqlmock) {
	mock.ExpectBegin()

	expectDefaultTenantQuery(mock)
	expectFreePlanQuery(mock)

	for range seededPermissions() {
		mock.ExpectQuery("SELECT \\* FROM `permissions`").
			WillReturnRows(sqlmock.NewRows(nil))
		mock.ExpectExec("INSERT INTO `permissions`").
			WillReturnResult(sqlmock.NewResult(1, 1))
	}

	mock.ExpectQuery("SELECT \\* FROM `roles`").
		WillReturnRows(sqlmock.NewRows(nil))
	mock.ExpectExec("INSERT INTO `roles`").
		WillReturnResult(sqlmock.NewResult(1, 1))

	for range seededPermissions() {
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

	require.NoError(t, RunSeed(db, catalog.All()))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestRunSeed_MissingAdminPassword(t *testing.T) {
	t.Setenv("ADMIN_PASSWORD", "")
	err := RunSeed(nil, nil)
	require.ErrorContains(t, err, "ADMIN_PASSWORD is required")
}

func TestRunSeed_PermissionQueryError(t *testing.T) {
	t.Setenv("ADMIN_PASSWORD", "secret123")

	db, mock := newMockGormDB(t)
	mock.ExpectBegin()
	expectDefaultTenantQuery(mock)
	expectFreePlanQuery(mock)
	mock.ExpectQuery("SELECT \\* FROM `permissions`").
		WillReturnError(errors.New("select boom"))
	mock.ExpectRollback()

	err := RunSeed(db, catalog.All())
	require.ErrorContains(t, err, "seed permission failed")
}

func TestRunSeed_RoleInsertError(t *testing.T) {
	t.Setenv("ADMIN_PASSWORD", "secret123")

	db, mock := newMockGormDB(t)
	mock.ExpectBegin()
	expectDefaultTenantQuery(mock)
	expectFreePlanQuery(mock)
	for range seededPermissions() {
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

	err := RunSeed(db, catalog.All())
	require.ErrorContains(t, err, "seed admin role failed")
}

func TestRunSeed_AssignPermissionError(t *testing.T) {
	t.Setenv("ADMIN_PASSWORD", "secret123")

	db, mock := newMockGormDB(t)
	mock.ExpectBegin()
	expectDefaultTenantQuery(mock)
	expectFreePlanQuery(mock)
	for range seededPermissions() {
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

	err := RunSeed(db, catalog.All())
	require.ErrorContains(t, err, "assign permission failed")
}

func TestRunSeed_AdminUserCreateError(t *testing.T) {
	t.Setenv("ADMIN_PASSWORD", "secret123")

	db, mock := newMockGormDB(t)
	mock.ExpectBegin()
	expectDefaultTenantQuery(mock)
	expectFreePlanQuery(mock)
	for range seededPermissions() {
		mock.ExpectQuery("SELECT \\* FROM `permissions`").
			WillReturnRows(sqlmock.NewRows(nil))
		mock.ExpectExec("INSERT INTO `permissions`").
			WillReturnResult(sqlmock.NewResult(1, 1))
	}
	mock.ExpectQuery("SELECT \\* FROM `roles`").
		WillReturnRows(sqlmock.NewRows(nil))
	mock.ExpectExec("INSERT INTO `roles`").
		WillReturnResult(sqlmock.NewResult(1, 1))
	for range seededPermissions() {
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

	err := RunSeed(db, catalog.All())
	require.ErrorContains(t, err, "seed admin user failed")
}

func TestRunSeed_AssignAdminRoleError(t *testing.T) {
	t.Setenv("ADMIN_PASSWORD", "secret123")

	db, mock := newMockGormDB(t)
	mock.ExpectBegin()
	expectDefaultTenantQuery(mock)
	expectFreePlanQuery(mock)
	for range seededPermissions() {
		mock.ExpectQuery("SELECT \\* FROM `permissions`").
			WillReturnRows(sqlmock.NewRows(nil))
		mock.ExpectExec("INSERT INTO `permissions`").
			WillReturnResult(sqlmock.NewResult(1, 1))
	}
	mock.ExpectQuery("SELECT \\* FROM `roles`").
		WillReturnRows(sqlmock.NewRows(nil))
	mock.ExpectExec("INSERT INTO `roles`").
		WillReturnResult(sqlmock.NewResult(1, 1))
	for range seededPermissions() {
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

	err := RunSeed(db, catalog.All())
	require.ErrorContains(t, err, "assign admin role failed")
}

// TestRunSeed_RolePermissionCountError 验证 Count 查询失败时报错回滚（不再静默 count=0 导致重复插入）
func TestRunSeed_RolePermissionCountError(t *testing.T) {
	t.Setenv("ADMIN_PASSWORD", "secret123")

	db, mock := newMockGormDB(t)
	mock.ExpectBegin()
	expectDefaultTenantQuery(mock)
	expectFreePlanQuery(mock)
	for range seededPermissions() {
		mock.ExpectQuery("SELECT \\* FROM `permissions`").
			WillReturnRows(sqlmock.NewRows(nil))
		mock.ExpectExec("INSERT INTO `permissions`").
			WillReturnResult(sqlmock.NewResult(1, 1))
	}
	mock.ExpectQuery("SELECT \\* FROM `roles`").
		WillReturnRows(sqlmock.NewRows(nil))
	mock.ExpectExec("INSERT INTO `roles`").
		WillReturnResult(sqlmock.NewResult(1, 1))
	// 第一个 role_permissions count 查询失败
	mock.ExpectQuery("SELECT count\\(\\*\\) FROM `role_permissions`").
		WillReturnError(errors.New("count boom"))
	mock.ExpectRollback()

	err := RunSeed(db, catalog.All())
	require.ErrorContains(t, err, "check role permission failed")
}

func TestSeedCasbinPolicies_CreateEnforcerError(t *testing.T) {
	// sqlmock DB 上 gormadapter 建表失败 → NewEnforcer 报错
	db, _ := newMockGormDB(t)
	err := SeedCasbinPolicies(db)
	require.ErrorContains(t, err, "create enforcer")
}

func TestRunSeedWithCasbin_SeedError(t *testing.T) {
	t.Setenv("ADMIN_PASSWORD", "")
	err := RunSeedWithCasbin(nil, nil)
	require.ErrorContains(t, err, "ADMIN_PASSWORD is required")
}

// TestRunSeedWithCasbin 用 sqlite 内存库端到端跑完整种子（含 casbin 策略）。
// 需要 cwd 切到项目根以定位 conf/rbac_model.conf。
func TestRunSeedWithCasbin(t *testing.T) {
	t.Chdir(repoRoot(t))
	t.Setenv("ADMIN_PASSWORD", "secret123")

	db := newSeedSqliteDB(t)
	migrateSeedTables(t, db)

	require.NoError(t, RunSeedWithCasbin(db, catalog.All()))

	// 管理员已创建
	var admin userdomain.User
	require.NoError(t, db.Where("username = ?", "admin").First(&admin).Error)
	assert.Equal(t, int8(1), admin.Status)

	// 权限均已落库
	var permCount int64
	require.NoError(t, db.Table("permissions").Count(&permCount).Error)
	assert.Equal(t, int64(len(seededPermissions())), permCount)

	// 角色-权限关系已落库
	var rpCount int64
	require.NoError(t, db.Table("role_permissions").Count(&rpCount).Error)
	assert.Equal(t, int64(len(seededPermissions())), rpCount)

	// 幂等：再次执行不报错且数据不重复（row count 不变）
	var userCountBefore int64
	require.NoError(t, db.Table("users").Count(&userCountBefore).Error)
	require.NoError(t, RunSeed(db, catalog.All()))

	var permCount2, rpCount2, userCount2 int64
	require.NoError(t, db.Table("permissions").Count(&permCount2).Error)
	require.NoError(t, db.Table("role_permissions").Count(&rpCount2).Error)
	require.NoError(t, db.Table("users").Count(&userCount2).Error)
	assert.Equal(t, int64(len(seededPermissions())), permCount2, "permissions 不应重复插入")
	assert.Equal(t, int64(len(seededPermissions())), rpCount2, "role_permissions 不应重复插入")
	assert.Equal(t, userCountBefore, userCount2, "admin 用户不应重复创建")
}

// TestRunSeed_SkipsDisabledCapabilityPermissions 验证权限点来自启用集聚合：
// 只启用 user+access 时，audit/tenant/admin 的权限点不得落库。
func TestRunSeed_SkipsDisabledCapabilityPermissions(t *testing.T) {
	t.Setenv("ADMIN_PASSWORD", "secret123")

	db := newSeedSqliteDB(t)
	migrateSeedTables(t, db)

	// 只启用 user+access：audit/tenant/console 的权限点不得出现
	caps, err := catalog.Resolve([]string{"user", "access"})
	require.NoError(t, err)
	require.NoError(t, RunSeed(db, caps))

	var count int64
	require.NoError(t, db.Table("permissions").Count(&count).Error)
	// user 5 + access 11 = 16；audit/tenant/admin 的权限点不在（access 闭包自动带 user）
	assert.Equal(t, int64(16), count)
}
