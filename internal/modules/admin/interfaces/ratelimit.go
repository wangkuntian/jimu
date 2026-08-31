package interfaces

import (
	"strings"
	"time"

	"jimu/internal/platform/auth"
	"jimu/internal/shared/errors"
	"jimu/internal/shared/response"

	redistore "jimu/internal/platform/redis"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// AdminRateLimitHandler 限流状态可视化 handler（仅读，不消费令牌）
type AdminRateLimitHandler struct {
	rdb redistore.Client
}

// NewAdminRateLimitHandler 创建限流状态 handler
func NewAdminRateLimitHandler(rdb redistore.Client) *AdminRateLimitHandler {
	return &AdminRateLimitHandler{rdb: rdb}
}

// AuthPeek 查看某 scope+key 的当前限流计数与剩余 TTL
//
// @Summary 查看认证限流状态
// @Description 不消费令牌地返回某 scope（如 login）+ key（如 ip:1.2.3.4）当前的计数与剩余窗口
// @Tags admin/ratelimit
// @Param scope query string true "限流作用域" Enums(login, register)
// @Param key query string true "限流键（如 ip:1.2.3.4 或 username:alice）"
// @Success 200 {object} response.Body
// @Failure 400 {object} response.Body
// @Failure 500 {object} response.Body
// @Router /api/v1/admin/ratelimit/auth [get]
func (h *AdminRateLimitHandler) AuthPeek(c *gin.Context) {
	scope := strings.TrimSpace(c.Query("scope"))
	key := strings.TrimSpace(c.Query("key"))
	if scope == "" || key == "" {
		response.Fail(c, errors.New(errors.CodeInvalidParam, "scope and key are required"))
		return
	}

	ctx := c.Request.Context()
	redisKey := auth.LimitKey(scope, key)

	count, err := h.rdb.Get(ctx, redisKey).Int64()
	if err != nil && err != redis.Nil {
		response.Fail(c, errors.Wrap(errors.CodeInternalError, "peek rate limit failed", err))
		return
	}
	// key 不存在时 count=0
	if err == redis.Nil {
		count = 0
	}

	ttl, err := h.rdb.TTL(ctx, redisKey).Result()
	if err != nil {
		response.Fail(c, errors.Wrap(errors.CodeInternalError, "peek rate limit ttl failed", err))
		return
	}
	// miniredis/某些场景返回 -1ns，统一成 0
	if ttl < 0 {
		ttl = 0
	}

	response.OK(c, gin.H{
		"scope":     scope,
		"key":       key,
		"count":     count,
		"ttl_ms":    ttl.Milliseconds(),
		"reset_at":  time.Now().Add(ttl).Unix(),
		"redis_key": redisKey,
	})
}
