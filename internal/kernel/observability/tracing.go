package observability

import (
	"context"
	"encoding/base64"
	"fmt"
	"time"

	"jimu/internal/config"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	sdkresource "go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.27.0"
)

// otlpHeaders 构造 OpenObserve OTLP gRPC 认证与组织 metadata：
//   - organization: 组织标识（默认 default）
//   - stream-name: 目标 stream（logs/traces 各自指定；空则省略，落 OpenObserve 默认流 default）
//   - authorization: Basic base64(email:password)，AuthEmail 非空时携带
func otlpHeaders(cfg TracingConfig, streamName string) map[string]string {
	org := cfg.OrgID
	if org == "" {
		org = "default"
	}
	headers := map[string]string{"organization": org}
	if streamName != "" {
		headers["stream-name"] = streamName
	}
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
		otlptracegrpc.WithHeaders(otlpHeaders(cfg, cfg.TracesStreamName)),
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

// TracingConfig 是 config.OTELConfig 的别名：配置结构归 config 包，
// observability 只消费它，从而让 config 不必反向依赖本包。
type TracingConfig = config.OTELConfig

// DefaultTracingConfig 见 config.DefaultOTELConfig。
func DefaultTracingConfig() TracingConfig { return config.DefaultOTELConfig() }
