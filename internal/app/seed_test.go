package app_test

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"jimu/internal/app"
	roledomain "jimu/internal/capabilities/access/domain"
	"jimu/internal/capabilities/catalog"
	tenantdomain "jimu/internal/capabilities/tenant/domain"
	userdomain "jimu/internal/capabilities/user/domain"
	"jimu/internal/contract"
	"jimu/internal/shared/testutil"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/mysql"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
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
// 可选传入 GORM logger，用于断言探针不会经生产 logger 打出 error 行。
func newSeedSqliteDB(t *testing.T, gormLog ...gormlogger.Interface) *gorm.DB {
	t.Helper()
	seq := atomic.AddUint64(&seedDBSeq, 1)
	dsn := fmt.Sprintf("file:seed_%d_%d?mode=memory&cache=shared", time.Now().UnixNano(), seq)
	cfg := &gorm.Config{}
	if len(gormLog) > 0 && gormLog[0] != nil {
		cfg.Logger = gormLog[0]
	}
	db, err := gorm.Open(sqlite.Open(dsn), cfg)
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

func seededPermissionSet() map[contract.ProvisioningPermission]bool {
	out := make(map[contract.ProvisioningPermission]bool)
	for _, permission := range seededPermissions() {
		out[contract.ProvisioningPermission{Resource: permission.Resource, Action: permission.Action}] = true
	}
	return out
}

func platformRolePermissionCount(spec contract.SystemRoleSpec) int {
	if spec.AllPermissions {
		return len(seededPermissions())
	}
	set := seededPermissionSet()
	count := 0
	for _, permission := range spec.Permissions {
		if set[permission] {
			count++
		}
	}
	return count
}

func expectedSeedRolePermissionCount() int {
	total := 0
	for _, spec := range contract.PlatformSystemRoles() {
		total += platformRolePermissionCount(spec)
	}
	return total
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

// expectRunSeedQueries 为 RunSeed 全流程编排 sqlmock 预期。
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

	expectPlatformSystemRoles(mock)

	mock.ExpectQuery("SELECT \\* FROM `users`").
		WillReturnRows(sqlmock.NewRows(nil))
	mock.ExpectExec("INSERT INTO `users`").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectQuery("SELECT count\\(\\*\\) FROM `user_roles`").
		WillReturnRows(sqlmock.NewRows([]string{"count(*)"}).AddRow(0))
	mock.ExpectExec("INSERT INTO user_roles \\(user_id, role_id\\)").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery("SELECT COUNT").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	mock.ExpectCommit()
}

func expectPlatformSystemRoles(mock sqlmock.Sqlmock) {
	for i, spec := range contract.PlatformSystemRoles() {
		mock.ExpectQuery("SELECT \\* FROM `roles`").
			WillReturnRows(sqlmock.NewRows(nil))
		mock.ExpectExec("INSERT INTO `roles`").
			WillReturnResult(sqlmock.NewResult(int64(i+1), 1))
		mock.ExpectExec("DELETE FROM role_permissions WHERE role_id").
			WillReturnResult(sqlmock.NewResult(0, 0))
		for range platformRolePermissionCount(spec) {
			mock.ExpectExec("INSERT INTO role_permissions \\(role_id, permission_id\\)").
				WillReturnResult(sqlmock.NewResult(0, 1))
		}
	}
}

func TestRunSeed_HappyPath(t *testing.T) {
	t.Setenv("ADMIN_PASSWORD_FILE", "")
	t.Setenv("ADMIN_PASSWORD", "secret123")

	sqlDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = sqlDB.Close() }()

	db, err := gorm.Open(mysql.New(mysql.Config{Conn: sqlDB, SkipInitializeWithVersion: true}), &gorm.Config{})
	require.NoError(t, err)

	expectRunSeedQueries(mock)

	require.NoError(t, app.RunSeed(db, catalog.All()))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestRunSeed_UsesAdminPasswordFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "admin_password")
	require.NoError(t, os.WriteFile(path, []byte("secret-from-file\n"), 0o600))
	t.Setenv("ADMIN_PASSWORD_FILE", path)
	t.Setenv("ADMIN_PASSWORD", "")

	db, mock := newMockGormDB(t)
	expectRunSeedQueries(mock)

	require.NoError(t, app.RunSeed(db, catalog.All()))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestRunSeed_MissingAdminPassword(t *testing.T) {
	t.Setenv("ADMIN_PASSWORD_FILE", "")
	t.Setenv("ADMIN_PASSWORD", "")
	err := app.RunSeed(nil, nil)
	require.ErrorContains(t, err, "ADMIN_PASSWORD or ADMIN_PASSWORD_FILE is required")
}

