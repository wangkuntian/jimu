package grpc

import (
	"context"
	"errors"
	"fmt"
	"time"

	"jimu/internal/platform/logger"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Reporter 错误上报接口（避免反向依赖 platform/reporter 包，与 HTTP Recovery 同构）
type Reporter interface {
	Report(ctx context.Context, err error, attrs ...string)
}

// panicErr 已上报的 panic 包装，供 recovery 拦截器识别
var errPanicRecovered = errors.New("grpc server panic recovered")

// serverRecoveryInterceptor 捕获业务 handler 的 panic 并转为 codes.Internal。
// 没有这层拦截器时，handler 内的 panic 会直接终止整个进程（grpc-go 不 recover）。
func serverRecoveryInterceptor(log *logger.Logger, report Reporter) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
		defer func() {
			if r := recover(); r != nil {
				cause := fmt.Errorf("%w: %v", errPanicRecovered, r)
				if report != nil {
					report.Report(ctx, cause, "grpc_method", info.FullMethod)
				}
				if log != nil {
					log.Errorw("grpc handler panic recovered", "method", info.FullMethod, "error", fmt.Sprint(r))
				}
				err = status.Errorf(codes.Internal, "internal server error")
			}
		}()
		return handler(ctx, req)
	}
}

// serverMetricsInterceptor 记录请求量/错误量/在途数与耗时。
func serverMetricsInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		service, method := splitFullMethod(info.FullMethod)
		grpcServerInFlight.WithLabelValues(service).Inc()
		defer grpcServerInFlight.WithLabelValues(service).Dec()

		start := time.Now()
		resp, err := handler(ctx, req)
		grpcServerRequestDuration.WithLabelValues(service, method).Observe(time.Since(start).Seconds())

		code := status.Code(err)
		grpcServerRequestsTotal.WithLabelValues(service, method, code.String()).Inc()
		if err != nil && !errors.Is(err, context.Canceled) {
			grpcServerErrorsTotal.WithLabelValues(service, method).Inc()
		}
		return resp, err
	}
}

// serverTimeoutInterceptor 为 handler 设置执行超时（timeout<=0 表示不限制）。
// 超时后 handler 应自行响应 ctx.Done；ctx 已被取消时统一返回 DeadlineExceeded。
func serverTimeoutInterceptor(timeout time.Duration) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		if timeout <= 0 {
			return handler(ctx, req)
		}
		timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()

		resp, err := handler(timeoutCtx, req)
		if err != nil && errors.Is(timeoutCtx.Err(), context.DeadlineExceeded) && status.Code(err) == codes.Unknown {
			return nil, status.Error(codes.DeadlineExceeded, "grpc handler timeout")
		}
		return resp, err
	}
}

// splitFullMethod 把 /pkg.Service/Method 拆成 service 与 method 标签。
func splitFullMethod(fullMethod string) (string, string) {
	name := fullMethod
	if len(name) > 0 && name[0] == '/' {
		name = name[1:]
	}
	for i := 0; i < len(name); i++ {
		if name[i] == '/' {
			return name[:i], name[i+1:]
		}
	}
	return name, ""
}

// --- 指标 ---

var (
	grpcServerRequestsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "jimu",
		Subsystem: "grpc_server",
		Name:      "requests_total",
		Help:      "Total gRPC server requests",
	}, []string{"service", "method", "code"})

	grpcServerErrorsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "jimu",
		Subsystem: "grpc_server",
		Name:      "errors_total",
		Help:      "Total gRPC server errors",
	}, []string{"service", "method"})

	grpcServerInFlight = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "jimu",
		Subsystem: "grpc_server",
		Name:      "in_flight",
		Help:      "Number of gRPC server requests in flight",
	}, []string{"service"})

	grpcServerRequestDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "jimu",
		Subsystem: "grpc_server",
		Name:      "request_duration_seconds",
		Help:      "gRPC server request duration in seconds",
		Buckets:   prometheus.DefBuckets,
	}, []string{"service", "method"})
)
