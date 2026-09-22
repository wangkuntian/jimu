package enterprise

import (
	"testing"

	"jimu/internal/assembly"
	"jimu/internal/capabilities/storage"

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

// TestEnterpriseDriverSelection 钉住 enterprise 形态的驱动选中集（T1 按现状声明：
// 仍是全量驱动，收敛到 local + csv 是 Task 5 的行为变更）。
func TestEnterpriseDriverSelection(t *testing.T) {
	assert.Equal(t, map[string][]string{
		"storage": {"local", "s3"},
		"dataops": {"csv", "excel"},
	}, driverSelection(Assembly()))
}

// TestEnterpriseCompiledStorageDrivers 钉住进程内注册表：本形态实际编译进来的 storage
// 驱动类型必须与下沉前的行为一致（T2 仍全量）。注意 s3 驱动包一个包承载 s3/minio/oss
// 三种 S3 兼容类型（与下沉前核心 switch 的四个分支逐值一致），故注册表为 4 项，而驱动
// **包**集合仍是 assembly 声明的 {local, s3}；收敛为 local-only 是 Task 5 的行为变更。
func TestEnterpriseCompiledStorageDrivers(t *testing.T) {
	assert.Equal(t, []storage.StorageType{
		storage.StorageTypeLocal,
		storage.StorageTypeMinIO,
		storage.StorageTypeOSS,
		storage.StorageTypeS3,
	}, storage.RegisteredTypes())
}

// driverSelection 汇总清单里各能力的驱动选中集（测试辅助）。
func driverSelection(a assembly.Assembly) map[string][]string {
	out := map[string][]string{}
	for _, c := range a.Capabilities {
		if len(c.Drivers) > 0 {
			out[c.Descriptor.Name] = c.Drivers
		}
	}
	return out
}

func capabilityNames(a assembly.Assembly) []string {
	out := make([]string, 0, len(a.Capabilities))
	for _, c := range a.Capabilities {
		out = append(out, c.Descriptor.Name)
	}
	return out
}