func TestRunSeed_PermissionQueryError(t *testing.T) {
	t.Setenv("ADMIN_PASSWORD_FILE", "")
	t.Setenv("ADMIN_PASSWORD", "secret123")

	db, mock := newMockGormDB(t)
	mock.ExpectBegin()
	expectDefaultTenantQuery(mock)
	expectFreePlanQuery(mock)
	mock.ExpectQuery("SELECT \\* FROM `permissions`").
		WillReturnError(errors.New("select boom"))
	mock.ExpectRollback()

	err := app.RunSeed(db, catalog.All())
	require.ErrorContains(t, err, "seed permission failed")
}

func TestRunSeed_RoleInsertError(t *testing.T) {
	t.Setenv("ADMIN_PASSWORD_FILE", "")
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
	mock.ExpectQuery("SELECT \\* FROM `roles`").
		WillReturnRows(sqlmock.NewRows(nil))
	mock.ExpectExec("INSERT INTO `roles`").
		WillReturnError(errors.New("role boom"))
	mock.ExpectRollback()

	err := app.RunSeed(db, catalog.All())
	require.ErrorContains(t, err, "seed system role platform_super_admin failed")
}

func TestRunSeed_AssignPermissionError(t *testing.T) {
	t.Setenv("ADMIN_PASSWORD_FILE", "")
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
	mock.ExpectExec("DELETE FROM role_permissions WHERE role_id").
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec("INSERT INTO role_permissions \\(role_id, permission_id\\)").
		WillReturnError(errors.New("assign boom"))
	mock.ExpectRollback()

	err := app.RunSeed(db, catalog.All())
	require.ErrorContains(t, err, "assign permissions for platform_super_admin failed")
}

func TestRunSeed_AdminUserCreateError(t *testing.T) {
	t.Setenv("ADMIN_PASSWORD_FILE", "")
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
	expectPlatformSystemRoles(mock)
	mock.ExpectQuery("SELECT \\* FROM `users`").
		WillReturnRows(sqlmock.NewRows(nil))
	mock.ExpectExec("INSERT INTO `users`").
		WillReturnError(errors.New("user boom"))
	mock.ExpectRollback()

	err := app.RunSeed(db, catalog.All())
	require.ErrorContains(t, err, "seed admin user failed")
}

func TestRunSeed_AssignAdminRoleError(t *testing.T) {
	t.Setenv("ADMIN_PASSWORD_FILE", "")
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
	expectPlatformSystemRoles(mock)
	mock.ExpectQuery("SELECT \\* FROM `users`").
		WillReturnRows(sqlmock.NewRows(nil))
	mock.ExpectExec("INSERT INTO `users`").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectQuery("SELECT count\\(\\*\\) FROM `user_roles`").
		WillReturnRows(sqlmock.NewRows([]string{"count(*)"}).AddRow(0))
	mock.ExpectExec("INSERT INTO user_roles \\(user_id, role_id\\)").
		WillReturnError(errors.New("userrole boom"))
	mock.ExpectRollback()

	err := app.RunSeed(db, catalog.All())
	require.ErrorContains(t, err, "assign admin role failed")
}

// TestRunSeed_RolePermissionDeleteError 验证清理旧系统角色权限失败时回滚。
func TestRunSeed_RolePermissionDeleteError(t *testing.T) {
	t.Setenv("ADMIN_PASSWORD_FILE", "")
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
	mock.ExpectExec("DELETE FROM role_permissions WHERE role_id").
		WillReturnError(errors.New("delete boom"))
	mock.ExpectRollback()

	err := app.RunSeed(db, catalog.All())
	require.ErrorContains(t, err, "clear role permissions failed")
}

