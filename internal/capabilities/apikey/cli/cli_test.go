package cli

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"jimu/internal/capabilities/apikey/domain"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	// 每个用例独立的库名：shared cache 的 `file:...?mode=memory` 在同名时会跨用例共享数据。
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	gdb, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, gdb.AutoMigrate(&domain.APIKey{}))
	// 关闭底层连接：`t.Name()` 在 `-count=N` 的每轮里是同一个名字，不关闭会让同名内存库
	// 跨轮累积数据（表现为 -count=2 时的假红）。
	sqlDB, err := gdb.DB()
	require.NoError(t, err)
	t.Cleanup(func() { _ = sqlDB.Close() })
	return gdb
}

func TestIssueKeyReturnsPlaintextOnce(t *testing.T) {
	gdb := newTestDB(t)

	plain, err := issueKey(context.Background(), gdb, issueOptions{Name: "ci", Scopes: []string{"api:access"}, CreatedBy: 1})
	require.NoError(t, err)
	assert.NotEmpty(t, plain)

	// 明文只返回一次：库里存的是哈希与前缀，不是明文。
	var stored domain.APIKey
	require.NoError(t, gdb.First(&stored).Error)
	assert.Equal(t, "ci", stored.Name)
	assert.NotEqual(t, plain, stored.KeyHash)
	assert.Equal(t, domain.HashKey(plain), stored.KeyHash)
	assert.True(t, len(stored.KeyPrefix) > 0)
	assert.Contains(t, plain, stored.KeyPrefix)
}

func TestIssueKeyRequiresName(t *testing.T) {
	_, err := issueKey(context.Background(), newTestDB(t), issueOptions{})
	require.ErrorContains(t, err, "name is required")
}

func TestListKeysReturnsIssuedKey(t *testing.T) {
	gdb := newTestDB(t)
	_, err := issueKey(context.Background(), gdb, issueOptions{Name: "ci", Scopes: []string{"api:access"}})
	require.NoError(t, err)

	keys, total, err := listKeys(context.Background(), gdb, 0, 20)
	require.NoError(t, err)
	require.Equal(t, int64(1), total)
	require.Len(t, keys, 1)
	assert.Equal(t, "ci", keys[0].Name)
}

// Commands 必须提供 apikey 命令组，且组内含 issue 与 list（注册到 cmd/cli 的接缝）。
func TestCommandsExposeGroupWithIssueAndList(t *testing.T) {
	cmds := Commands()
	require.Len(t, cmds, 1)

	group := cmds[0]
	assert.Equal(t, "apikey", group.Use)
	uses := make([]string, 0, len(group.Commands()))
	for _, c := range group.Commands() {
		uses = append(uses, c.Use)
	}
	assert.Contains(t, uses, "issue")
	assert.Contains(t, uses, "list")
}

func TestSplitScopes(t *testing.T) {
	assert.Equal(t, []string{"api:access", "admin"}, splitScopes("api:access, admin"))
	assert.Equal(t, []string{"api:access"}, splitScopes("api:access"))
	assert.Empty(t, splitScopes(" , , "))
}
