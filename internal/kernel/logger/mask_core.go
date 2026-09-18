package logger

import (
	"jimu/internal/kernel/mask"

	"go.uber.org/zap/zapcore"
)

// maskingCore 包装 zapcore.Core，在写入前按字段名脱敏敏感值：
// 凭证类整体替换为 ***，邮箱/手机号等 PII 部分保留。
// 对所有下游 sink（文件、stdout、OTLP 导出）统一生效，避免 PII 外泄。
type maskingCore struct {
	zapcore.Core
}

// WithMasking 返回带脱敏能力的 Core；core 为 nil 时原样返回
func WithMasking(core zapcore.Core) zapcore.Core {
	if core == nil {
		return core
	}
	return &maskingCore{Core: core}
}

func (c *maskingCore) With(fields []zapcore.Field) zapcore.Core {
	return &maskingCore{Core: c.Core.With(maskFields(fields))}
}

// Check 必须返回包装后的 Core，否则 Write 会绕过脱敏直接落到内层 Core
func (c *maskingCore) Check(ent zapcore.Entry, ce *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	if c.Enabled(ent.Level) {
		return ce.AddCore(ent, c)
	}
	return ce
}

func (c *maskingCore) Write(ent zapcore.Entry, fields []zapcore.Field) error {
	return c.Core.Write(ent, maskFields(fields))
}

// maskFields 就地改写字段值（字段由调用方临时构造，改写不影响业务数据）
func maskFields(fields []zapcore.Field) []zapcore.Field {
	for i := range fields {
		f := &fields[i]
		switch f.Type {
		case zapcore.StringType:
			if masked, ok := mask.RedactByKey(f.Key, f.String); ok {
				f.String = masked
			}
		case zapcore.ReflectType:
			switch v := f.Interface.(type) {
			case string:
				if masked, ok := mask.RedactByKey(f.Key, v); ok {
					f.Interface = masked
				}
			case map[string]any:
				f.Interface = mask.Map(v)
			}
		}
	}
	return fields
}
