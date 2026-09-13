package middleware

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"jimu/internal/platform/tenant"
	"jimu/internal/shared/errors"
	"jimu/internal/shared/response"

	redistore "jimu/internal/platform/redis"

	"github.com/gin-gonic/gin"
)

const (
	// IdempotencyHeader 幂等键请求头：同一键的重复请求返回首次结果，不重复执行业务逻辑
	IdempotencyHeader = "Idempotency-Key"
	// IdempotencyReplayedHeader 标记响应来自幂等缓存（便于客户端与排障区分）
	IdempotencyReplayedHeader = "Idempotency-Replayed"
	// idempotencyKeyPrefix Redis key 前缀
	idempotencyKeyPrefix = "idempotency:"
	// idempotencyMaxKeyLen 幂等键长度上限
	idempotencyMaxKeyLen = 128
	// idempotencyMinKeyLen 幂等键长度下限（避免过短的键互相碰撞）
	idempotencyMinKeyLen = 8
	// idempotencyMaxBody 可缓存响应体上限；超过则不缓存（重试会重新执行）
	idempotencyMaxBody = 256 << 10
)

// idempotencyState 幂等记录状态
type idempotencyState string

const (
	idempotencyInProgress idempotencyState = "in_progress"
	idempotencyDone       idempotencyState = "done"
)

// IdempotencyMiddleware 幂等性中间件。
// 客户端在请求头中携带 Idempotency-Key: <uuid> 时，同键重复请求返回首次结果：
//   - 键按「租户 + 用户 + 方法 + 路径 + 客户端键」哈希存储，避免跨用户/跨接口互相命中；
//   - 首个请求用 SET NX 占位，并发的同键请求返回 409/1009（而不是各执行一次）；
//   - 完成的响应（2xx）连同状态码与 Content-Type 一起缓存，重放时带 Idempotency-Replayed: true；
//   - 5xx 与超过上限的响应体不缓存，释放占位以便客户端用同键重试；
//   - Redis 不可用时直接放行（fail-open），幂等是增强能力，不应阻断业务。
func IdempotencyMiddleware(redis redistore.Client, ttl time.Duration) gin.HandlerFunc {
	if ttl <= 0 {
		ttl = 24 * time.Hour
	}
	return func(c *gin.Context) {
		key := c.GetHeader(IdempotencyHeader)
		if key == "" {
			c.Next()
			return
		}

		// 验证 key 格式（长度区间，避免过短碰撞与超长键撑爆 Redis）
		if len(key) < idempotencyMinKeyLen || len(key) > idempotencyMaxKeyLen {
			response.Fail(c, errors.New(errors.CodeInvalidParam, "invalid idempotency key format"))
			c.Abort()
			return
		}
		if redis == nil {
			c.Next()
			return
		}

		ctx := c.Request.Context()
		cacheKey := idempotencyCacheKey(c, key)

		pending, err := json.Marshal(cachedResponse{State: idempotencyInProgress})
		if err != nil {
			c.Next()
			return
		}
		acquired, err := redis.SetNX(ctx, cacheKey, pending, ttl).Result()
		if err != nil {
			// Redis 异常：放行，不阻断业务
			c.Next()
			return
		}
		if !acquired {
			replayCachedResponse(c, redis, cacheKey)
			return
		}

		w := newResponseBodyWriter(c.Writer, idempotencyMaxBody)
		c.Writer = w

		c.Next()
		status := c.Writer.Status()

		// 只缓存成功响应；失败或响应过大时释放占位，允许同键重试
		if status < 200 || status >= 300 || w.isTruncated() {
			_ = redis.Del(ctx, cacheKey).Err()
			return
		}
		cached := cachedResponse{
			State:   idempotencyDone,
			Status:  status,
			Headers: map[string]string{"Content-Type": w.Header().Get("Content-Type")},
			Body:    w.capturedBody(),
		}
		if data, err := json.Marshal(cached); err == nil {
			_ = redis.Set(ctx, cacheKey, data, ttl).Err()
		}
	}
}

// idempotencyCacheKey 生成幂等缓存键：绑定租户、用户、方法与路径，避免跨上下文互相命中
func idempotencyCacheKey(c *gin.Context, key string) string {
	scope := strings.Join([]string{
		strconv.FormatUint(tenant.FromContext(c.Request.Context()), 10),
		strconv.FormatUint(ginUserID(c), 10),
		c.Request.Method,
		c.Request.URL.Path,
		key,
	}, "|")
	sum := sha256.Sum256([]byte(scope))
	return idempotencyKeyPrefix + hex.EncodeToString(sum[:])
}

// ginUserID 读取认证中间件写入的用户 ID（未认证时为 0）
func ginUserID(c *gin.Context) uint64 {
	if v, ok := c.Get("user_id"); ok {
		if id, ok := v.(uint64); ok {
			return id
		}
	}
	return 0
}

// replayCachedResponse 重放已完成的响应；仍在处理中时返回冲突，避免重复执行
func replayCachedResponse(c *gin.Context, redis redistore.Client, cacheKey string) {
	ctx := c.Request.Context()
	raw, err := redis.Get(ctx, cacheKey).Result()
	if err != nil {
		// 记录刚好过期/被清理：放行重新执行，不做额外保护
		c.Next()
		return
	}
	var cached cachedResponse
	if err := json.Unmarshal([]byte(raw), &cached); err != nil || cached.State != idempotencyDone {
		response.Fail(c, errors.New(errors.CodeConflict, "a request with this idempotency key is still in progress"))
		c.Abort()
		return
	}

	c.Header(IdempotencyReplayedHeader, "true")
	for k, v := range cached.Headers {
		c.Header(k, v)
	}
	c.Status(cached.Status)
	_, _ = c.Writer.Write([]byte(cached.Body))
	c.Abort()
}

// cachedResponse 缓存的响应
type cachedResponse struct {
	State   idempotencyState  `json:"state"`
	Status  int               `json:"status"`
	Headers map[string]string `json:"headers"`
	Body    string            `json:"body"`
}
