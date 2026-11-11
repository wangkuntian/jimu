package application

import (
	"context"
	"time"
)

// ResetCodeStore 是密码重置验证码的一次性存储。
type ResetCodeStore interface {
	Set(ctx context.Context, emailHash, code string) error
	GetAndDelete(ctx context.Context, emailHash string) (string, error)
	TTL() time.Duration
}
