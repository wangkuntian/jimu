package minimal

import (
	"testing"

	"jimu/internal/assembly"
	"jimu/internal/capabilities/queue"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMinimalAssemblyExcludesTenantAndMFA 设计 §2 ①：internal 微服务/新项目起点形态
// 不含租户、MFA、审计、控制台。
func TestMinimalAssemblyExcludesTenantAndMFA(t *testing.T) {
	names := capabilityNames(Assembly())
	assert.NotContains(t, names, "tenant")
	assert.NotContains(t, names, "mfa")
	assert.NotContains(t, names, "audit")
	assert.NotContains(t, names, "console")
}

// TestMinimalAssemblyShape 名字集合 = 设计 §2 ① 到仓库能力名的映射：catalog 取
// user/access/auth，非 catalog 取 notification/encryption（零配置日志渠道兜底 + 字段加密）。
func TestMinimalAssemblyShape(t *testing.T) {
	require.ElementsMatch(t,
		[]string{"user", "access", "auth", "notification", "encryption"},
		capabilityNames(Assembly()),
	)
}

// TestMinimalAssemblyModulesAreWired 每个条目都必须给出 Wire。
func TestMinimalAssemblyModulesAreWired(t *testing.T) {
	for _, c := range Assembly().Capabilities {
		require.NotNil(t, c.Wire, "capability %q has no Wire", c.Descriptor.Name)
	}
}

// TestMinimalCompilesNoQueueDriver 钉住进程内队列驱动注册表为空：本形态不装配 queue
// 能力，闭包里不得被动编进任何队列驱动（kafka/amqp 依赖随之退出轻形态）。驱动只在
// profiles/<name>/drivers.go 的 blank import 中注册，测试文件 import 核心 queue 包
// 不影响生产 import 图。
func TestMinimalCompilesNoQueueDriver(t *testing.T) {
	assert.Empty(t, queue.RegisteredTypes(), "形态未装配 queue，不得编进任何队列驱动")
}

func capabilityNames(a assembly.Assembly) []string {
	out := make([]string, 0, len(a.Capabilities))
	for _, c := range a.Capabilities {
		out = append(out, c.Descriptor.Name)
	}
	return out
}
