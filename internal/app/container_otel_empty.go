//go:build jimu_no_otel

package app

import (
	"context"

	"jimu/internal/config"

	"go.uber.org/zap/zapcore"
)

// otelState 是未编译 OTel 的构建的空实现。
type otelState struct{}

func newOTELState(*config.Config) otelState { return otelState{} }

func (*otelState) extraCores() []zapcore.Core { return nil }

func (*otelState) shutdown(context.Context) error { return nil }
