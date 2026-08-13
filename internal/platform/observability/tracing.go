package observability

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	sdkresource "go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.27.0"
)

// TracingConfig OpenTelemetry 追踪配置
type TracingConfig struct {
	Enabled        bool    `mapstructure:"enabled"`
	Endpoint       string  `mapstructure:"endpoint"`
	ServiceName    string  `mapstructure:"service_name"`
	ServiceVersion string  `mapstructure:"service_version"`
	SampleRate     float64 `mapstructure:"sample_rate"`
}

// DefaultTracingConfig 返回默认追踪配置
func DefaultTracingConfig() TracingConfig {
	return TracingConfig{
		Enabled:        false,
		Endpoint:       "localhost:4317",
		ServiceName:    "jimu",
		ServiceVersion: "dev",
		SampleRate:     1.0,
	}
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
