package application

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"jimu/internal/capabilities/access/domain"
	"jimu/internal/config"
	"jimu/internal/contract"
	"jimu/internal/kernel/tenant"
	"jimu/internal/shared/testutil"

	mysqlDriver "github.com/go-sql-driver/mysql"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func newConcurrencyTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	host := envOr("DB_HOST", "127.0.0.1")
	port := envOr("DB_PORT", "3306")
	user := envOr("DB_USER", "root")
	password := os.Getenv("DB_PASSWORD")
	if _, err := strconv.Atoi(port); err != nil {
		t.Skipf("invalid DB_PORT %q", port)
	}

	adminConfig := mysqlDriver.NewConfig()
	adminConfig.User = user
	adminConfig.Passwd = password
	adminConfig.Net = "tcp"
	adminConfig.Addr = host + ":" + port
	adminDSN := adminConfig.FormatDSN()
	admin, err := sql.Open("mysql", adminDSN)
	if err != nil {
		t.Skipf("open mysql admin connection: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := admin.PingContext(ctx); err != nil {
		t.Skipf("mysql unavailable: %v", err)
	}

	dbName := "jimu_superadmin_race_" + strconv.FormatInt(time.Now().UnixNano(), 36)
	if _, err := admin.ExecContext(ctx, "CREATE DATABASE `"+dbName+"` CHARACTER SET utf8mb4"); err != nil {
		t.Skipf("create isolated database: %v", err)
	}
	t.Cleanup(func() {
		_, _ = admin.Exec("DROP DATABASE IF EXISTS `" + dbName + "`")
		_ = admin.Close()
	})

	portNumber, _ := strconv.Atoi(port)
	repoRoot := testutil.MustRepoRoot()
	testutil.SetMigrateCaps([]contract.Descriptor{
		{Name: "user", Migrations: os.DirFS(filepath.Join(repoRoot, "internal", "capabilities", "user"))},
		{Name: "access", Migrations: os.DirFS(filepath.Join(repoRoot, "internal", "capabilities", "access"))},
		{Name: "tenant", Migrations: os.DirFS(filepath.Join(repoRoot, "internal", "capabilities", "tenant"))},
	})
	t.Cleanup(func() { testutil.SetMigrateCaps(nil) })
	tdb, err := testutil.NewTestDB(config.DBConfig{
		Driver:   "mysql",
		Host:     host,
		Port:     portNumber,
		User:     user,
		Password: password,
		Database: dbName,
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		if sqlDB, err := tdb.DB.DB(); err == nil {
			_ = sqlDB.Close()
		}
	})
	require.NoError(t, tdb.Migrate())
	return tdb.DB
}

func envOr(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func TestLastPlatformSuperAdminIsSerializedOnMySQL(t *testing.T) {
	db := newConcurrencyTestDB(t)
	superRole := domain.Role{
		Name: "平台超级管理员", Code: contract.RoleCodePlatformSuperAdmin,
		Scope: contract.RoleScopePlatform, IsSystem: true, TenantID: tenant.DefaultTenantID, Status: 1,
	}
	viewer := domain.Role{Name: "viewer", Scope: contract.RoleScopeTenant, TenantID: tenant.DefaultTenantID, Status: 1}
	require.NoError(t, db.Create(&superRole).Error)
	require.NoError(t, db.Create(&viewer).Error)

	users := []struct {
		username string
	}{{"race-a"}, {"race-b"}}
	var userIDs []uint64
	for i, item := range users {
		userID := uint64(time.Now().UnixNano()) + uint64(i)
		require.NoError(t, db.Exec(
			"INSERT INTO users (id, username, password, status, tenant_id, created_at, updated_at) VALUES (?, ?, ?, 1, ?, NOW(), NOW())",
			userID, item.username, "hash", tenant.DefaultTenantID,
		).Error)
		userIDs = append(userIDs, userID)
		require.NoError(t, db.Exec("INSERT INTO user_roles (user_id, role_id) VALUES (?, ?)", userID, superRole.ID).Error)
	}

	service := NewUserRoleService(db)
	ctx := tenant.WithPlatformScope(tenant.WithTenant(context.Background(), tenant.DefaultTenantID))
	start := make(chan struct{})
	results := make(chan error, 2)
	for _, userID := range userIDs {
		userID := userID
		go func() {
			<-start
			results <- service.AssignRoleIDs(ctx, userID, nil)
		}()
	}
	close(start)

	var success, conflict int
	for range userIDs {
		err := <-results
		if err == nil {
			success++
			continue
		}
		if userRoleAppCode(err) == 1009 {
			conflict++
			continue
		}
		t.Fatalf("unexpected assign error: %v", err)
	}
	assert.Equal(t, 1, success)
	assert.Equal(t, 1, conflict)

	var activeCount int64
	require.NoError(t, db.Table("users").
		Joins("JOIN user_roles ur ON ur.user_id = users.id").
		Joins("JOIN roles ON roles.id = ur.role_id").
		Where("users.status = 1 AND users.deleted_at IS NULL").
		Where("roles.code = ? AND roles.scope = ? AND roles.status = 1 AND roles.deleted_at IS NULL",
			contract.RoleCodePlatformSuperAdmin, contract.RoleScopePlatform).
		Distinct("users.id").Count(&activeCount).Error)
	assert.Equal(t, int64(1), activeCount)
}
