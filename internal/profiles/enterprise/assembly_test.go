package enterprise

import (
	"testing"

	"jimu/internal/assembly"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestEnterpriseAssemblyIsSingleTenantWithoutMFA 设计 §2 ③：公司内部系统、单租户
// （tid=0 平台级视角）—— 无 tenant/mfa，含 console/audit/oauth/dataops/storage。
func TestEnterpriseAssemblyIsSingleTenantWithoutMFA(t *testing.T) {
	names := capabilityNames(Assembly())
	assert.NotContains(t, names, "tenant")
	assert.NotContains(t, names, "mfa")
	for _, want := range []string{"console", "audit", "oauth", "dataops", "storage"} {
		assert.Contains(t, names, want)
	}
}

// TestEnterpriseAssemblyShape 名字集合 = 设计 §2 ③ 到仓库能力名的映射：minimal 的
// user/access/auth + console/audit/oauth/dataops，非 catalog 取 minimal + storage。
func TestEnterpriseAssemblyShape(t *testing.T) {
	require.ElementsMatch(t,
		[]string{"user", "access", "auth", "console", "audit", "oauth", "dataops", "notification", "encryption", "storage"},
		capabilityNames(Assembly()),
	)
}

// TestEnterpriseAssemblyModulesAreWired 每个条目都必须给出 Wire。
func TestEnterpriseAssemblyModulesAreWired(t *testing.T) {
	for _, c := range Assembly().Capabilities {
		require.NotNil(t, c.Wire, "capability %q has no Wire", c.Descriptor.Name)
	}
}

func capabilityNames(a assembly.Assembly) []string {
	out := make([]string, 0, len(a.Capabilities))
	for _, c := range a.Capabilities {
		out = append(out, c.Descriptor.Name)
	}
	return out
}
