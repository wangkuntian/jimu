// Package authmemory 提供 auth 存储的进程内实现，用于单实例或无 Redis 的部署。
package authmemory

import (
	"time"

	"jimu/internal/capabilities/auth/authstore"
	"jimu/internal/platform/identity"
)

func init() {
	authstore.Register("memory", NewFromOptions)
}

// NewFromOptions 是 authstore 的 memory 驱动工厂。
func NewFromOptions(opts authstore.Options) (authstore.Stores, error) {
	return NewStores(opts.ResetCodeTTL), nil
}

// NewStores 构造进程内 auth 存储。重启会丢失会话与重置码；多实例之间不共享。
func NewStores(resetTTL time.Duration) authstore.Stores {
	return authstore.Stores{
		Sessions:    NewSessionStore(),
		RateCounter: identity.NewMemoryRateCounter(),
		Failures:    identity.NewMemoryFailureStore(),
		ResetCodes:  NewResetStore(resetTTL),
	}
}
