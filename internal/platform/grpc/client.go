// Package grpc 提供框架级 gRPC 通信能力（与 HTTP 双栈并存）：
//   - Server：gRPC server 生命周期管理（见 server.go）
//   - Client：统一出站 gRPC 客户端封装（本文件）
//
// Client 对齐 httpclient 的框架风格：连接管理 + 超时 + 指数退避重试（幂等安全状态码）+
// 恢复拦截器 + Prometheus 指标（jimu_grpc_client_*），支持 TLS/insecure 连接。
package grpc

import (
	"context"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

// 默认配置常量
const (
	defaultClientDialTimeout = 10 * time.Second
	defaultClientCallTimeout = 10 * time.Second
	defaultClientMaxRetries  = 2
	defaultClientRetryBase   = 200 * time.Millisecond
	maxRetryInterval         = 3 * time.Second
)

// ClientConfig 出站 gRPC 客户端配置
type ClientConfig struct {
	// 连接目标（"host:port"）。
	Address string
	// 连接超时（0 用默认 10s）
	DialTimeout time.Duration
	// 单次调用超时（0 用默认 10s）
	CallTimeout time.Duration
	// 失败重试次数（仅幂等安全状态码 Unavailable/ResourceExhausted），0 用默认 2
	MaxRetries int
	// 重试基础间隔（0 用默认 200ms，指数退避 ×2，封顶 3s）
	RetryInterval time.Duration
	// 是否启用 TLS（false 用 insecure 明文，适合内网）
	TLS bool
	// 自定义 TLS 凭据（设置后 TLS 必须为 true）
	TLSCredentials credentials.TransportCredentials
	// 服务名（仅用于日志与指标标签）
	ServiceName string
}

// DefaultClientConfig 返回默认 gRPC 客户端配置
func DefaultClientConfig() ClientConfig {
	return ClientConfig{
		DialTimeout:   defaultClientDialTimeout,
		CallTimeout:   defaultClientCallTimeout,
		MaxRetries:    defaultClientMaxRetries,
		RetryInterval: defaultClientRetryBase,
	}
}

// Client 统一出站 gRPC 客户端：连接管理 + 超时 + 重试 + 恢复 + 指标。
// 协程安全，可并发复用同一实例。
type Client struct {
	conn        *grpc.ClientConn
	callTimeout time.Duration
	maxRetries  int
	retryBase   time.Duration
	serviceName string
}

// NewClient 创建 gRPC 客户端并建立连接（懒连接：不阻塞等待就绪，底层自动重连）。
// 返回错误仅发生在本地配置问题，网络不可达由 grpc.NewClient 的重连机制兜底。
func NewClient(cfg ClientConfig) (*Client, error) {
	dialTimeout := cfg.DialTimeout
	if dialTimeout <= 0 {
		dialTimeout = defaultClientDialTimeout
	}
	callTimeout := cfg.CallTimeout
	if callTimeout <= 0 {
		callTimeout = defaultClientCallTimeout
	}
	maxRetries := cfg.MaxRetries
	if maxRetries < 0 {
		maxRetries = defaultClientMaxRetries
	}
	retryBase := cfg.RetryInterval
	if retryBase <= 0 {
		retryBase = defaultClientRetryBase
	}
	if cfg.Address == "" {
		return nil, fmt.Errorf("grpc client: address is required")
	}

	creds := insecure.NewCredentials()
	if cfg.TLS {
		if cfg.TLSCredentials != nil {
			creds = cfg.TLSCredentials
		} else {
			creds = credentials.NewClientTLSFromCert(nil, "")
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), dialTimeout)
	defer cancel()

	conn, err := grpc.NewClient(cfg.Address,
		grpc.WithTransportCredentials(creds),
		grpc.WithChainUnaryInterceptor(
			clientRecoveryInterceptor(),
			clientMetricsInterceptor(cfg.ServiceName),
			clientRetryInterceptor(maxRetries, retryBase, cfg.ServiceName),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("grpc client dial %s: %w", cfg.Address, err)
	}
	_ = ctx

	return &Client{
		conn:        conn,
		callTimeout: callTimeout,
		maxRetries:  maxRetries,
		retryBase:   retryBase,
		serviceName: cfg.ServiceName,
	}, nil
}

// Conn 返回底层连接（业务用生成的强类型客户端：userinfopb.NewUserInfoServiceClient(c.Conn())）。
func (c *Client) Conn() *grpc.ClientConn { return c.conn }

// Invoke 带默认超时调用一元 RPC。业务代码优先用生成的强类型客户端，Invoke 供动态调用与测试。
func (c *Client) Invoke(ctx context.Context, method string, args, reply interface{}, opts ...grpc.CallOption) error {
	callCtx, cancel := context.WithTimeout(ctx, c.callTimeout)
	defer cancel()
	return c.conn.Invoke(callCtx, method, args, reply, opts...)
}

// Close 关闭连接（幂等；重复调用安全）。
// grpc-go 对已关闭连接再次 Close 返回错误，这里统一归一化为 nil，避免调用方误判。
func (c *Client) Close() error {
	if c.conn == nil {
		return nil
	}
	_ = c.conn.Close()
	return nil
}

// State 返回当前连接状态（诊断用）。
func (c *Client) State() connectivity.State {
	return c.conn.GetState()
}

// --- 拦截器 ---

// retryableCode 判断状态码是否可安全重试。
func retryableCode(c codes.Code) bool {
	switch c {
	case codes.Unavailable, codes.ResourceExhausted:
		return true
	default:
		return false
	}
}

// clientRetryInterceptor 对可重试错误做指数退避重试（限制总重试次数）。
func clientRetryInterceptor(maxRetries int, retryBase time.Duration, service string) grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		attempt := 0
		for {
			err := invoker(ctx, method, req, reply, cc, opts...)
			if err == nil || attempt >= maxRetries || !retryableCode(status.Code(err)) {
				return err
			}
			attempt++
			delay := retryBase * time.Duration(math.Pow(2, float64(attempt-1)))
			if delay > maxRetryInterval {
				delay = maxRetryInterval
			}
			grpcClientRetriesTotal.WithLabelValues(service, method).Inc()
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
			}
		}
	}
}

