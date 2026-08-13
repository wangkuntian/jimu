// internal/platform/httpclient/client.go
package httpclient

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

const (
	defaultTimeout       = 10 * time.Second
	defaultMaxRetries    = 2
	defaultRetryInterval = 200 * time.Millisecond
	defaultMaxFailures   = 5
	defaultResetTimeout  = 30 * time.Second
)

// ErrCircuitOpen 熔断开启时请求被快速拒绝
var ErrCircuitOpen = errors.New("httpclient: circuit breaker open")

var (
	circuitRejectedTotal = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: "jimu",
		Subsystem: "httpclient",
		Name:      "circuit_rejected_total",
		Help:      "Total number of requests rejected by circuit breaker while open",
	})
	circuitOpen = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: "jimu",
		Subsystem: "httpclient",
		Name:      "circuit_open",
		Help:      "Whether circuit breaker is open (1) or closed (0)",
	})
)

// Config 出站 HTTP client 配置
type Config struct {
	TimeoutSec      int `mapstructure:"timeout_sec"`       // 单次请求超时（秒），0 用默认 10
	MaxRetries      int `mapstructure:"max_retries"`       // 失败重试次数，0 用默认 2
	RetryIntervalMS int `mapstructure:"retry_interval_ms"` // 重试基础间隔（毫秒），0 用默认 200
	MaxFailures     int `mapstructure:"max_failures"`      // 连续失败次数达此值熔断，0 用默认 5
	ResetTimeoutMS  int `mapstructure:"reset_timeout_ms"`  // 熔断后冷却时长（毫秒），0 用默认 30_000
}

// Client 统一出站 HTTP client：超时 + 指数退避重试（仅网络错误与 5xx）+ trace 注入 + 熔断
type Client struct {
	hc            *http.Client
	maxRetries    int
	retryInterval time.Duration
	tracer        trace.Tracer
	circuit       *circuit
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
	return &Client{
		hc:            &http.Client{Timeout: timeout},
		maxRetries:    maxRetries,
		retryInterval: retryInterval,
		tracer:        otel.Tracer("jimu.httpclient"),
		circuit:       newCircuit(maxFailures, resetTimeout),
	}
}

// Do 执行请求：注入 traceparent，对网络错误与 5xx 指数退避重试，熔断保护。
// 4xx 与 2xx 不重试；请求体无法重放（GetBody 为 nil）时放弃重试。
// 熔断开启时直接返回 ErrCircuitOpen，不发起请求。
func (c *Client) Do(ctx context.Context, req *http.Request) (*http.Response, error) {
	if !c.circuit.allow() {
		circuitRejectedTotal.Inc()
		return nil, ErrCircuitOpen
	}
	resp, err := c.do(ctx, req)
	if err != nil {
		c.circuit.record(false)
		return nil, err
	}
	c.circuit.record(true)
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
type circuitState int

const (
	stateClosed circuitState = iota
	stateOpen
	stateHalfOpen
)

type circuit struct {
	mu            sync.Mutex
	state         circuitState
	consecutive   int
	probeInFlight bool
	openedAt      time.Time
	maxFailures   int
	resetTimeout  time.Duration
}

func newCircuit(maxFailures int, resetTimeout time.Duration) *circuit {
	return &circuit{state: stateClosed, maxFailures: maxFailures, resetTimeout: resetTimeout}
}

// allow 是否允许发请求：closed 全放行；open 冷却结束后转 half-open 放行单次探测；
// half-open 仅放行一个在途探测请求，避免并发打爆恢复中的下游。
func (c *circuit) allow() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	switch c.state {
	case stateClosed:
		return true
	case stateOpen:
		if time.Since(c.openedAt) >= c.resetTimeout {
			c.state = stateHalfOpen
			c.probeInFlight = true
			return true
		}
		return false
	default: // stateHalfOpen
		if c.probeInFlight {
			return false
		}
		c.probeInFlight = true
		return true
	}
}

// record 记录一次调用结果，驱动状态转移。
func (c *circuit) record(success bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	switch c.state {
	case stateClosed:
		if success {
			c.consecutive = 0
			return
		}
		c.consecutive++
		if c.consecutive >= c.maxFailures {
			c.open()
		}
	case stateHalfOpen:
		c.probeInFlight = false
		if success {
			c.state = stateClosed
			c.consecutive = 0
			circuitOpen.Set(0)
		} else {
			c.open()
		}
	}
}

func (c *circuit) open() {
	c.state = stateOpen
	c.consecutive = 0
	c.probeInFlight = false
	c.openedAt = time.Now()
	circuitOpen.Set(1)
}
