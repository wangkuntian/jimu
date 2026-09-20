package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"sort"
	"strconv"
	"strings"
	"time"

	"jimu/internal/config"
	"jimu/internal/contract"
	"jimu/internal/kernel/logger"

	_ "github.com/jackc/pgx/v5/stdlib" // 注册 pgx driver，供 goose postgres 迁移使用
	"github.com/pressly/goose/v3"
	"gorm.io/gorm"
)

// Migrate 执行全能力迁移（兼容旧接口，无重试）。caps 传启用集
// （生产路径 catalog.Resolve 结果，测试路径 catalog.All()）；caps 为 nil 时无迁移可执行。
func Migrate(cfg config.DBConfig, caps []contract.Descriptor, direction string) error {
	return MigrateEnabled(cfg, caps, direction)
}

// MigrateEnabled 按能力拓扑序逐个执行迁移：每个能力用独立 goose Provider
// 与独立版本表 goose_db_version_<capability>，互不干扰；删除能力即删其表与记录。
func MigrateEnabled(cfg config.DBConfig, caps []contract.Descriptor, direction string) error {
	gooseDialect := goose.DialectMySQL
	if cfg.Dialect() == "postgres" {
		gooseDialect = goose.DialectPostgres
	}
	sqlDB, err := openSQLForMigrate(cfg)
	if err != nil {
		return err
	}
	defer func() { _ = sqlDB.Close() }()

	for _, c := range caps {
		if c.Migrations == nil {
			continue
		}
		// 声明了 Migrations 却没有 migrations/ 根属开发期错误：报错而非静默跳过
		// （静默漏迁移）。缺当前方言子目录仍是跳过（能力可在另一方言下无迁移）。
		if _, err := fs.Stat(c.Migrations, "migrations"); err != nil {
			return fmt.Errorf("capability %s migrations: missing migrations/ root: %w", c.Name, err)
		}
		if _, err := fs.Stat(c.Migrations, "migrations/"+cfg.Dialect()); err != nil {
			continue
		}
		fsys, err := fs.Sub(c.Migrations, "migrations/"+cfg.Dialect())
		if err != nil {
			return fmt.Errorf("capability %s migrations: %w", c.Name, err)
		}
		if err := migrateOne(gooseDialect, sqlDB, fsys, "goose_db_version_"+c.Name, direction); err != nil {
			return fmt.Errorf("capability %s: %w", c.Name, err)
		}
	}
	return nil
}

// openSQLForMigrate 按 Driver 配置返回迁移用的 *sql.DB（调用方负责关闭）
func openSQLForMigrate(cfg config.DBConfig) (*sql.DB, error) {
	driver, dsnStr, err := sqlDriverAndDSN(cfg)
	if err != nil {
		return nil, err
	}
	sqlDB, err := sql.Open(driver, dsnStr)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}
	return sqlDB, nil
}

func migrateOne(dialect goose.Dialect, sqlDB *sql.DB, fsys fs.FS, table, direction string) error {
	p, err := goose.NewProvider(dialect, sqlDB, fsys, goose.WithTableName(table))
	if err != nil {
		return fmt.Errorf("create provider: %w", err)
	}
	// 注意：不调用 p.Close()——它会关闭共享的 *sql.DB，导致后续能力拿到已关闭的库；
	// Provider 的连接按操作内部管理，无显式释放需求。
	ctx := context.Background()
	switch direction {
	case "up":
		_, err = p.Up(ctx)
	case "down":
		_, err = p.Down(ctx)
	case "redo":
		if _, err = p.Down(ctx); err == nil {
			_, err = p.Up(ctx)
		}
	case "status":
		err = printStatus(ctx, p, table)
	default:
		return fmt.Errorf("unknown direction: %s", direction)
	}
	return err
}

// printStatus 逐条打印该能力版本表的迁移状态（沿用旧 status 输出风格）
func printStatus(ctx context.Context, p *goose.Provider, table string) error {
	statuses, err := p.Status(ctx)
	if err != nil {
		return err
	}
	for _, s := range statuses {
		state := "Pending"
		if s.State == goose.StateApplied {
			state = "Applied"
		}
		fmt.Printf("%s: %d %s (%s)\n", table, s.Source.Version, s.Source.Path, state)
	}
	return nil
}

