package testutil

import (
	"context"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"jimu/internal/config"
	"jimu/internal/platform/db"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// TestDB 测试数据库连接
type TestDB struct {
	*gorm.DB
	cfg config.DBConfig
}

// NewTestDB 创建测试数据库连接
func NewTestDB(cfg config.DBConfig) (*TestDB, error) {
	gdb, err := openByDriver(cfg)
	if err != nil {
		return nil, err
	}
	return &TestDB{DB: gdb, cfg: cfg}, nil
}

func openByDriver(cfg config.DBConfig) (*gorm.DB, error) {
	switch strings.ToLower(cfg.Driver) {
	case "postgres", "postgresql":
		return db.New(cfg, nil)
	case "", "mysql", "mariadb":
		dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
			cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.Database)
		return gorm.Open(mysql.Open(dsn), &gorm.Config{})
	default:
		return nil, fmt.Errorf("unsupported db driver: %s", cfg.Driver)
	}
}

// envDBConfig 从环境变量读取测试数据库配置（CI 通过 services.mariadb 提供，DB_DRIVER 可切 postgres）
func envDBConfig() config.DBConfig {
	cfg := config.DBConfig{
		Driver:   os.Getenv("DB_DRIVER"),
		Host:     os.Getenv("DB_HOST"),
		Port:     3306,
		User:     os.Getenv("DB_USER"),
		Password: os.Getenv("DB_PASSWORD"),
		Database: os.Getenv("DB_NAME"),
	}
	if cfg.Driver == "" {
		cfg.Driver = "mysql"
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

// defaultDBPort 返回 driver 对应的默认端口
func defaultDBPort(driver string) int {
	if strings.HasPrefix(driver, "postgres") {
		return 5432
	}
	return 3306
}

// dbReachable 探测数据库端口是否可连接
func dbReachable(host string, port int) bool {
	d := net.Dialer{Timeout: 2 * time.Second}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	conn, err := d.DialContext(ctx, "tcp", fmt.Sprintf("%s:%d", host, port))
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}

// SkipUnlessDB 返回真实数据库测试连接；数据库不可达时跳过测试。
// 供 env-gated 集成测试使用：CI 的 mariadb/postgres service 满足条件，本地无 DB 时自动跳过。
func SkipUnlessDB(t *testing.T) *TestDB {
	t.Helper()
	cfg := envDBConfig()
	if !dbReachable(cfg.Host, cfg.Port) {
		t.Skipf("db %s:%d unreachable; skipping integration test", cfg.Host, cfg.Port)
	}
	tdb, err := NewTestDB(cfg)
	if err != nil {
		t.Skipf("connect db failed: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	sqlDB, err := tdb.DB.DB()
	if err != nil {
		t.Skipf("get sql.DB failed: %v", err)
	}
	if err := sqlDB.PingContext(ctx); err != nil {
		t.Skipf("db ping failed: %v", err)
	}
	return tdb
}

// SkipUnlessMysql 返回真实 MySQL 测试连接（兼容旧调用方）
func SkipUnlessMysql(t *testing.T) *TestDB {
	t.Helper()
	cfg := envDBConfig()
	cfg.Driver = "mysql"
	if !dbReachable(cfg.Host, cfg.Port) {
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
	return &TestDB{DB: gdb, cfg: cfg}, nil
}

// Migrate 执行迁移（根据 cfg.Driver 选择 dialect 与迁移目录）
func (tdb *TestDB) Migrate() error {
	return db.Migrate(tdb.cfg, "up")
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
