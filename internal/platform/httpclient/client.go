// internal/platform/httpclient/client.go
package httpclient

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

const (
	defaultTimeout       = 10 * time.Second
	defaultMaxRetries    = 2
	defaultRetryInterval = 200 * time.Millisecond
)

// Config 出站 HTTP client 配置
type Config struct {
	TimeoutSec      int `mapstructure:"timeout_sec"`       // 单次请求超时（秒），0 用默认 10
	MaxRetries      int `mapstructure:"max_retries"`       // 失败重试次数，0 用默认 2
	RetryIntervalMS int `mapstructure:"retry_interval_ms"` // 重试基础间隔（毫秒），0 用默认 200
}

// Client 统一出站 HTTP client：超时 + 指数退避重试（仅网络错误与 5xx）+ trace 注入
type Client struct {
	hc            *http.Client
	maxRetries    int
	retryInterval time.Duration
	tracer        trace.Tracer
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
	return &Client{
		hc:            &http.Client{Timeout: timeout},
		maxRetries:    maxRetries,
		retryInterval: retryInterval,
		tracer:        otel.Tracer("jimu.httpclient"),
	}
}

// Do 执行请求：注入 traceparent，对网络错误与 5xx 指数退避重试。
// 4xx 与 2xx 不重试；请求体无法重放（GetBody 为 nil）时放弃重试。
func (c *Client) Do(ctx context.Context, req *http.Request) (*http.Response, error) {
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
