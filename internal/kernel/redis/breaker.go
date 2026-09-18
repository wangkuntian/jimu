package redis

import (
	"context"
	"errors"
	"io"
	"net"
	"strings"

	"jimu/internal/kernel/breaker"

	"github.com/redis/go-redis/v9"
)

// breakerHook 在连接与命令级别接入熔断：Redis 不可用时快速失败，
// 避免每个请求都等待读/写超时（session/限流/缓存几乎每请求都会用到 Redis）。
type breakerHook struct {
	b *breaker.Breaker
}

// AttachBreaker 给客户端挂载熔断 hook，返回熔断器（供测试/指标观察）
func AttachBreaker(client Client, cfg breaker.Config) *breaker.Breaker {
	b := breaker.New("redis", cfg)
	client.AddHook(&breakerHook{b: b})
	return b
}

func (h *breakerHook) DialHook(next redis.DialHook) redis.DialHook {
	return func(ctx context.Context, network, addr string) (net.Conn, error) {
		// 只做拦截不计数：拨号失败会由 ProcessHook 以命令错误形式统一上报，避免重复计数
		if err := h.b.Allow(); err != nil {
			return nil, err
		}
		return next(ctx, network, addr)
	}
}

func (h *breakerHook) ProcessHook(next redis.ProcessHook) redis.ProcessHook {
	return func(ctx context.Context, cmd redis.Cmder) error {
		if err := h.b.Allow(); err != nil {
			return err
		}
		err := next(ctx, cmd)
		h.record(err)
		return err
	}
}

func (h *breakerHook) ProcessPipelineHook(next redis.ProcessPipelineHook) redis.ProcessPipelineHook {
	return func(ctx context.Context, cmds []redis.Cmder) error {
		if err := h.b.Allow(); err != nil {
			return err
		}
		err := next(ctx, cmds)
		h.record(err)
		return err
	}
}

// record 只把传输类错误计为失败；redis.Nil（未命中）与业务错误视为成功，避免误熔断
func (h *breakerHook) record(err error) {
	if err == nil || errors.Is(err, redis.Nil) {
		h.b.OnSuccess()
		return
	}
	if IsTransportError(err) {
		h.b.OnFailure()
		return
	}
	h.b.OnSuccess()
}

// IsTransportError 判断是否为连接/网络类错误（可触发熔断）
func IsTransportError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, redis.ErrClosed) ||
		errors.Is(err, io.EOF) ||
		errors.Is(err, io.ErrUnexpectedEOF) ||
		errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	var netErr net.Error
	if errors.As(err, &netErr) {
		return true
	}
	msg := err.Error()
	for _, s := range []string{
		"connection refused", "connection reset", "broken pipe",
		"i/o timeout", "no such host", "server closed the connection", "unexpected EOF",
	} {
		if strings.Contains(msg, s) {
			return true
		}
	}
	return false
}
