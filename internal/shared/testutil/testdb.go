package testutil

import (
	"context"
	"fmt"
	"net"
	"os"
	"strconv"
	"testing"
	"time"

	"jimu/internal/config"
	"jimu/internal/platform/db"

	"github.com/pressly/goose/v3"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// TestDB 测试数据库连接
type TestDB struct {
	*gorm.DB
}

// NewTestDB 创建测试数据库连接
func NewTestDB(cfg config.DBConfig) (*TestDB, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.Database)

	gdb, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to connect test database: %w", err)
	}

	return &TestDB{DB: gdb}, nil
}

// mysqlEnvDBConfig 从环境变量读取 MySQL 测试配置（CI 通过 services.mariadb 提供）
func mysqlEnvDBConfig() config.DBConfig {
	cfg := config.DBConfig{
		Host:     os.Getenv("DB_HOST"),
		Port:     3306,
		User:     os.Getenv("DB_USER"),
		Password: os.Getenv("DB_PASSWORD"),
		Database: os.Getenv("DB_NAME"),
	}
	if cfg.Host == "" {
		cfg.Host = "127.0.0.1"
	}
	if cfg.User == "" {
		cfg.User = "root"
	}
	if cfg.Database == "" {
		cfg.Database = "jimu_test"
	}
	if p := os.Getenv("DB_PORT"); p != "" {
		if n, err := strconv.Atoi(p); err == nil {
			cfg.Port = n
		}
	}
	return cfg
}

// mysqlReachable 探测 MySQL 端口是否可连接
func mysqlReachable(host string, port int) bool {
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", host, port), 2*time.Second)
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}

// SkipUnlessMysql 返回真实 MySQL 测试连接；数据库不可达时跳过测试。
// 供 env-gated 集成测试使用：CI 的 mariadb service 满足条件，本地无 DB 时自动跳过。
func SkipUnlessMysql(t *testing.T) *TestDB {
	t.Helper()
	cfg := mysqlEnvDBConfig()
	if !mysqlReachable(cfg.Host, cfg.Port) {
		t.Skipf("mysql %s:%d unreachable; skipping integration test", cfg.Host, cfg.Port)
	}
	tdb, err := NewTestDB(cfg)
	if err != nil {
		t.Skipf("connect mysql failed: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	sqlDB, err := tdb.DB.DB()
	if err != nil {
		t.Skipf("get sql.DB failed: %v", err)
	}
	if err := sqlDB.PingContext(ctx); err != nil {
		t.Skipf("mysql ping failed: %v", err)
	}
	return tdb
}

// NewTestDBWithPool 创建带连接池配置的测试数据库连接
func NewTestDBWithPool(cfg config.DBConfig) (*TestDB, error) {
	gdb, err := db.ConnectWithRetry(cfg, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to connect test database: %w", err)
	}
	return &TestDB{DB: gdb}, nil
}

// Migrate 执行迁移
func (tdb *TestDB) Migrate() error {
	sqlDB, err := tdb.DB.DB()
	if err != nil {
		return err
	}

	if err := goose.SetDialect("mysql"); err != nil {
		return err
	}

	return goose.Up(sqlDB, db.MigrationDir())
}

// Reset 清空所有表数据（保留表结构）
func (tdb *TestDB) Reset(models ...interface{}) error {
	for _, model := range models {
		if err := tdb.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(model).Error; err != nil {
			return fmt.Errorf("failed to reset table: %w", err)
		}
	}
	return nil
}

// Truncate 截断表（更快）
func (tdb *TestDB) Truncate(tableName string) error {
	return tdb.Exec(fmt.Sprintf("TRUNCATE TABLE %s", tableName)).Error
}

// Close 关闭连接
func (tdb *TestDB) Close() error {
	sqlDB, err := tdb.DB.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}