func TestSeedCasbinPolicies_CreateEnforcerError(t *testing.T) {
	// sqlmock DB 上 gormadapter 建表失败 → NewEnforcer 报错
	db, _ := newMockGormDB(t)
	err := app.SeedCasbinPolicies(db)
	require.ErrorContains(t, err, "create enforcer")
}

func TestRunSeedWithCasbin_SeedError(t *testing.T) {
	t.Setenv("ADMIN_PASSWORD_FILE", "")
	t.Setenv("ADMIN_PASSWORD", "")
	err := app.RunSeedWithCasbin(nil, nil)
	require.ErrorContains(t, err, "ADMIN_PASSWORD or ADMIN_PASSWORD_FILE is required")
}

// TestRunSeedWithCasbin 用 sqlite 内存库端到端跑完整种子（含 casbin 策略）。
// 需要 cwd 切到项目根以定位 conf/rbac_model.conf。
func TestRunSeedWithCasbin(t *testing.T) {
	t.Chdir(testutil.RepoRoot(t))
	t.Setenv("ADMIN_PASSWORD_FILE", "")
	t.Setenv("ADMIN_PASSWORD", "secret123")

	db := newSeedSqliteDB(t)
	migrateSeedTables(t, db)

	require.NoError(t, app.RunSeedWithCasbin(db, catalog.All()))

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
	assert.Equal(t, int64(expectedSeedRolePermissionCount()), rpCount)

	// 幂等：再次执行不报错且数据不重复（row count 不变）
	var userCountBefore int64
	require.NoError(t, db.Table("users").Count(&userCountBefore).Error)
	require.NoError(t, app.RunSeed(db, catalog.All()))

	var permCount2, rpCount2, userCount2 int64
	require.NoError(t, db.Table("permissions").Count(&permCount2).Error)
	require.NoError(t, db.Table("role_permissions").Count(&rpCount2).Error)
	require.NoError(t, db.Table("users").Count(&userCount2).Error)
	assert.Equal(t, int64(len(seededPermissions())), permCount2, "permissions 不应重复插入")
	assert.Equal(t, int64(expectedSeedRolePermissionCount()), rpCount2, "role_permissions 不应重复插入")
	assert.Equal(t, userCountBefore, userCount2, "admin 用户不应重复创建")
}

func TestRunSeedCreatesPlatformSystemRoles(t *testing.T) {
	t.Setenv("ADMIN_PASSWORD_FILE", "")
	t.Setenv("ADMIN_PASSWORD", "secret123")

	db := newSeedSqliteDB(t)
	migrateSeedTables(t, db)
	require.NoError(t, app.RunSeed(db, catalog.All()))

	var roles []roledomain.Role
	require.NoError(t, db.Where("tenant_id = ? AND code <> ''", 1).Order("code").Find(&roles).Error)
	require.Len(t, roles, 2)
	byCode := map[string]roledomain.Role{}
	for _, role := range roles {
		byCode[role.Code] = role
	}
	superAdmin := byCode[contract.RoleCodePlatformSuperAdmin]
	auditor := byCode[contract.RoleCodePlatformAuditor]
	require.NotZero(t, superAdmin.ID)
	require.NotZero(t, auditor.ID)
	assert.Equal(t, contract.RoleScopePlatform, superAdmin.Scope)
	assert.Equal(t, contract.RoleScopePlatform, auditor.Scope)
	assert.True(t, superAdmin.IsSystem)
	assert.True(t, auditor.IsSystem)

	var superPermissionCount int64
	require.NoError(t, db.Table("role_permissions").Where("role_id = ?", superAdmin.ID).Count(&superPermissionCount).Error)
	assert.Equal(t, int64(len(seededPermissions())), superPermissionCount)

	var auditorRows []struct {
		Resource string
		Action   string
	}
	require.NoError(t, db.Table("role_permissions rp").
		Select("p.resource, p.action").
		Joins("JOIN permissions p ON p.id = rp.permission_id").
		Where("rp.role_id = ?", auditor.ID).
		Order("p.resource, p.action").
		Scan(&auditorRows).Error)
	auditorPermissions := make([]string, 0, len(auditorRows))
	for _, row := range auditorRows {
		auditorPermissions = append(auditorPermissions, row.Resource+" "+row.Action)
	}
	assert.Contains(t, auditorPermissions, "/api/v1/tenants GET")
	assert.Contains(t, auditorPermissions, "/api/v1/audits GET")
	assert.NotContains(t, auditorPermissions, "/api/v1/users POST")

	var tenantSystemCount int64
	require.NoError(t, db.Model(&roledomain.Role{}).
		Where("tenant_id = ? AND code IN ?", 1, []string{contract.RoleCodeTenantAdmin, contract.RoleCodeTenantMember}).
		Count(&tenantSystemCount).Error)
	assert.Zero(t, tenantSystemCount)

	var admin userdomain.User
	require.NoError(t, db.Where("username = ?", "admin").First(&admin).Error)
	var adminRoles []string
	require.NoError(t, db.Table("user_roles ur").
		Select("roles.code").
		Joins("JOIN roles ON roles.id = ur.role_id").
		Where("ur.user_id = ?", admin.ID).
		Pluck("roles.code", &adminRoles).Error)
	assert.Equal(t, []string{contract.RoleCodePlatformSuperAdmin}, adminRoles)
}

