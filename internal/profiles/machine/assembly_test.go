package machine

import (
	"testing"

	"jimu/internal/assembly"
	"jimu/internal/capabilities/queue"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMachineAssemblyExcludesAuth 设计 §2 ④：无界面、服务间调用形态没有登录/注册/会话
// 端点，因而不得装配 auth（也不得装配其软依赖 mfa/tenant）。
func TestMachineAssemblyExcludesAuth(t *testing.T) {
	names := capabilityNames(Assembly())
	assert.NotContains(t, names, "auth")
	assert.NotContains(t, names, "mfa")
	assert.NotContains(t, names, "tenant")
	assert.NotContains(t, names, "console")
}

// TestMachineAssemblyShape 名字集合 = 设计 §2 ④ 到仓库能力名的映射：catalog 取
// user/access/apikey，非 catalog 取 grpc/encryption（无 auth 时由 apikey 承担受保护路由）。
func TestMachineAssemblyShape(t *testing.T) {
	require.ElementsMatch(t,
		[]string{"user", "access", "apikey", "grpc", "encryption"},
		capabilityNames(Assembly()),
	)
}

// TestMachineAssemblyModulesAreWired 每个条目都必须给出 Wire。
func TestMachineAssemblyModulesAreWired(t *testing.T) {
	for _, c := range Assembly().Capabilities {
		require.NotNil(t, c.Wire, "capability %q has no Wire", c.Descriptor.Name)
	}
}

// TestMachineCompilesNoQueueDriver 钉住进程内队列驱动注册表为空（与 minimal 同款）：
// 本形态不装配 queue 能力，闭包里不得被动编进任何队列驱动。
func TestMachineCompilesNoQueueDriver(t *testing.T) {
	assert.Empty(t, queue.RegisteredTypes(), "形态未装配 queue，不得编进任何队列驱动")
}

func capabilityNames(a assembly.Assembly) []string {
	out := make([]string, 0, len(a.Capabilities))
	for _, c := range a.Capabilities {
		out = append(out, c.Descriptor.Name)
	}
	return out
}
