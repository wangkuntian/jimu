// Package authredis 提供 auth 存储的 Redis 实现。
package authredis

import (
	"errors"
	"time"

	"jimu/internal/capabilities/auth/authstore"
	identityredis "jimu/internal/platform/identity/redisstore"
	redistore "jimu/internal/platform/redisclient"
)

func init() {
	authstore.Register("redis", NewFromOptions)
}

// NewFromOptions 是 authstore 的 Redis 驱动工厂。
func NewFromOptions(opts authstore.Options) (authstore.Stores, error) {
	client, ok := opts.RedisClient.(redistore.Client)
	if !ok || client == nil {
		return authstore.Stores{}, errors.New("redis client is not available")
	}
	return NewStores(client, opts.ResetCodeTTL), nil
}

// NewStores 用给定 Redis 客户端构造完整 auth 存储。
func NewStores(client redistore.Client, resetTTL time.Duration) authstore.Stores {
	return authstore.Stores{
		Sessions:    identityredis.NewSessionStore(client),
		RateCounter: identityredis.NewRateCounter(client),
		Failures:    identityredis.NewFailureStore(client),
		ResetCodes:  NewResetStore(client, resetTTL),
	}
}