func TestRunSeedConvergesSystemRolePermissions(t *testing.T) {
	t.Setenv("ADMIN_PASSWORD_FILE", "")
	t.Setenv("ADMIN_PASSWORD", "secret123")

	db := newSeedSqliteDB(t)
	migrateSeedTables(t, db)
	require.NoError(t, app.RunSeed(db, catalog.All()))

	var auditor roledomain.Role
	require.NoError(t, db.Where("code = ?", contract.RoleCodePlatformAuditor).First(&auditor).Error)
	extra := roledomain.Permission{Name: "额外写权限", Resource: "/api/v1/extra", Action: "POST"}
	require.NoError(t, db.Create(&extra).Error)
	require.NoError(t, db.Exec("INSERT INTO role_permissions (role_id, permission_id) VALUES (?, ?)", auditor.ID, extra.ID).Error)

	require.NoError(t, app.RunSeed(db, catalog.All()))
	var count int64
	require.NoError(t, db.Table("role_permissions").
		Where("role_id = ? AND permission_id = ?", auditor.ID, extra.ID).
		Count(&count).Error)
	assert.Zero(t, count)
}

func TestRunSeedRequiresActivePlatformSuperAdmin(t *testing.T) {
	t.Setenv("ADMIN_PASSWORD_FILE", "")
	t.Setenv("ADMIN_PASSWORD", "secret123")

	for _, tc := range []struct {
		name  string
		setup func(t *testing.T, db *gorm.DB) userdomain.User
	}{
		{
			name: "disabled",
			setup: func(t *testing.T, db *gorm.DB) userdomain.User {
				admin := userdomain.User{Username: "admin", Password: "existing-hash", Status: 1, TenantID: 1}
				require.NoError(t, db.Create(&admin).Error)
				require.NoError(t, db.Model(&userdomain.User{}).Where("id = ?", admin.ID).Update("status", 0).Error)
				admin.Status = 0
				return admin
			},
		},
		{
			name: "deleted",
			setup: func(t *testing.T, db *gorm.DB) userdomain.User {
				admin := userdomain.User{Username: "admin", Password: "existing-hash", Status: 1, TenantID: 1}
				require.NoError(t, db.Create(&admin).Error)
				require.NoError(t, db.Delete(&admin).Error)
				return admin
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			db := newSeedSqliteDB(t)
			migrateSeedTables(t, db)
			admin := tc.setup(t, db)

			err := app.RunSeed(db, catalog.All())
			require.ErrorContains(t, err, "no active platform superadmin")

			var count int64
			require.NoError(t, db.Unscoped().Model(&userdomain.User{}).
				Where("username = ?", "admin").
				Count(&count).Error)
			assert.Equal(t, int64(1), count)

			var old userdomain.User
			require.NoError(t, db.Unscoped().First(&old, admin.ID).Error)
			assert.Equal(t, admin.Status, old.Status)
			assert.Equal(t, admin.DeletedAt.Valid, old.DeletedAt.Valid)
		})
	}
}

// captureLogger 捕获 Errorw 调用，供启动自检断言。
type captureLogger struct {
	messages []string
	fields   []any
}

func (l *captureLogger) Errorw(msg string, keysAndValues ...interface{}) {
	l.messages = append(l.messages, msg)
	l.fields = append(l.fields, keysAndValues...)
}

