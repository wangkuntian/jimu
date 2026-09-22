package enterprise

import (
	"testing"

	"jimu/internal/assembly"
	"jimu/internal/capabilities/dataops/exporter"
	"jimu/internal/capabilities/dataops/importer"
	"jimu/internal/capabilities/queue"
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

// TestEnterpriseDriverSelection 钉住 enterprise 形态收敛后的驱动选中集与进程内注册表
// （Task 5 行为变更）：本形态只编入 local + csv。声明（assembly.go 的 Drivers）与 blank
// import（drivers.go）必须逐值一致，只钉声明或只钉注册表都挡不住单侧漂移。
func TestEnterpriseDriverSelection(t *testing.T) {
	assert.Equal(t, map[string][]string{"storage": {"local"}, "dataops": {"csv"}}, driverSelection(Assembly()))
	assert.Equal(t, []storage.StorageType{storage.StorageTypeLocal}, storage.RegisteredTypes())
	assert.Equal(t, []importer.Format{importer.FormatCSV}, importer.RegisteredFormats())
	assert.Equal(t, []exporter.Format{exporter.FormatCSV}, exporter.RegisteredFormats())
	assert.Empty(t, queue.RegisteredTypes())
}

// TestEnterpriseRejectsUncompiledStorageDriver 直接在本形态的**测试二进制**里构造 s3：
// drivers.go 不再 blank import storage/s3，注册表里没有 s3，构造即 fail-closed。用进程内
// 聚焦断言而不是启动冒烟 —— assembly.Run 先建内核容器（DB 连接）再逐个 Wire，本机无 DB 时
// 根本走不到 storage.Wire，冒烟证不到驱动语义。
func TestEnterpriseRejectsUncompiledStorageDriver(t *testing.T) {
	_, err := storage.New(storage.Config{Type: storage.StorageTypeS3})
	require.ErrorContains(t, err, `storage driver "s3" is not compiled into this build (compiled: local)`)
}

// TestEnterpriseRejectsUncompiledDataopsFormat 同理：xlsx（excel 驱动）未编入本形态，
// 请求期取实现即 fail-closed，不静默回退到其它格式。
func TestEnterpriseRejectsUncompiledDataopsFormat(t *testing.T) {
	_, err := importer.Get(importer.FormatExcel)
	require.ErrorContains(t, err, `import format "xlsx" is not compiled into this build (compiled: csv)`)
}

// TestEnterpriseCompilesNoQueueDriver 钉住进程内队列驱动注册表为空：enterprise 不装配
// queue 能力（只经 outbox/auth/user 的类型级传递残留引用核心 queue 包），闭包里不得
// 被动编进任何队列驱动 —— kafka/amqp 依赖随之退出本形态。上面的
// TestEnterpriseDriverSelection 也一并钉了这条，此处保留独立用例是为了让失败信息直接
// 指向「形态未装配 queue」这个原因。
func TestEnterpriseCompilesNoQueueDriver(t *testing.T) {
	assert.Empty(t, queue.RegisteredTypes(), "形态未装配 queue，不得编进任何队列驱动")
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
