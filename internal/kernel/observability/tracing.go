package observability

import (
	"context"
	"encoding/base64"

	"jimu/internal/config"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
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

// TracingConfig 是 config.OTELConfig 的别名：配置结构归 config 包，
// observability 只消费它，从而让 config 不必反向依赖本包。
type TracingConfig = config.OTELConfig

// DefaultTracingConfig 见 config.DefaultOTELConfig。
func DefaultTracingConfig() TracingConfig { return config.DefaultOTELConfig() }
