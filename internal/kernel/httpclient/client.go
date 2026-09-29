// internal/kernel/httpclient/client.go
package httpclient

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"jimu/internal/kernel/breaker"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/time/rate"
)

const (
	defaultTimeout       = 10 * time.Second
	defaultMaxRetries    = 2
	defaultRetryInterval = 200 * time.Millisecond
	defaultMaxFailures   = 5
	defaultResetTimeout  = 30 * time.Second
)

// ErrCircuitOpen 熔断开启时请求被快速拒绝（与 kernel/breaker.ErrOpen 同一错误值）
var ErrCircuitOpen = breaker.ErrOpen

// Config 出站 HTTP client 配置
type Config struct {
	TimeoutSec      int `mapstructure:"timeout_sec"`       // 单次请求超时（秒），0 用默认 10
	MaxRetries      int `mapstructure:"max_retries"`       // 失败重试次数，0 用默认 2
	RetryIntervalMS int `mapstructure:"retry_interval_ms"` // 重试基础间隔（毫秒），0 用默认 200
	MaxFailures     int `mapstructure:"max_failures"`      // 连续失败次数达此值熔断，0 用默认 5
	ResetTimeoutMS  int `mapstructure:"reset_timeout_ms"`  // 熔断后冷却时长（毫秒），0 用默认 30_000
	RateLimitRate   int `mapstructure:"rate_limit_rate"`   // 每秒请求数（按目标 host 独立限流），0 不限流
	RateLimitBurst  int `mapstructure:"rate_limit_burst"`  // 令牌桶容量，0 用 rate（桶=平均速率）
}

// Client 统一出站 HTTP client：超时 + 指数退避重试（仅网络错误与 5xx）+ trace 注入 + 熔断 + per-host 限流
type Client struct {
	hc            *http.Client
	maxRetries    int
	retryInterval time.Duration
	tracer        trace.Tracer
	circuit       *breaker.Breaker
	limiter       *hostRateLimiter
}

// New 创建出站 HTTP client
func New(cfg Config) *Client {
	timeout := time.Duration(cfg.TimeoutSec) * time.Second
	if timeout <= 0 {
		timeout = defaultTimeout
	}
	maxRetries := cfg.MaxRetries
	if maxRetries < 0 {
		maxRetries = 0
	} else if maxRetries == 0 {
		maxRetries = defaultMaxRetries
	}
	retryInterval := time.Duration(cfg.RetryIntervalMS) * time.Millisecond
	if retryInterval <= 0 {
		retryInterval = defaultRetryInterval
	}
	maxFailures := cfg.MaxFailures
	if maxFailures <= 0 {
		maxFailures = defaultMaxFailures
	}
	resetTimeout := time.Duration(cfg.ResetTimeoutMS) * time.Millisecond
	if resetTimeout <= 0 {
		resetTimeout = defaultResetTimeout
	}
	burst := cfg.RateLimitBurst
	if burst <= 0 {
		burst = cfg.RateLimitRate
	}
	var limiter *hostRateLimiter
	if cfg.RateLimitRate > 0 {
		limiter = newHostRateLimiter(float64(cfg.RateLimitRate), burst)
	}
	return &Client{
		hc:            &http.Client{Timeout: timeout},
		maxRetries:    maxRetries,
		retryInterval: retryInterval,
		tracer:        otel.Tracer("jimu.httpclient"),
		circuit:       breaker.New("httpclient", breaker.Config{MaxFailures: maxFailures, ResetTimeout: resetTimeout}),
		limiter:       limiter,
	}
}

// Do 执行请求：注入 traceparent，对网络错误与 5xx 指数退避重试，熔断与限流保护。
// 4xx 与 2xx 不重试；请求体无法重放（GetBody 为 nil）时放弃重试。
// 熔断开启时直接返回 ErrCircuitOpen，不发起请求；限流等待以 ctx 控制超时。
func (c *Client) Do(ctx context.Context, req *http.Request) (*http.Response, error) {
	if c.circuit.Allow() != nil {
		return nil, ErrCircuitOpen
	}
	if c.limiter != nil {
		if err := c.limiter.wait(ctx, req.URL.Host); err != nil {
			return nil, err
		}
	}
	resp, err := c.do(ctx, req)
	if err != nil {
		c.circuit.OnFailure()
		return nil, err
	}
	c.circuit.OnSuccess()
	return resp, nil
}

func (c *Client) do(ctx context.Context, req *http.Request) (*http.Response, error) {
	ctx, span := c.tracer.Start(ctx, req.Method+" "+req.URL.Path)
	defer span.End()
	otel.GetTextMapPropagator().Inject(ctx, propagation.HeaderCarrier(req.Header))

	var lastErr error
	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		if attempt > 0 {
			// 仅当请求体非空且不可重放时放弃重试（空 body 的请求无需重建即可重放）
			if req.GetBody == nil && req.Body != nil {
				break
			}
			retryReq := req.Clone(ctx)
			if req.GetBody != nil {
				body, err := req.GetBody()
				if err != nil {
					return nil, fmt.Errorf("rebuild request body: %w", err)
				}
				retryReq.Body = body
			}
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(c.backoff(attempt)):
			}
			req = retryReq
		}

		resp, err := c.hc.Do(req)
		if err != nil {
			lastErr = err
			continue
		}
		if resp.StatusCode >= 500 && resp.StatusCode < 600 {
			resp.Body.Close()
			lastErr = fmt.Errorf("upstream returned %d", resp.StatusCode)
			continue
		}
		return resp, nil
	}
	span.RecordError(lastErr)
	return nil, fmt.Errorf("request failed after %d attempt(s): %w", c.maxRetries+1, lastErr)
}

// backoff 指数退避：interval, interval*2, interval*4 ...
func (c *Client) backoff(attempt int) time.Duration {
	return c.retryInterval * time.Duration(1<<uint(attempt-1))
}

// circuitState 熔断状态：closed 正常 / open 熔断拒绝 / halfOpen 冷却后单次探测
// hostRateLimiter 按目标 host 独立限流的令牌桶集合。
// 每个 host 懒创建独立 limiter，避免打爆单一第三方时拖累其他调用。
type hostRateLimiter struct {
	mu       sync.Mutex
	rate     rate.Limit
	burst    int
	limiters map[string]*rate.Limiter
}

func newHostRateLimiter(ratePerSec float64, burst int) *hostRateLimiter {
	return &hostRateLimiter{
		rate:     rate.Limit(ratePerSec),
		burst:    burst,
		limiters: make(map[string]*rate.Limiter),
	}
}

// wait 阻塞直到令牌可用或 ctx 结束。
func (h *hostRateLimiter) wait(ctx context.Context, host string) error {
	return h.limiter(host).Wait(ctx)
}

func (h *hostRateLimiter) limiter(host string) *rate.Limiter {
	h.mu.Lock()
	defer h.mu.Unlock()
	l, ok := h.limiters[host]
	if !ok {
		l = rate.NewLimiter(h.rate, h.burst)
		h.limiters[host] = l
	}
	return l
}
