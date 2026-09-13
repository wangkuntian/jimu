package db

import (
	"bytes"
	"context"
	"os"
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

func TestBuildBackup(t *testing.T) {
	var out bytes.Buffer

	spec, err := BuildBackup(mysqlCfg(), &out)
	require.NoError(t, err)
	assert.Equal(t, "mysqldump", spec.Tool)
	assert.Contains(t, spec.Args, "jimu", "应包含数据库名")
	assert.Contains(t, spec.Args, "--single-transaction")
	assert.Contains(t, spec.Env, "MYSQL_PWD=s3cret")
	assert.NotContains(t, strings.Join(spec.Args, " "), "s3cret", "口令不得出现在命令行参数中")

	spec, err = BuildBackup(pgCfg(), &out)
	require.NoError(t, err)
	assert.Equal(t, "pg_dump", spec.Tool)
	assert.Contains(t, spec.Args, "--format=plain")
	assert.Contains(t, spec.Env, "PGPASSWORD=s3cret")

	_, err = BuildBackup(config.DBConfig{Driver: "sqlite"}, &out)
	assert.ErrorContains(t, err, "unsupported driver")
}

func TestBuildRestore(t *testing.T) {
	in := strings.NewReader("SELECT 1;")

	spec, err := BuildRestore(mysqlCfg(), in)
	require.NoError(t, err)
	assert.Equal(t, "mysql", spec.Tool)
	assert.Same(t, in, spec.Stdin.((*strings.Reader)))
	assert.Contains(t, spec.Env, "MYSQL_PWD=s3cret")

	spec, err = BuildRestore(pgCfg(), in)
	require.NoError(t, err)
	assert.Equal(t, "psql", spec.Tool)
	assert.Contains(t, spec.Args, "--set=ON_ERROR_STOP=on", "恢复遇错必须立即失败")
	assert.Contains(t, spec.Args, "--dbname=jimu")

	_, err = BuildRestore(config.DBConfig{Driver: "sqlite"}, in)
	assert.ErrorContains(t, err, "unsupported driver")
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
	spec, err := BuildBackup(mysqlCfg(), os.Stdout)
	require.NoError(t, err)
	assert.NotContains(t, spec.RedactedCommand(), "s3cret")
	assert.Contains(t, spec.RedactedCommand(), "mysqldump")
}