// clientRecoveryInterceptor 捕获 panic 转 error，避免客户端进程崩溃。
func clientRecoveryInterceptor() grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) (err error) {
		defer func() {
			if r := recover(); r != nil {
				err = fmt.Errorf("grpc client panic in method %s: %v", method, r)
			}
		}()
		return invoker(ctx, method, req, reply, cc, opts...)
	}
}

// clientMetricsInterceptor 记录请求/错误/在途指标。
func clientMetricsInterceptor(service string) grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		grpcClientInFlight.Inc()
		defer grpcClientInFlight.Dec()
		err := invoker(ctx, method, req, reply, cc, opts...)
		grpcClientRequestsTotal.WithLabelValues(service, method).Inc()
		if err != nil && !errors.Is(err, context.Canceled) {
			grpcClientErrorsTotal.WithLabelValues(service, method).Inc()
		}
		return err
	}
}

// --- 指标 ---

var (
	grpcClientRequestsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "jimu",
		Subsystem: "grpc_client",
		Name:      "requests_total",
		Help:      "Total gRPC client requests",
	}, []string{"service", "method"})

	grpcClientErrorsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "jimu",
		Subsystem: "grpc_client",
		Name:      "errors_total",
		Help:      "Total gRPC client errors",
	}, []string{"service", "method"})

	grpcClientRetriesTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "jimu",
		Subsystem: "grpc_client",
		Name:      "retries_total",
		Help:      "Total gRPC client retry attempts",
	}, []string{"service", "method"})

	grpcClientInFlight = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: "jimu",
		Subsystem: "grpc_client",
		Name:      "in_flight",
		Help:      "Number of gRPC client requests in flight",
	})
)
