package observability

import (
	"context"
	"encoding/base64"
	"fmt"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	sdkresource "go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.27.0"
)

// TracingConfig OpenTelemetry 可观测性配置（追踪/指标/日志统一走 OTLP gRPC）
type TracingConfig struct {
	Enabled        bool    `mapstructure:"enabled"`
	Endpoint       string  `mapstructure:"endpoint"`
	ServiceName    string  `mapstructure:"service_name"`
	ServiceVersion string  `mapstructure:"service_version"`
	SampleRate     float64 `mapstructure:"sample_rate"`
	// MetricsEnabled 是否将 Prometheus 指标转换为 OTLP 推送到 OpenObserve
	MetricsEnabled bool `mapstructure:"metrics_enabled"`
	// LogsEnabled 是否将结构化日志推送到 OpenObserve
	LogsEnabled bool `mapstructure:"logs_enabled"`
	// MetricsInterval 指标推送间隔（秒），<=0 默认 15
	MetricsInterval int `mapstructure:"metrics_interval_sec"`
	// AuthEmail / AuthPassword OpenObserve 账号凭据（Basic Auth，gRPC metadata authorization）。
	// AuthEmail 为空则不携带凭据（OpenObserve 默认拒绝匿名 OTLP 写入）。
	AuthEmail    string `mapstructure:"auth_email"`
	AuthPassword string `mapstructure:"auth_password"`
	// OrgID OpenObserve 组织（gRPC metadata organization；OpenObserve OTLP 默认组织 default）
	OrgID string `mapstructure:"org_id"`
}

// DefaultTracingConfig 返回默认可观测性配置
func DefaultTracingConfig() TracingConfig {
	return TracingConfig{
		Enabled:         false,
		Endpoint:        "localhost:4317",
		ServiceName:     "jimu",
		ServiceVersion:  "dev",
		SampleRate:      1.0,
		MetricsEnabled:  true,
		LogsEnabled:     true,
		MetricsInterval: 15,
		OrgID:           "default",
	}
}

// otlpHeaders 构造 OpenObserve OTLP gRPC 认证与组织 metadata：
//   - organization: 组织标识（默认 default）
//   - authorization: Basic base64(email:password)，AuthEmail 非空时携带
func otlpHeaders(cfg TracingConfig) map[string]string {
	org := cfg.OrgID
	if org == "" {
		org = "default"
	}
	headers := map[string]string{"organization": org}
	if cfg.AuthEmail != "" {
		token := base64.StdEncoding.EncodeToString([]byte(cfg.AuthEmail + ":" + cfg.AuthPassword))
		headers["authorization"] = "Basic " + token
	}
	return headers
}

// InitTracing 初始化 OpenTelemetry 追踪
// 若 Enabled=false，返回 NoOp TracerProvider（零开销）
func InitTracing(ctx context.Context, cfg TracingConfig) (*sdktrace.TracerProvider, error) {
	if !cfg.Enabled {
		// 设置 NoOp provider，所有 Tracer 调用都是空操作
		tp := sdktrace.NewTracerProvider()
		otel.SetTracerProvider(tp)
		return tp, nil
	}

	exporter, err := otlptracegrpc.New(ctx,
		otlptracegrpc.WithEndpoint(cfg.Endpoint),
		otlptracegrpc.WithInsecure(),
		otlptracegrpc.WithHeaders(otlpHeaders(cfg)),
		otlptracegrpc.WithTimeout(10*time.Second),
	)
	if err != nil {
		return nil, fmt.Errorf("create OTLP exporter: %w", err)
	}

	serviceName := cfg.ServiceName
	if serviceName == "" {
		serviceName = "jimu"
	}
	serviceVersion := cfg.ServiceVersion
	if serviceVersion == "" {
		serviceVersion = "dev"
	}

	resource := sdkresource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceName(serviceName),
		semconv.ServiceVersion(serviceVersion),
	)

	sampleRate := cfg.SampleRate
	if sampleRate <= 0 {
		sampleRate = 1.0
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(resource),
		sdktrace.WithSampler(sdktrace.TraceIDRatioBased(sampleRate)),
	)

	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	return tp, nil
}

// TraceFromContext 从 context 提取 W3C traceparent/tracestate，供跨异步边界透传
// （队列消息、Outbox 事件元数据）。追踪禁用时返回空串，零开销。
func TraceFromContext(ctx context.Context) (traceparent, tracestate string) {
	carrier := propagation.MapCarrier{}
	otel.GetTextMapPropagator().Inject(ctx, carrier)
	return carrier.Get("traceparent"), carrier.Get("tracestate")
}

// ContextWithTrace 从 traceparent/tracestate 恢复追踪上下文，链接上游 span。
// 两者为空时原样返回 ctx。
func ContextWithTrace(ctx context.Context, traceparent, tracestate string) context.Context {
	if traceparent == "" && tracestate == "" {
		return ctx
	}
	return otel.GetTextMapPropagator().Extract(ctx, propagation.MapCarrier{
		"traceparent": traceparent,
		"tracestate":  tracestate,
	})
}

// ShutdownTracing 优雅关闭追踪，flush 剩余 span
func ShutdownTracing(ctx context.Context, tp *sdktrace.TracerProvider) error {
	if tp == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	return tp.Shutdown(ctx)
}
