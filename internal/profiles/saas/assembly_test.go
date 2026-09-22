package saas

import (
	"testing"

	"jimu/internal/assembly"
	"jimu/internal/capabilities/queue"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSaasAssemblyAddsTenantAndAudit 设计 §2 ②：面向外部客户的多租户形态 = minimal +
// tenant（含开通式注册）+ audit。
func TestSaasAssemblyAddsTenantAndAudit(t *testing.T) {
	names := capabilityNames(Assembly())
	assert.Contains(t, names, "tenant")
	assert.Contains(t, names, "audit")
	assert.NotContains(t, names, "console")
	assert.NotContains(t, names, "mfa")
}

// TestSaasAssemblyShape 名字集合 = 设计 §2 ② 到仓库能力名的映射：minimal 的
// user/access/auth + tenant/audit，非 catalog 同 minimal（邮件渠道由 email.enabled 开启）。
func TestSaasAssemblyShape(t *testing.T) {
	require.ElementsMatch(t,
		[]string{"user", "access", "auth", "tenant", "audit", "notification", "encryption"},
		capabilityNames(Assembly()),
	)
}

// TestSaasAssemblyModulesAreWired 每个条目都必须给出 Wire。
func TestSaasAssemblyModulesAreWired(t *testing.T) {
	for _, c := range Assembly().Capabilities {
		require.NotNil(t, c.Wire, "capability %q has no Wire", c.Descriptor.Name)
	}
}

// TestSaasCompilesNoQueueDriver 钉住进程内队列驱动注册表为空（与 minimal 同款）：
// 本形态不装配 queue 能力，闭包里不得被动编进任何队列驱动。
func TestSaasCompilesNoQueueDriver(t *testing.T) {
	assert.Empty(t, queue.RegisteredTypes(), "形态未装配 queue，不得编进任何队列驱动")
}

func capabilityNames(a assembly.Assembly) []string {
	out := make([]string, 0, len(a.Capabilities))
	for _, c := range a.Capabilities {
		out = append(out, c.Descriptor.Name)
	}
	return out
}
