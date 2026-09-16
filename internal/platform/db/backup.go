package db

import (
	"context"
	"fmt"
	"io"
	"os/exec"
	"strconv"
	"strings"

	"jimu/internal/config"
)

// MySQL 客户端候选命令：MariaDB 11+ 只提供 mariadb-dump/mariadb（官方镜像内没有
// mysqldump/mysql），Oracle MySQL 则相反，因此按顺序探测、取第一个可用的。
var (
	mysqlBackupTools  = []string{"mariadb-dump", "mysqldump"}
	mysqlRestoreTools = []string{"mariadb", "mysql"}
	pgBackupTools     = []string{"pg_dump"}
	pgRestoreTools    = []string{"psql"}
)

// DumpSpec 外部备份/恢复工具的调用描述。
// 口令通过环境变量传入（MYSQL_PWD / PGPASSWORD），不放进命令行参数，避免进程列表泄漏。
type DumpSpec struct {
	Tool   string // 已解析的命令（绝对路径，构建时即校验存在）
	Args   []string
	Env    []string
	Stdin  io.Reader // 恢复时由调用方设置
	Stdout io.Writer // 备份时由调用方设置
	Stderr io.Writer
}

// BuildBackup 构造备份命令并在构建阶段校验工具可用（MySQL 用 mariadb-dump/mysqldump，
// PostgreSQL 用 pg_dump）。工具缺失时立即报错，调用方无需先创建输出文件。
func BuildBackup(cfg config.DBConfig) (*DumpSpec, error) {
	switch cfg.Driver {
	case "mysql":
		tool, err := resolveTool(mysqlBackupTools, "mysql backup")
		if err != nil {
			return nil, err
		}
		args := []string{
			"--host=" + cfg.Host,
			"--port=" + strconv.Itoa(cfg.Port),
			"--user=" + cfg.User,
			"--single-transaction", // 不锁表（InnoDB 一致性快照）
			"--quick",
			"--routines",
			"--triggers",
			"--default-character-set=utf8mb4",
			cfg.Database,
		}
		return &DumpSpec{Tool: tool, Args: args, Env: []string{"MYSQL_PWD=" + cfg.Password}}, nil
	case "postgres":
		tool, err := resolveTool(pgBackupTools, "postgres backup")
		if err != nil {
			return nil, err
		}
		args := []string{
			"--host=" + cfg.Host,
			"--port=" + strconv.Itoa(cfg.Port),
			"--username=" + cfg.User,
			"--no-password",
			"--format=plain",
			cfg.Database,
		}
		return &DumpSpec{Tool: tool, Args: args, Env: []string{"PGPASSWORD=" + cfg.Password}}, nil
	default:
		return nil, fmt.Errorf("backup: unsupported driver %q (supported: mysql, postgres)", cfg.Driver)
	}
}

// BuildRestore 构造恢复命令并在构建阶段校验工具可用（MySQL 用 mariadb/mysql，
// PostgreSQL 用 psql，遇错即停）。
func BuildRestore(cfg config.DBConfig) (*DumpSpec, error) {
	switch cfg.Driver {
	case "mysql":
		tool, err := resolveTool(mysqlRestoreTools, "mysql restore")
		if err != nil {
			return nil, err
		}
		args := []string{
			"--host=" + cfg.Host,
			"--port=" + strconv.Itoa(cfg.Port),
			"--user=" + cfg.User,
			"--default-character-set=utf8mb4",
			cfg.Database,
		}
		return &DumpSpec{Tool: tool, Args: args, Env: []string{"MYSQL_PWD=" + cfg.Password}}, nil
	case "postgres":
		tool, err := resolveTool(pgRestoreTools, "postgres restore")
		if err != nil {
			return nil, err
		}
		args := []string{
			"--host=" + cfg.Host,
			"--port=" + strconv.Itoa(cfg.Port),
			"--username=" + cfg.User,
			"--no-password",
			"--set=ON_ERROR_STOP=on", // 遇错立即失败，避免半截恢复被当成成功
			"--dbname=" + cfg.Database,
		}
		return &DumpSpec{Tool: tool, Args: args, Env: []string{"PGPASSWORD=" + cfg.Password}}, nil
	default:
		return nil, fmt.Errorf("restore: unsupported driver %q (supported: mysql, postgres)", cfg.Driver)
	}
}

// resolveTool 返回 PATH 中第一个可用的候选命令（绝对路径）。
// 全部缺失时给出包含候选项与安装建议的错误，便于容器/主机环境排障。
func resolveTool(candidates []string, purpose string) (string, error) {
	for _, name := range candidates {
		if path, err := exec.LookPath(name); err == nil {
			return path, nil
		}
	}
	return "", fmt.Errorf("dump: no client tool found for %s, tried %s; install mariadb-client, mysql-client or postgresql-client",
		purpose, strings.Join(candidates, ", "))
}

// AvailableTool 返回候选命令中当前环境实际可用的那个（都不可用返回空串），供排障提示
func AvailableTool(candidates []string) string {
	for _, name := range candidates {
		if path, err := exec.LookPath(name); err == nil {
			return path
		}
	}
	return ""
}

// MySQLBackupTools 返回 MySQL 备份候选命令（副本，供 CLI 提示缺失时列出候选项）
func MySQLBackupTools() []string { return append([]string{}, mysqlBackupTools...) }

// Run 执行外部工具；工具缺失或退出码非 0 时返回带上下文的错误
func (s *DumpSpec) Run(ctx context.Context) error {
	if s.Tool == "" {
		return fmt.Errorf("dump: tool is not set")
	}
	if _, err := exec.LookPath(s.Tool); err != nil {
		return fmt.Errorf("dump: %s not found in PATH, install the client tools (mariadb-client / mysql-client / postgresql-client) or run inside the server container", s.Tool)
	}

	cmd := exec.CommandContext(ctx, s.Tool, s.Args...) //nolint:gosec // 参数由配置构造，无用户输入拼接
	cmd.Env = append(cmd.Environ(), s.Env...)
	cmd.Stdin = s.Stdin
	cmd.Stdout = s.Stdout
	cmd.Stderr = s.Stderr

	if err := cmd.Run(); err != nil {
		if ctx.Err() != nil {
			return fmt.Errorf("dump: %s timed out: %w", s.Tool, ctx.Err())
		}
		return fmt.Errorf("dump: %s failed: %w", s.Tool, err)
	}
	return nil
}

// RedactedCommand 返回可安全打印的命令行（口令只在环境变量里，本就不出现在参数中）
func (s *DumpSpec) RedactedCommand() string {
	return strings.Join(append([]string{s.Tool}, s.Args...), " ")
}
