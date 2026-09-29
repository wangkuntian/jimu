// Package cli 由 apikey 能力自带其 CLI 子命令（P2.6 的命令归属接缝：
// 能力在自己的包内提供命令，cmd/cli 显式 import 并注册；internal/contract 不引入 cobra）。
//
// 存在意义之一是补上 machine 形态的缺口：该形态刻意没有 auth/JWT 链，
// /api/v1/admin/apikeys 无法自助访问，首把 API Key 需要带外签发。
package cli

import (
	"context"
	"fmt"
	"strings"
	"time"

	"jimu/internal/capabilities/apikey"
	"jimu/internal/capabilities/apikey/application"
	"jimu/internal/capabilities/apikey/domain"
	"jimu/internal/capabilities/apikey/infrastructure"
	"jimu/internal/config"
	"jimu/internal/kernel/db"
	"jimu/internal/kernel/logger"

	"github.com/spf13/cobra"
	"gorm.io/gorm"
)

// issueOptions 是 issue 子命令的输入（与 application.CreateKeyInput 对齐）。
type issueOptions struct {
	Name      string
	Scopes    []string
	ExpiresIn int // 天，0 = 不过期
	CreatedBy uint64
}

// issueKey 签发一把 API Key 并返回明文（仅此一次）。
// 抽成独立函数以便用 sqlite 单测覆盖，不依赖 CLI 进程环境。
func issueKey(ctx context.Context, gdb *gorm.DB, in issueOptions) (string, error) {
	svc := application.NewAdminAPIKeyService(infrastructure.NewMysqlAPIKeyRepository(gdb))
	plain, _, err := svc.CreateKey(ctx, application.CreateKeyInput{
		Name:      in.Name,
		Scopes:    in.Scopes,
		ExpiresIn: in.ExpiresIn,
		CreatedBy: in.CreatedBy,
	})
	if err != nil {
		return "", err
	}
	return plain, nil
}

// listKeys 列出 API Key（不返回明文，也不返回哈希）。
func listKeys(ctx context.Context, gdb *gorm.DB, offset, limit int) ([]domain.APIKey, int64, error) {
	svc := application.NewAdminAPIKeyService(infrastructure.NewMysqlAPIKeyRepository(gdb))
	return svc.ListKeys(ctx, offset, limit)
}

// Commands 返回本能力贡献的命令（一个 apikey 命令组，由 cmd/cli 注册到根命令）。
func Commands() []*cobra.Command {
	group := &cobra.Command{
		Use:   "apikey",
		Short: "API key management commands",
	}
	group.AddCommand(newIssueCmd(), newListCmd())
	return []*cobra.Command{group}
}

func newIssueCmd() *cobra.Command {
	var (
		name      string
		scopes    string
		expiresIn int
		createdBy uint64
	)
	cmd := &cobra.Command{
		Use:   "issue",
		Short: "Issue a new API key (plaintext is shown only once)",
		RunE: func(cmd *cobra.Command, _ []string) error {
			// 先校验必填项再连库：缺 --name 时给出明确提示，而不是先报连接错误。
			if name == "" {
				return fmt.Errorf("--name is required")
			}
			gdb, err := connectDB()
			if err != nil {
				return err
			}
			plain, err := issueKey(cmd.Context(), gdb, issueOptions{
				Name:      name,
				Scopes:    splitScopes(scopes),
				ExpiresIn: expiresIn,
				CreatedBy: createdBy,
			})
			if err != nil {
				return err
			}
			fmt.Printf("API key: %s\n", plain)
			fmt.Println("请立即保存：明文只显示这一次。")
			return nil
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "key name (required)")
	cmd.Flags().StringVar(&scopes, "scopes", apikey.ScopeProtected, "comma separated scopes")
	cmd.Flags().IntVar(&expiresIn, "expires-days", 0, "expire after N days (0 = never)")
	cmd.Flags().Uint64Var(&createdBy, "created-by", 0, "creator user id (0 = unset)")
	return cmd
}

func newListCmd() *cobra.Command {
	var limit int
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List API keys",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if limit < 1 {
				return fmt.Errorf("--limit must be >= 1")
			}
			gdb, err := connectDB()
			if err != nil {
				return err
			}
			keys, total, err := listKeys(cmd.Context(), gdb, 0, limit)
			if err != nil {
				return err
			}
			fmt.Printf("%-6s %-8s %-20s %-16s %-28s %s\n", "ID", "TENANT", "NAME", "PREFIX", "SCOPES", "CREATED_AT")
			for _, k := range keys {
				fmt.Printf("%-6d %-8d %-20s %-16s %-28s %s\n",
					k.ID, k.TenantID, k.Name, k.KeyPrefix, k.Scopes, k.CreatedAt.Format(time.RFC3339))
			}
			fmt.Printf("共 %d 把（本次列出 %d 把）\n", total, len(keys))
			return nil
		},
	}
	cmd.Flags().IntVar(&limit, "limit", 20, "max keys to list (>= 1)")
	return cmd
}

// connectDB 与 cmd/cli 的其它命令同一写法：加载配置 → 日志 → 雪花 ID → 连接（带重试）。
func connectDB() (*gorm.DB, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}
	log := logger.New(cfg.Log)
	if err := db.InitSnowflake(cfg.ID.WorkerID); err != nil {
		return nil, fmt.Errorf("failed to init snowflake: %w", err)
	}
	gdb, err := db.ConnectWithRetry(cfg.DB, log)
	if err != nil {
		return nil, fmt.Errorf("failed to connect database: %w", err)
	}
	return gdb, nil
}

// splitScopes 解析逗号分隔的 scope 列表，忽略空白项。
func splitScopes(raw string) []string {
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}
