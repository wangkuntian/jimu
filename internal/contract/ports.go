// Package contract 端口定义：能力间只经此处接口调用（见 AGENTS.md 能力边界）。
package contract

import (
	"context"
	"errors"
	"time"
)

// ErrNotFound 端口查询目标不存在（由端口实现返回，消费方据此映射语义）。
var ErrNotFound = errors.New("record not found")

// Userinfo 用户信息端口视图（端口自有结构，不含 gorm 标签）。
type Userinfo struct {
	ID        uint64
	Username  string
	Status    int8
	CreatedAt time.Time
}

// UserinfoSource UserInfoService 所需的用户只读数据端口，
// 由 user 能力提供实现，替代消费方对 user/domain 的直接引用。
type UserinfoSource interface {
	// GetByID 按 ID 查用户；不存在时返回 ErrNotFound。
	GetByID(ctx context.Context, id uint64) (*Userinfo, error)
	// List 分页查用户（page 从 1 起；按 id 降序，平台级视角不过滤租户）。
	List(ctx context.Context, page, pageSize int) ([]Userinfo, int64, error)
}
