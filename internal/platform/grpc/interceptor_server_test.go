package grpc

import (
	"context"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

// fakeReporter 记录上报的 panic
type fakeReporter struct {
	mu    sync.Mutex
	errs  []error
	attrs [][]string
}

func (f *fakeReporter) Report(_ context.Context, err error, attrs ...string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.errs = append(f.errs, err)
	f.attrs = append(f.attrs, attrs)
}

func (f *fakeReporter) reported() ([]error, [][]string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]error{}, f.errs...), append([][]string{}, f.attrs...)
}

// startInterceptorTCPServer 用真实 TCP 监听起测试 server（挂载三个服务端拦截器），返回地址。
func startInterceptorTCPServer(t *testing.T, handler grpc.UnaryHandler, reporters ...Reporter) string {
	t.Helper()
	srv := grpc.NewServer(grpc.ChainUnaryInterceptor(
		serverRecoveryInterceptor(nil, firstReporter(reporters)),
		serverMetricsInterceptor(),
		serverTimeoutInterceptor(50*time.Millisecond),
	))
	srv.RegisterService(interceptorServiceDesc(), &interceptorImpl{handler: handler})
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	go func() { _ = srv.Serve(lis) }()
	t.Cleanup(srv.Stop)
	return lis.Addr().String()
}

// interceptorImpl 免 protoc 的测试服务实现，handler 由各测试提供
type interceptorImpl struct {
	handler grpc.UnaryHandler
}

// interceptorServiceDesc 构造一个免 protoc 的一元方法描述（与 ping.go 的写法一致）
func interceptorServiceDesc() *grpc.ServiceDesc {
	return &grpc.ServiceDesc{
		ServiceName: "jimu.test.Interceptor",
		HandlerType: (*interface{})(nil),
		Methods: []grpc.MethodDesc{{
			MethodName: "Call",
			Handler: func(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
				in := new(emptypb.Empty)
				if err := dec(in); err != nil {
					return nil, err
				}
				impl, ok := srv.(*interceptorImpl)
				if !ok {
					return nil, status.Error(codes.Internal, "server does not implement interceptorImpl")
				}
				handler := func(ctx context.Context, req interface{}) (interface{}, error) {
					return impl.handler(ctx, req)
				}
				if interceptor == nil {
					return handler(ctx, in)
				}
				// 与 ping.go 一致：必须经 interceptor 调用，否则服务端拦截器链不生效
				info := &grpc.UnaryServerInfo{FullMethod: "/jimu.test.Interceptor/Call"}
				return interceptor(ctx, in, info, handler)
			},
		}},
		Metadata: "interceptor_test.proto",
	}
}

func dialInterceptor(t *testing.T, addr string) *grpc.ClientConn {
	t.Helper()
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close() })
	return conn
}

func TestServerRecoveryReturnsInternalAndReports(t *testing.T) {
	reporter := &fakeReporter{}
	addr := startInterceptorTCPServer(t, func(context.Context, interface{}) (interface{}, error) {
		panic("boom")
	}, reporter)

	conn := dialInterceptor(t, addr)
	err := conn.Invoke(context.Background(), "/jimu.test.Interceptor/Call", &emptypb.Empty{}, &emptypb.Empty{})
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err), "panic 应转为 Internal 而不是拖垮进程")

	errs, attrs := reporter.reported()
	require.Len(t, errs, 1)
	assert.Contains(t, errs[0].Error(), "boom")
	require.Len(t, attrs, 1)
	assert.Contains(t, attrs[0], "grpc_method")
}

func TestServerRecoveryWithoutLoggerOrReporter(t *testing.T) {
	interceptor := serverRecoveryInterceptor(nil, nil)
	_, err := interceptor(context.Background(), nil, &grpc.UnaryServerInfo{FullMethod: "/x/y"},
		func(context.Context, interface{}) (interface{}, error) { panic("no deps") })
	assert.Equal(t, codes.Internal, status.Code(err))
}

func TestServerTimeoutReturnsDeadlineExceeded(t *testing.T) {
	addr := startInterceptorTCPServer(t, func(ctx context.Context, _ interface{}) (interface{}, error) {
		<-ctx.Done() // 模拟慢 handler
		return nil, ctx.Err()
	})

	conn := dialInterceptor(t, addr)
	start := time.Now()
	err := conn.Invoke(context.Background(), "/jimu.test.Interceptor/Call", &emptypb.Empty{}, &emptypb.Empty{})
	require.Error(t, err)
	assert.Equal(t, codes.DeadlineExceeded, status.Code(err))
	assert.Less(t, time.Since(start), 2*time.Second, "超时由服务端拦截器兜底")
}

func TestServerTimeoutDisabled(t *testing.T) {
	interceptor := serverTimeoutInterceptor(0)
	called := false
	_, err := interceptor(context.Background(), nil, &grpc.UnaryServerInfo{},
		func(context.Context, interface{}) (interface{}, error) {
			called = true
			return nil, nil
		})
	require.NoError(t, err)
	assert.True(t, called, "timeout=0 时不应设置 deadline")
}

func TestServerMetricsRecorded(t *testing.T) {
	addr := startInterceptorTCPServer(t, func(context.Context, interface{}) (interface{}, error) {
		return &emptypb.Empty{}, nil
	})

	conn := dialInterceptor(t, addr)
	require.NoError(t, conn.Invoke(context.Background(), "/jimu.test.Interceptor/Call", &emptypb.Empty{}, &emptypb.Empty{}))

	counter := grpcServerRequestsTotal.WithLabelValues("jimu.test.Interceptor", "Call", codes.OK.String())
	assert.Greater(t, readCounter(t, counter), float64(0))
	assert.Equal(t, float64(0), readGauge(t, grpcServerInFlight.WithLabelValues("jimu.test.Interceptor")),
		"在途计数应在请求结束后归零")
}

func TestSplitFullMethod(t *testing.T) {
	service, method := splitFullMethod("/pkg.Service/Method")
	assert.Equal(t, "pkg.Service", service)
	assert.Equal(t, "Method", method)

	service, method = splitFullMethod("noslash")
	assert.Equal(t, "noslash", service)
	assert.Empty(t, method)
}

// readCounter 读取 Counter 当前值（Gauge 也满足 Counter 接口，故不共用类型分支）
func readCounter(t *testing.T, c prometheus.Counter) float64 {
	t.Helper()
	var metric = &dto.Metric{}
	require.NoError(t, c.Write(metric))
	return metric.GetCounter().GetValue()
}

// readGauge 读取 Gauge 当前值
func readGauge(t *testing.T, g prometheus.Gauge) float64 {
	t.Helper()
	var metric = &dto.Metric{}
	require.NoError(t, g.Write(metric))
	return metric.GetGauge().GetValue()
}
