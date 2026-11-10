//go:build jimu_no_otel

package app

// setupObservability 在未编译 OTel 的构建里是 no-op。
func setupObservability(*Container) error { return nil }
