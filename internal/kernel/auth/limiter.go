package auth

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type Limiter struct {
	client     redis.Scripter
	failClosed bool
}

func NewLimiter(client redis.Scripter, failClosed bool) *Limiter {
	return &Limiter{client: client, failClosed: failClosed}
}

func (l *Limiter) Allow(ctx context.Context, scope, key string, limit int, window time.Duration) (bool, error) {
	if limit <= 0 || window <= 0 {
		return true, nil
	}
	res, err := limiterScript.Run(ctx, l.client, []string{LimitKey(scope, key)}, int(window.Milliseconds()), limit).Int()
	if err != nil {
		return !l.failClosed, err
	}
	return res == 1, nil
}

var limiterScript = redis.NewScript(`
local current = redis.call("INCR", KEYS[1])
if current == 1 then
  redis.call("PEXPIRE", KEYS[1], ARGV[1])
end
if current > tonumber(ARGV[2]) then
  return 0
end
return 1
`)

// LimitKey 返回限流计数器在 Redis 中的键名（sha256(key) 避免明文泄露）。
// 导出供运维 peek 端点复用，保证查询与限流器写入同一 key。
func LimitKey(scope, key string) string {
	sum := sha256.Sum256([]byte(key))
	return fmt.Sprintf("jimu:auth:limit:%s:%s", scope, hex.EncodeToString(sum[:]))
}