// MigrateWithRetry 带重试的数据库迁移
func MigrateWithRetry(cfg config.DBConfig, caps []contract.Descriptor, log *logger.Logger, direction string) error {
	maxRetries := cfg.MaxRetries
	if maxRetries <= 0 {
		maxRetries = 5
	}
	interval := cfg.RetryIntervalSec
	if interval <= 0 {
		interval = 3
	}

	var lastErr error
	for attempt := 1; attempt <= maxRetries; attempt++ {
		if err := Migrate(cfg, caps, direction); err != nil {
			lastErr = err
			if log != nil {
				log.Warnw("retrying database migration",
					"direction", direction,
					"attempt", attempt,
					"max_retries", maxRetries,
					"error", err.Error(),
				)
			}
			time.Sleep(time.Duration(interval) * time.Second)
			continue
		}
		return nil
	}
	return fmt.Errorf("migration %s failed after %d attempts: %w", direction, maxRetries, lastErr)
}

// sqlDriverAndDSN 根据 Driver 配置返回 database/sql driver 名与 DSN
func sqlDriverAndDSN(cfg config.DBConfig) (string, string, error) {
	dialect := strings.ToLower(cfg.Driver)
	switch dialect {
	case "postgres", "postgresql":
		return "pgx", pgDSN(cfg), nil
	case "", "mysql", "mariadb":
		return "mysql", mysqlDSN(cfg), nil
	default:
		return "", "", fmt.Errorf("unsupported db driver: %s", cfg.Driver)
	}
}

func mysqlDSN(cfg config.DBConfig) string {
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.Database)
}

// capabilityMigrationVersions 列出能力 Migrations FS 中指定方言子目录的迁移版本号
// （文件名形如 004_user_totp.sql，取前缀数字），升序返回。目录缺失返回空集；
// 存在无数字前缀的 .sql 文件时报错，避免静默漏迁移。AdoptCapabilities（Task 5）
// 基线登记复用此解析。
func capabilityMigrationVersions(fsys fs.FS, dialect string) ([]int64, error) {
	entries, err := fs.ReadDir(fsys, "migrations/"+dialect)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	versions := make([]int64, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".sql") {
			continue
		}
		name := e.Name()
		// 约定：文件名 = 数字前缀 + "_" + 描述 + ".sql"（如 004_user_totp.sql）
		prefix := name
		if idx := strings.IndexByte(name, '_'); idx > 0 {
			prefix = name[:idx]
		}
		v, perr := strconv.ParseInt(prefix, 10, 64)
		if perr != nil {
			return nil, fmt.Errorf("invalid migration filename %q: %w", name, perr)
		}
		versions = append(versions, v)
	}
	sort.Slice(versions, func(i, j int) bool { return versions[i] < versions[j] })
	return versions, nil
}

// AdoptCapabilities 为存量实例登记各能力版本表基线：读取全局 goose_db_version
// 的最大已应用版本 V，对每个能力把原编号 <= V 的迁移版本直接写入
// goose_db_version_<capability>（is_applied=true，不执行 SQL）。
// 全新库（全局版本表不存在）返回错误并引导先 migrate up。
// 返回 map[能力名]已登记版本号（升序）；未基线任何版本的能力不出现在结果中。
// 全局 goose_db_version 表保留不动（历史记录，新运行器不再读写）。
func AdoptCapabilities(cfg config.DBConfig, caps []contract.Descriptor) (map[string][]int64, error) {
	gooseDialect := goose.DialectMySQL
	if cfg.Dialect() == "postgres" {
		gooseDialect = goose.DialectPostgres
	}
	sqlDB, err := openSQLForMigrate(cfg)
	if err != nil {
		return nil, err
	}
	defer func() { _ = sqlDB.Close() }()

	// 1. 读全局版本表最大版本 V（表不存在 → 引导错误；V=0/空表为合法：不基线任何版本）
	var maxVersion sql.NullInt64
	err = sqlDB.QueryRow("SELECT MAX(version_id) FROM goose_db_version").Scan(&maxVersion)
	if err != nil {
		return nil, fmt.Errorf(
			"global version table goose_db_version not found: %w (fresh database? run `jimu migrate up` first — adopt is only for existing databases migrated by the legacy global-table runner)",
			err)
	}
	v := maxVersion.Int64

	// 2. 逐能力登记 <= V 的版本
	result := make(map[string][]int64)
	for _, c := range caps {
		if c.Migrations == nil {
			continue
		}
		if _, err := fs.Stat(c.Migrations, "migrations/"+cfg.Dialect()); err != nil {
			continue
		}
		versions, err := capabilityMigrationVersions(c.Migrations, cfg.Dialect())
		if err != nil {
			return nil, fmt.Errorf("capability %s: %w", c.Name, err)
		}
		adopted := make([]int64, 0, len(versions))
		for _, ver := range versions {
			if ver <= v {
				adopted = append(adopted, ver)
			}
		}
		if len(adopted) == 0 {
			continue
		}
		if err := adoptOne(gooseDialect, sqlDB, "goose_db_version_"+c.Name, adopted); err != nil {
			return nil, fmt.Errorf("capability %s: %w", c.Name, err)
		}
		result[c.Name] = adopted
	}
	return result, nil
}