// TestWarnIfNoAccounts 启动自检：库中一个账号都没有时必须给出可执行的提示；
// 有账号、表不存在、db 为空时都不得报错（容器启动早于迁移是允许的时序）。
func TestWarnIfNoAccounts(t *testing.T) {
	t.Run("empty table warns", func(t *testing.T) {
		db := newSeedSqliteDB(t)
		migrateSeedTables(t, db)
		log := &captureLogger{}

		if !app.WarnIfNoAccounts(db, log) {
			t.Fatal("returned false, want true")
		}
		require.Len(t, log.messages, 1)
		require.Contains(t, log.fields, "hint")
		// 提示必须可执行：同时给出补种与放开注册两条下一步
		hint := fmt.Sprint(log.fields...)
		require.Contains(t, hint, "ADMIN_PASSWORD")
		require.Contains(t, hint, "auth.public_registration")
	})

	t.Run("existing account stays quiet", func(t *testing.T) {
		db := newSeedSqliteDB(t)
		migrateSeedTables(t, db)
		// 走既有种子路径造出 admin，避免依赖模型的 ID 钩子/原始 SQL
		t.Setenv("ADMIN_PASSWORD_FILE", "")
		t.Setenv("ADMIN_PASSWORD", "secret123")
		require.NoError(t, app.RunSeed(db, catalog.All()))
		log := &captureLogger{}

		if app.WarnIfNoAccounts(db, log) {
			t.Fatal("returned true, want false")
		}
		require.Empty(t, log.messages)
	})

	t.Run("unmigrated database stays quiet", func(t *testing.T) {
		db := newSeedSqliteDB(t)
		log := &captureLogger{}

		if app.WarnIfNoAccounts(db, log) {
			t.Fatal("returned true, want false（表不存在时不得误报）")
		}
		require.Empty(t, log.messages)
	})

	// 生产 GORM logger 把**每条**报错语句都打到 error 级（kernel/db/gorm_logger.go 的 Trace），
	// 因此探针在表未建时必须连语句都不发，否则每次启动都会多一行 "db query failed"。
	t.Run("unmigrated database issues no gorm error line", func(t *testing.T) {
		var buf bytes.Buffer
		gormLog := gormlogger.New(log.New(&buf, "", 0), gormlogger.Config{LogLevel: gormlogger.Error, Colorful: false})
		db := newSeedSqliteDB(t, gormLog)
		log := &captureLogger{}

		if app.WarnIfNoAccounts(db, log) {
			t.Fatal("returned true, want false（表不存在时不得误报）")
		}
		require.Empty(t, log.messages)
		require.Empty(t, buf.String(), "探针不得经生产 GORM logger 打出 error 行")
	})

	t.Run("nil logger stays quiet", func(t *testing.T) {
		db := newSeedSqliteDB(t)
		migrateSeedTables(t, db)

		if app.WarnIfNoAccounts(db, nil) {
			t.Fatal("returned true, want false")
		}
	})

	t.Run("nil database stays quiet", func(t *testing.T) {
		log := &captureLogger{}
		if app.WarnIfNoAccounts(nil, log) {
			t.Fatal("returned true, want false")
		}
		require.Empty(t, log.messages)
	})
}

// TestRunSeed_SkipsDisabledCapabilityPermissions 验证权限点来自启用集聚合：
// 只启用 user+access 时，audit/tenant/admin 的权限点不得落库。
func TestRunSeed_SkipsDisabledCapabilityPermissions(t *testing.T) {
	t.Setenv("ADMIN_PASSWORD_FILE", "")
	t.Setenv("ADMIN_PASSWORD", "secret123")

	db := newSeedSqliteDB(t)
	migrateSeedTables(t, db)

	// 只启用 user+access：audit/tenant/console 的权限点不得出现
	caps, err := catalog.Resolve([]string{"user", "access"})
	require.NoError(t, err)
	require.NoError(t, app.RunSeed(db, caps))

	var count int64
	require.NoError(t, db.Table("permissions").Count(&count).Error)
	// user 5 + access 14 = 19；audit/tenant/admin 的权限点不在（access 闭包自动带 user）
	assert.Equal(t, int64(19), count)
}
