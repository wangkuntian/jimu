package http

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"jimu/internal/platform/logger"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/trace"
)

// Client 封装 *http.Client，集成超时、重试、追踪传播
type Client struct {
	client     *http.Client
	logger     *logger.Logger
	maxRetries int
	baseURL    string
}

// ClientOption 配置 Client 的函数选项
type ClientOption func(*Client)

// WithTimeout 设置超时
func WithTimeout(d time.Duration) ClientOption {
	return func(c *Client) {
		c.client.Timeout = d
	}
}

// WithMaxRetries 设置最大重试次数
func WithMaxRetries(n int) ClientOption {
	return func(c *Client) {
		c.maxRetries = n
	}
}

// WithBaseURL 设置基础 URL
func WithBaseURL(baseURL string) ClientOption {
	return func(c *Client) {
		c.baseURL = baseURL
	}
}

// NewClient 创建 HTTP Client
func NewClient(log *logger.Logger, opts ...ClientOption) *Client {
	transport := otelhttp.NewTransport(http.DefaultTransport,
		otelhttp.WithSpanNameFormatter(func(operation string, r *http.Request) string {
			return fmt.Sprintf("%s %s", r.Method, r.URL.Path)
		}),
	)

	c := &Client{
		client: &http.Client{
			Timeout:   30 * time.Second,
			Transport: transport,
		},
		logger:     log,
		maxRetries: 3,
	}

	for _, opt := range opts {
		opt(c)
	}

	return c
}

// Do 执行 HTTP 请求，带重试
func (c *Client) Do(req *http.Request) (*http.Response, error) {
	var lastErr error

	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		if attempt > 0 {
			// 指数退避
			backoff := time.Duration(1<<uint(attempt-1)) * time.Second
			c.logger.Debug("http client retrying",
				"method", req.Method,
				"url", req.URL.String(),
				"attempt", attempt,
				"backoff", backoff.String(),
			)
			time.Sleep(backoff)
		}

		// 克隆请求体（重试时需要重新读取）
		var bodyBytes []byte
		if req.Body != nil {
			bodyBytes, _ = io.ReadAll(req.Body)
			req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
		}

		resp, err := c.client.Do(req)
		if err != nil {
			lastErr = err
			c.logger.Debug("http client error",
				"method", req.Method,
				"url", req.URL.String(),
				"error", err.Error(),
			)
			continue
		}

		// 仅对 5xx 和网络错误重试
		if resp.StatusCode >= 500 {
			resp.Body.Close()
			lastErr = fmt.Errorf("server error: %d", resp.StatusCode)
			continue
		}

		return resp, nil
	}

	return nil, fmt.Errorf("max retries exceeded: %w", lastErr)
}

// Get 发送 GET 请求
func (c *Client) Get(ctx context.Context, url string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+url, nil)
	if err != nil {
		return nil, err
	}
	return c.Do(req)
}

// Post 发送 POST 请求
func (c *Client) Post(ctx context.Context, url string, body interface{}) (*http.Response, error) {
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+url, bodyReader)
	if err != nil {
		return nil, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return c.Do(req)
}

// DoJSON 执行请求并将响应解析为 JSON
func (c *Client) DoJSON(req *http.Request, dest interface{}) error {
	resp, err := c.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("http %d: %s", resp.StatusCode, string(body))
	}

	if dest != nil {
		if err := json.NewDecoder(resp.Body).Decode(dest); err != nil {
			return fmt.Errorf("decode response: %w", err)
		}
	}
	return nil
}

// TraceID 从 context 获取当前 trace_id（用于日志关联）
func (c *Client) TraceID(ctx context.Context) string {
	spanContext := trace.SpanContextFromContext(ctx)
	if spanContext.IsValid() {
		return spanContext.TraceID().String()
	}
	return ""
}

var _ = bytes.NewBuffer
