package httpclient

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

func TestDoRetriesOn5xx(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	c := New(Config{MaxRetries: 2, RetryIntervalMS: 1})
	req, _ := http.NewRequest(http.MethodGet, srv.URL, nil)
	resp, err := c.Do(context.Background(), req)
	require.Error(t, err)
	assert.Nil(t, resp)
	assert.Equal(t, int32(3), calls.Load()) // 初始 + 2 次重试
}

func TestDoNoRetryOn4xx(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer srv.Close()

	c := New(Config{MaxRetries: 2, RetryIntervalMS: 1})
	req, _ := http.NewRequest(http.MethodGet, srv.URL, nil)
	resp, err := c.Do(context.Background(), req)
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	assert.Equal(t, int32(1), calls.Load())
}

func TestDoSucceedsAfterTransient5xx(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if calls.Add(1) < 2 {
			w.WriteHeader(http.StatusBadGateway)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := New(Config{MaxRetries: 2, RetryIntervalMS: 1})
	req, _ := http.NewRequest(http.MethodGet, srv.URL, nil)
	resp, err := c.Do(context.Background(), req)
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, int32(2), calls.Load())
}

func TestDoCancelsDuringRetryWait(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	c := New(Config{MaxRetries: 5, RetryIntervalMS: 100})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	req, _ := http.NewRequest(http.MethodGet, srv.URL, nil)
	go func() {
		time.Sleep(30 * time.Millisecond)
		cancel()
	}()
	_, err := c.Do(ctx, req)
	require.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)
}

func TestDoInjectsTraceParent(t *testing.T) {
	otel.SetTextMapPropagator(propagation.TraceContext{})

	var gotTrace string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotTrace = r.Header.Get("traceparent")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	// 构造有效 span context（默认全局无 active span，TraceContext 传播器不会注入）
	sc := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    trace.TraceID{0x01},
		SpanID:     trace.SpanID{0x01},
		TraceFlags: trace.FlagsSampled,
	})
	ctx := trace.ContextWithSpanContext(context.Background(), sc)

	c := New(Config{})
	req, _ := http.NewRequest(http.MethodGet, srv.URL, nil)
	resp, err := c.Do(ctx, req)
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.NotEmpty(t, gotTrace)
}

func TestCircuitOpensAfterConsecutiveFailures(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	c := New(Config{MaxRetries: -1, MaxFailures: 2, RetryIntervalMS: 1})
	req, _ := http.NewRequest(http.MethodGet, srv.URL, nil)

	// 连续 2 次失败触发熔断
	for i := 0; i < 2; i++ {
		_, err := c.Do(context.Background(), req)
		require.Error(t, err)
	}

	// 熔断开启：第三次快速失败，不触达服务端
	_, err := c.Do(context.Background(), req)
	require.ErrorIs(t, err, ErrCircuitOpen)
	assert.Equal(t, int32(2), calls.Load())
}

func TestCircuitRecoversAfterCooldown(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if calls.Add(1) <= 2 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := New(Config{MaxRetries: -1, MaxFailures: 2, RetryIntervalMS: 1, ResetTimeoutMS: 50})
	req, _ := http.NewRequest(http.MethodGet, srv.URL, nil)

	// 2 次失败触发熔断
	for i := 0; i < 2; i++ {
		_, err := c.Do(context.Background(), req)
		require.Error(t, err)
	}
	_, err := c.Do(context.Background(), req)
	require.ErrorIs(t, err, ErrCircuitOpen)

	// 冷却结束后 half-open 放行单次探测，下游恢复则重新闭合
	time.Sleep(60 * time.Millisecond)
	resp, err := c.Do(context.Background(), req)
	require.NoError(t, err)
	resp.Body.Close()

	resp2, err := c.Do(context.Background(), req)
	require.NoError(t, err)
	resp2.Body.Close()

	assert.Equal(t, int32(4), calls.Load()) // 2 失败 + 探测 + 恢复后
}

func TestCircuitHalfOpenOnlyAllowsProbe(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	c := New(Config{MaxRetries: -1, MaxFailures: 1, RetryIntervalMS: 1, ResetTimeoutMS: 10})
	req, _ := http.NewRequest(http.MethodGet, srv.URL, nil)

	_, err := c.Do(context.Background(), req)
	require.Error(t, err)

	// 冷却后进入 half-open：放行单次探测，其余并发请求被拒绝
	time.Sleep(20 * time.Millisecond)
	var rejected, allowed atomic.Int32
	done := make(chan struct{})
	for i := 0; i < 8; i++ {
		go func() {
			defer func() { done <- struct{}{} }()
			_, err := c.Do(context.Background(), req)
			if err != nil {
				if errors.Is(err, ErrCircuitOpen) {
					rejected.Add(1)
				}
			} else {
				allowed.Add(1)
			}
		}()
	}
	for i := 0; i < 8; i++ {
		<-done
	}
	assert.GreaterOrEqual(t, rejected.Load(), int32(1)) // 至少一个被熔断拒绝
	// 探测请求（1 次）触达服务端：之前 1 次失败 + 探测 = 2
	assert.Equal(t, int32(2), calls.Load())
}

func TestDoRateLimitsPerHost(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	// rate=1/s, burst=1：首个请求消费唯一令牌，第二个请求等待被 ctx 超时打断
	c := New(Config{MaxRetries: -1, RateLimitRate: 1, RateLimitBurst: 1})
	req, _ := http.NewRequest(http.MethodGet, srv.URL, nil)

	resp, err := c.Do(context.Background(), req)
	require.NoError(t, err)
	resp.Body.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	_, err = c.Do(ctx, req)
	require.Error(t, err)
	assert.ErrorContains(t, err, "would exceed context deadline")
	assert.Equal(t, int32(1), calls.Load())
}

func TestDoRateLimitIndependentPerHost(t *testing.T) {
	srv1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv1.Close()
	srv2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv2.Close()

	// 两个 host 各自独立限流：host1 消费令牌不影响 host2
	c := New(Config{MaxRetries: -1, RateLimitRate: 1, RateLimitBurst: 1})

	req1, _ := http.NewRequest(http.MethodGet, srv1.URL, nil)
	resp, err := c.Do(context.Background(), req1)
	require.NoError(t, err)
	resp.Body.Close()

	// host2 首次请求不受 host1 已消费令牌影响
	req2, _ := http.NewRequest(http.MethodGet, srv2.URL, nil)
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	resp2, err := c.Do(ctx, req2)
	require.NoError(t, err)
	resp2.Body.Close()
}
