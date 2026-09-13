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

// DumpSpec 外部备份/恢复工具的调用描述。
// 口令通过环境变量传入（MYSQL_PWD / PGPASSWORD），不放进命令行参数，避免进程列表泄漏。
type DumpSpec struct {
	Tool   string
	Args   []string
	Env    []string
	Stdin  io.Reader // 恢复时提供
	Stdout io.Writer // 备份时提供
	Stderr io.Writer
}

// BuildBackup 构造备份命令：MySQL 用 mysqldump，PostgreSQL 用 pg_dump（明文 SQL）。
func BuildBackup(cfg config.DBConfig, out io.Writer) (*DumpSpec, error) {
	switch cfg.Driver {
	case "mysql":
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
		return &DumpSpec{
			Tool:   "mysqldump",
			Args:   args,
			Env:    []string{"MYSQL_PWD=" + cfg.Password},
			Stdout: out,
		}, nil
	case "postgres":
		args := []string{
			"--host=" + cfg.Host,
			"--port=" + strconv.Itoa(cfg.Port),
			"--username=" + cfg.User,
			"--no-password",
			"--format=plain",
			cfg.Database,
		}
		return &DumpSpec{
			Tool:   "pg_dump",
			Args:   args,
			Env:    []string{"PGPASSWORD=" + cfg.Password},
			Stdout: out,
		}, nil
	default:
		return nil, fmt.Errorf("backup: unsupported driver %q (supported: mysql, postgres)", cfg.Driver)
	}
}

// BuildRestore 构造恢复命令：MySQL 用 mysql 客户端，PostgreSQL 用 psql（遇错即停）。
func BuildRestore(cfg config.DBConfig, in io.Reader) (*DumpSpec, error) {
	switch cfg.Driver {
	case "mysql":
		args := []string{
			"--host=" + cfg.Host,
			"--port=" + strconv.Itoa(cfg.Port),
			"--user=" + cfg.User,
			"--default-character-set=utf8mb4",
			cfg.Database,
		}
		return &DumpSpec{
			Tool:  "mysql",
			Args:  args,
			Env:   []string{"MYSQL_PWD=" + cfg.Password},
			Stdin: in,
		}, nil
	case "postgres":
		args := []string{
			"--host=" + cfg.Host,
			"--port=" + strconv.Itoa(cfg.Port),
			"--username=" + cfg.User,
			"--no-password",
			"--set=ON_ERROR_STOP=on", // 遇错立即失败，避免半截恢复被当成成功
			"--dbname=" + cfg.Database,
		}
		return &DumpSpec{
			Tool:  "psql",
			Args:  args,
			Env:   []string{"PGPASSWORD=" + cfg.Password},
			Stdin: in,
		}, nil
	default:
		return nil, fmt.Errorf("restore: unsupported driver %q (supported: mysql, postgres)", cfg.Driver)
	}
}

// Run 执行外部工具；工具缺失或退出码非 0 时返回带上下文的错误
func (s *DumpSpec) Run(ctx context.Context) error {
	if s.Tool == "" {
		return fmt.Errorf("dump: tool is not set")
	}
	if _, err := exec.LookPath(s.Tool); err != nil {
		return fmt.Errorf("dump: %s not found in PATH, install the client tools (mysql-client / postgresql-client) or run inside the server container", s.Tool)
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