// adoptOne 用合成空迁移 FS 把 versions 登记进 table（is_applied=true），
// 不执行任何业务迁移 SQL：goose 对无语句的 SQL 迁移只插入版本行
// （provider_run.go runIndividually → runSQL 零语句 + maybeInsertOrDelete 插入）。
// 注意：不调用 p.Close()——它会关闭共享的 *sql.DB。
func adoptOne(dialect goose.Dialect, sqlDB *sql.DB, table string, versions []int64) error {
	p, err := goose.NewProvider(dialect, sqlDB, adoptEmptyFS{versions: versions}, goose.WithTableName(table))
	if err != nil {
		return fmt.Errorf("create provider: %w", err)
	}
	ctx := context.Background()
	for _, ver := range versions {
		if _, err := p.ApplyVersion(ctx, ver, true); err != nil {
			return fmt.Errorf("apply version %d: %w", ver, err)
		}
	}
	return nil
}

// adoptEmptyFS 为每个要登记的版本生成一个同名空迁移文件（仅 -- +goose Up/Down
// 注解、无语句）：goose 解析后 Up/Down 均为空语句集，ApplyVersion 只插入版本行
// 不执行任何 SQL。文件名中的版本号是占位（不与业务迁移冲突），ApplyVersion
// 登记的是调用方传入的版本参数。
type adoptEmptyFS struct {
	versions []int64
}

func (f adoptEmptyFS) Open(name string) (fs.File, error) {
	if name == "." {
		// fs.Glob 依赖根目录列表来发现迁移文件
		entries := make([]fs.DirEntry, 0, len(f.versions))
		for _, ver := range f.versions {
			entries = append(entries, adoptDirEntry{name: fmt.Sprintf("%06d_adopt_baseline.sql", ver)})
		}
		return &adoptDirFile{entries: entries}, nil
	}
	for _, ver := range f.versions {
		if name == fmt.Sprintf("%06d_adopt_baseline.sql", ver) {
			return &adoptMemFile{Reader: strings.NewReader("-- +goose Up\n-- +goose Down\n")}, nil
		}
	}
	return nil, fs.ErrNotExist
}

type adoptMemFile struct {
	*strings.Reader
}

func (f *adoptMemFile) Stat() (fs.FileInfo, error) { return adoptFileInfo{}, nil }
func (f *adoptMemFile) Close() error               { return nil }

type adoptFileInfo struct{}

func (adoptFileInfo) Name() string       { return "000000_adopt_baseline.sql" }
func (adoptFileInfo) Size() int64        { return int64(len("-- +goose Up\n-- +goose Down\n")) }
func (adoptFileInfo) Mode() fs.FileMode  { return 0o444 }
func (adoptFileInfo) ModTime() time.Time { return time.Time{} }
func (adoptFileInfo) IsDir() bool        { return false }
func (adoptFileInfo) Sys() any           { return nil }

type adoptDirFile struct {
	entries []fs.DirEntry
	pos     int
}

func (f *adoptDirFile) Stat() (fs.FileInfo, error) { return adoptFileInfo{}, nil }
func (f *adoptDirFile) Close() error               { return nil }
func (f *adoptDirFile) Read([]byte) (int, error)   { return 0, fmt.Errorf("cannot read directory") }
func (f *adoptDirFile) ReadDir(int) ([]fs.DirEntry, error) {
	if f.pos >= len(f.entries) {
		return nil, io.EOF
	}
	rest := f.entries[f.pos:]
	f.pos = len(f.entries)
	return rest, nil
}

type adoptDirEntry struct{ name string }

func (e adoptDirEntry) Name() string               { return e.name }
func (e adoptDirEntry) IsDir() bool                { return false }
func (e adoptDirEntry) Type() fs.FileMode          { return 0o444 }
func (e adoptDirEntry) Info() (fs.FileInfo, error) { return adoptFileInfo{}, nil }

// AutoMigrate 使用 Gorm 自动迁移（开发用）
func AutoMigrate(db *gorm.DB, models ...interface{}) error {
	return db.AutoMigrate(models...)
}
