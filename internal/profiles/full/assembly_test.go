package full

import (
	"testing"

	"jimu/internal/assembly"
	"jimu/internal/capabilities/catalog"
	"jimu/internal/capabilities/storage"
	"jimu/internal/capability"
	"jimu/internal/contract"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// nonCatalogEntries 是 full 形态里的非 catalog 条目（P2.4 裁定 7）：它们是
// assembly.Capability 但不是 catalog 成员（catalog 仍 18 项，不入迁移/权限聚合）。
var nonCatalogEntries = []string{
	"storage", "notification", "retention", "ws", "grpc", "apidocs", "encryption",
}

// TestFullAssemblyShape full 形态的名字集合 = catalog 全量 ∪ 固定非 catalog 条目。
//
// 注意（裁定 12）：`Assembly.Capabilities` 的顺序是**装配顺序**——端口提供者必须排在
// 消费者之前（例如 `tenant`/`access` 必须先于 `user`，而 catalog 顺序里 `user` 在最前），
// 因此这里只断言**集合**而不是顺序。catalog 顺序只用于迁移与启用闭包，由
// `capability.Resolve` 保证（它按传入切片顺序返回并补齐硬依赖）。
func TestFullAssemblyShape(t *testing.T) {
	cat := catalog.Names() // 18 项
	extra := map[string]bool{}
	for _, n := range nonCatalogEntries {
		extra[n] = true
	}
	got := capabilityNames(Assembly())
	require.Len(t, got, len(cat)+len(extra))
	seen := make(map[string]bool, len(got))
	for _, n := range got {
		require.False(t, seen[n], "duplicate capability %q", n)
		seen[n] = true
	}
	for _, n := range cat {
		require.True(t, seen[n], "missing catalog capability %q", n)
	}
	for n := range extra {
		require.True(t, seen[n], "missing non-catalog entry %q", n)
	}
}

// TestFullAssemblyModulesAreWired 每个条目都必须给出 Wire。
func TestFullAssemblyModulesAreWired(t *testing.T) {
	for _, c := range Assembly().Capabilities {
		require.NotNil(t, c.Wire, "capability %q has no Wire", c.Descriptor.Name)
	}
}

// TestFullDriverSelection 钉住 full 形态的驱动选中集（T1 按现状声明：三种驱动能力
// 均为全量选中；后续收敛选中集（Task 2–5）时本用例必须同步修改）。
func TestFullDriverSelection(t *testing.T) {
	assert.Equal(t, map[string][]string{
		"storage": {"local", "s3"},
		"queue":   {"redis", "kafka", "rabbitmq"},
		"dataops": {"csv", "excel"},
	}, driverSelection(Assembly()))
}

// TestFullCompiledStorageDrivers 钉住进程内注册表（与 enterprise 侧
// TestEnterpriseCompiledStorageDrivers 同款）：`drivers.go` 的两个 blank import 一旦被删，
// `go build ./...` 与 TestFullDriverSelection 仍全绿（后者只钉 Capability.Drivers 声明），
// 失败只会在运行期的 wire.go fail-closed 文案里暴露 —— 而 full 正是出货形态
// （cmd/server/main.go），故此处直接断言本构建实际注册的 storage 类型。
// 注意 RegisteredTypes() 返回的是**配置取值**集合：s3 驱动包一个包承载 s3/minio/oss 三种
// S3 兼容类型，故为 4 项；驱动**包**集合仍是 assembly 声明的 {local, s3}。
func TestFullCompiledStorageDrivers(t *testing.T) {
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

// TestFullAssemblyHasNoDuplicates 清单内不得重名。
func TestFullAssemblyHasNoDuplicates(t *testing.T) {
	seen := map[string]bool{}
	for _, c := range Assembly().Capabilities {
		require.False(t, seen[c.Descriptor.Name], "duplicate capability %q", c.Descriptor.Name)
		seen[c.Descriptor.Name] = true
	}
}

// TestFullAssemblyUngatedFlagsMatchesCatalogMembership Ungated 精确标记非 catalog 条目：
// 非 catalog 条目必须 Ungated（由形态清单决定，不受 capabilities.enabled 门控），
// catalog 条目必须受门控。
func TestFullAssemblyUngatedFlagsMatchesCatalogMembership(t *testing.T) {
	cat := map[string]bool{}
	for _, n := range catalog.Names() {
		cat[n] = true
	}
	for _, c := range Assembly().Capabilities {
		if cat[c.Descriptor.Name] {
			require.False(t, c.Ungated, "catalog 能力 %q 不得标记 Ungated", c.Descriptor.Name)
			continue
		}
		require.True(t, c.Ungated, "非 catalog 条目 %q 必须标记 Ungated", c.Descriptor.Name)
	}
}

// TestFullAssemblyResolvesToCatalogDefault 默认配置（capabilities.enabled 为空）下，
// catalog 能力的解析集必须与 catalog.Resolve(nil) 逐名一致 —— full 零退化的第一道护栏。
func TestFullAssemblyResolvesToCatalogDefault(t *testing.T) {
	a := Assembly()
	known := map[string]bool{}
	for _, n := range catalog.Names() {
		known[n] = true
	}
	descriptors := make([]contract.Descriptor, 0, len(a.Capabilities))
	for _, c := range a.Capabilities {
		if known[c.Descriptor.Name] {
			descriptors = append(descriptors, c.Descriptor)
		}
	}

	got, err := capability.Resolve(descriptors, nil)
	require.NoError(t, err)

	fromCatalog, err := catalog.Resolve(nil)
	require.NoError(t, err)
	require.ElementsMatch(t, descriptorNames(fromCatalog), descriptorNames(got))
}

// TestFullAssemblyDeclarationsAreWellFormed 声明必须通过全量清单同款校验
// （软依赖不得是错别字），否则 profile 化后会带着声明缺陷上线。
func TestFullAssemblyDeclarationsAreWellFormed(t *testing.T) {
	a := Assembly()
	descriptors := make([]contract.Descriptor, 0, len(a.Capabilities))
	for _, c := range a.Capabilities {
		descriptors = append(descriptors, c.Descriptor)
	}
	require.NoError(t, capability.ValidateDeclarations(descriptors, catalog.Names()))
}

// TestFullAssemblyPortFlow 装配顺序护栏：full 清单里每个被 Wire 读取的端口都必须由内核
// 桥接端口或排在其前的能力提供。删掉 access.Wire 的 Provide("access", …) 或把 access
// 排到 user 之后都会让本用例失败。
func TestFullAssemblyPortFlow(t *testing.T) {
	require.NoError(t, assembly.ValidatePortFlow(Assembly()))
}

// TestFullAssemblyUngatedUnderEnabledSubset 回归（Task 3 评审 Finding 1；P2.4 裁定 7 /
// 设计裁定 B）：capabilities.enabled 为非空子集时，非 catalog 条目不得被裁剪 —— 它们由
// 形态清单决定，不由配置开关门控。encryption（其 Wire 注册字段级加密 hooks 并暴露 Cipher
// 端口）与 storage（uploadsec 消费的存储端口）是两个具体断言：二者仍在装配集内，端口仍
// 被提供；受门控条目仍按启用集闭包裁剪。
func TestFullAssemblyUngatedUnderEnabledSubset(t *testing.T) {
	res, err := assembly.ProbeAssembly(Assembly(), []string{"user", "auth", "uploadsec"})
	require.NoError(t, err)

	for _, name := range nonCatalogEntries {
		require.Contains(t, res.Capabilities, name, "非 catalog 条目 %q 不得被 capabilities.enabled 裁剪", name)
	}
	// 受门控条目照常裁剪：captcha 不在 ["user","auth","uploadsec"] 的硬依赖闭包内。
	require.NotContains(t, res.Capabilities, "captcha")
	require.Contains(t, res.Capabilities, "uploadsec", "storage 的消费方 uploadsec 应随启用集装配")

	// encryption.Wire 运行 → Cipher 端口被提供（email/phone 字段级加密 hooks 随之注册）。
	require.Contains(t, res.Provided["encryption"], "encryption")
	// storage.Wire 运行 → Storage 端口被提供（uploadsec 取回后不再降级为 nil）。
	require.Contains(t, res.Provided["storage"], "storage")
}

// TestFullAssemblyEventBusDoesNotConstructQueue 回归（Task 3 评审 Finding 2）：
// shipped 配置 outbox.publisher=event_bus 下不得在启动期构造队列客户端。base 只在 outbox
// 的 MQ 分支 queue.New（kafka/rabbitmq 构造会连 broker、缺 broker/topic 即启动失败）。
// 因此 queue 能力（Module/作业端点）仍照常装配，但 queue.Wire 不提供任何端口 ——
// 没有队列客户端可被构造或泄漏。
func TestFullAssemblyEventBusDoesNotConstructQueue(t *testing.T) {
	res, err := assembly.ProbeAssembly(Assembly(), nil)
	require.NoError(t, err)

	require.Contains(t, res.Capabilities, "queue", "queue 能力必须照常装配")
	require.NotContains(t, res.Provided, "queue", "event_bus 下不得构造队列客户端")
	// 对照：outbox 仍装配并提供端口（事件总线发布器路径）。
	require.Contains(t, res.Provided["outbox"], "outbox")
}

func capabilityNames(a assembly.Assembly) []string {
	out := make([]string, 0, len(a.Capabilities))
	for _, c := range a.Capabilities {
		out = append(out, c.Descriptor.Name)
	}
	return out
}

func descriptorNames(ds []contract.Descriptor) []string {
	out := make([]string, 0, len(ds))
	for _, d := range ds {
		out = append(out, d.Name)
	}
	return out
}
