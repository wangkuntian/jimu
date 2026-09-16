package db

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"jimu/internal/config"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func mysqlCfg() config.DBConfig {
	return config.DBConfig{Driver: "mysql", Host: "127.0.0.1", Port: 3306, User: "jimu", Password: "s3cret", Database: "jimu"}
}

func pgCfg() config.DBConfig {
	return config.DBConfig{Driver: "postgres", Host: "db", Port: 5432, User: "jimu", Password: "s3cret", Database: "jimu"}
}

// fakeToolPath 构造只含指定可执行文件的 PATH，用于稳定验证命令探测顺序
func fakeToolPath(t *testing.T, tools ...string) string {
	t.Helper()
	dir := t.TempDir()
	for _, tool := range tools {
		path := filepath.Join(dir, tool)
		require.NoError(t, os.WriteFile(path, []byte("#!/bin/sh\nexit 0\n"), 0o755))
	}
	t.Setenv("PATH", dir)
	return dir
}

func TestBuildBackupPrefersMariaDBTools(t *testing.T) {
	// 生产镜像（mariadb:12.x）只有 mariadb-dump，没有 mysqldump
	dir := fakeToolPath(t, "mariadb-dump")

	spec, err := BuildBackup(mysqlCfg())
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(dir, "mariadb-dump"), spec.Tool, "应优先使用 mariadb-dump")
	assert.Contains(t, spec.Args, "jimu", "应包含数据库名")
	assert.Contains(t, spec.Args, "--single-transaction", "InnoDB 备份不锁表")
	assert.Contains(t, spec.Args, "--routines")
	assert.Contains(t, spec.Args, "--triggers")
	assert.Contains(t, spec.Env, "MYSQL_PWD=s3cret")
	assert.NotContains(t, strings.Join(spec.Args, " "), "s3cret", "口令不得出现在命令行参数中")
}

func TestBuildBackupPrefersMysqldumpWhenOnlyThatExists(t *testing.T) {
	dir := fakeToolPath(t, "mysqldump")

	spec, err := BuildBackup(mysqlCfg())
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(dir, "mysqldump"), spec.Tool)
}

func TestBuildBackupPostgres(t *testing.T) {
	dir := fakeToolPath(t, "pg_dump")

	spec, err := BuildBackup(pgCfg())
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(dir, "pg_dump"), spec.Tool)
	assert.Contains(t, spec.Args, "--format=plain")
	assert.Contains(t, spec.Env, "PGPASSWORD=s3cret")
}

func TestBuildRestorePrefersMariaDBTools(t *testing.T) {
	dir := fakeToolPath(t, "mariadb", "mysql")

	spec, err := BuildRestore(mysqlCfg())
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(dir, "mariadb"), spec.Tool, "应优先使用 mariadb 客户端")
	assert.Contains(t, spec.Env, "MYSQL_PWD=s3cret")

	dir = fakeToolPath(t, "mysql")
	spec, err = BuildRestore(mysqlCfg())
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(dir, "mysql"), spec.Tool)
}

func TestBuildRestorePostgres(t *testing.T) {
	fakeToolPath(t, "psql")

	spec, err := BuildRestore(pgCfg())
	require.NoError(t, err)
	assert.Contains(t, spec.Args, "--set=ON_ERROR_STOP=on", "恢复遇错必须立即失败")
	assert.Contains(t, spec.Args, "--dbname=jimu")
}

func TestBuildFailsFastWhenToolMissing(t *testing.T) {
	fakeToolPath(t) // 空 PATH

	_, err := BuildBackup(mysqlCfg())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "mariadb-dump, mysqldump", "错误应列出候选命令")
	assert.Contains(t, err.Error(), "install")

	_, err = BuildRestore(mysqlCfg())
	assert.ErrorContains(t, err, "mariadb, mysql")

	_, err = BuildBackup(config.DBConfig{Driver: "sqlite"})
	assert.ErrorContains(t, err, "unsupported driver")

	_, err = BuildRestore(config.DBConfig{Driver: "sqlite"})
	assert.ErrorContains(t, err, "unsupported driver")
}

func TestAvailableTool(t *testing.T) {
	dir := fakeToolPath(t, "mariadb-dump")
	assert.Equal(t, filepath.Join(dir, "mariadb-dump"), AvailableTool(MySQLBackupTools()))

	fakeToolPath(t)
	assert.Empty(t, AvailableTool(MySQLBackupTools()))
}

func TestDumpSpecRunUsesStdinStdout(t *testing.T) {
	// 用 cat 冒充 dump 工具，验证 stdin → stdout 管道与参数传递
	var out bytes.Buffer
	spec := &DumpSpec{Tool: "cat", Stdin: strings.NewReader("hello dump"), Stdout: &out}
	require.NoError(t, spec.Run(context.Background()))
	assert.Equal(t, "hello dump", out.String())
}

func TestDumpSpecRunReportsFailure(t *testing.T) {
	spec := &DumpSpec{Tool: "sh", Args: []string{"-c", "exit 3"}}
	err := spec.Run(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed")

	missing := &DumpSpec{Tool: "definitely-not-a-real-tool"}
	err = missing.Run(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found in PATH")

	empty := &DumpSpec{}
	assert.ErrorContains(t, empty.Run(context.Background()), "tool is not set")
}

func TestDumpSpecRunHonoursTimeout(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	spec := &DumpSpec{Tool: "sleep", Args: []string{"5"}}
	err := spec.Run(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "timed out")
}

func TestRedactedCommandHidesNothingSensitive(t *testing.T) {
	fakeToolPath(t, "mariadb-dump")
	spec, err := BuildBackup(mysqlCfg())
	require.NoError(t, err)
	assert.NotContains(t, spec.RedactedCommand(), "s3cret")
	assert.Contains(t, spec.RedactedCommand(), "mariadb-dump")
}
