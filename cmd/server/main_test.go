package main

import (
	"slices"
	"testing"

	"jimu/internal/assembly"
	"jimu/internal/capabilities/catalog"
	"jimu/internal/capability"
	"jimu/internal/contract"

	"github.com/stretchr/testify/require"
)

// nonCatalogCapabilities 是 full 形态中的非 catalog 条目（P2.4 裁定 7）：它们是
// assembly.Capability 但不是 catalog 成员（catalog 仍 18 项，不入迁移/权限聚合）。
var nonCatalogCapabilities = []string{"encryption", "storage", "notification", "retention", "ws", "grpc", "apidocs"}

// TestFullAssemblyCoversCatalog 过渡形态的能力清单 = catalog 全量 ∪ 固定非 catalog 条目，
// 且每一项都必须给出 Wire（无 Module 实例的能力给出空 Wire）。
func TestFullAssemblyCoversCatalog(t *testing.T) {
	a := fullAssembly()
	require.Equal(t, "full", a.Name)
	require.Equal(t, version, a.Version, "构建版本必须传给驱动（console 状态页依赖它）")

	got := make([]string, 0, len(a.Capabilities))
	for _, c := range a.Capabilities {
		require.NotNil(t, c.Wire, "capability %q has no Wire", c.Descriptor.Name)
		got = append(got, c.Descriptor.Name)
	}
	want := append(catalog.Names(), nonCatalogCapabilities...)
	slices.Sort(got)
	slices.Sort(want)
	require.Equal(t, want, got, "过渡形态的能力清单必须 = catalog 全量 ∪ 非 catalog 条目")
}

// TestFullAssemblyResolvesToCatalogDefault 默认配置（capabilities.enabled 为空）下，
// 过渡形态的 catalog 能力解析集必须与 catalog.Resolve(nil) 逐名一致 —— full 零退化的第一道护栏。
func TestFullAssemblyResolvesToCatalogDefault(t *testing.T) {
	a := fullAssembly()
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

// TestFullAssemblyDeclarationsAreWellFormed 过渡形态的声明必须通过全量清单同款校验
// （软依赖不得是错别字），否则 profile 化后会带着声明缺陷上线。
func TestFullAssemblyDeclarationsAreWellFormed(t *testing.T) {
	a := fullAssembly()
	descriptors := make([]contract.Descriptor, 0, len(a.Capabilities))
	for _, c := range a.Capabilities {
		descriptors = append(descriptors, c.Descriptor)
	}
	require.NoError(t, capability.ValidateDeclarations(descriptors, catalog.Names()))
}

// TestFullAssemblyOrder 钉住过渡形态的装配顺序：提供端口的能力必须排在消费它的能力
// 之前（encryption/storage/notification/queue/outbox/breach 先于 tenant/user/auth/
// uploadsec/grpc；tenant/access 先于 user；captcha/mfa 先于 auth）。Task 4 的 profile
// 清单会接手这份顺序。
func TestFullAssemblyOrder(t *testing.T) {
	want := []string{
		"encryption", "storage", "notification", "queue", "outbox", "breach",
		"tenant", "access", "user", "captcha", "mfa", "auth", "passkey", "audit",
		"console", "oauth", "apikey", "dataops", "feature", "uploadsec", "search",
		"retention", "apidocs", "grpc", "ws",
	}
	got := make([]string, 0, len(want))
	for _, c := range fullAssembly().Capabilities {
		got = append(got, c.Descriptor.Name)
	}
	require.Equal(t, want, got)
}

// TestFullAssemblyPortFlow 装配顺序护栏：full 清单里每个被 Wire 读取的端口都必须由内核
// 桥接端口或排在其前的能力提供。过渡期的内联闭包无法静态内省，故该用例真实试运行各
// Wire 并观察 Provide/Port 调用（零值内核件、只读配置段，不连库）。删掉 wireAccess 的
// Provide("access", …) 或把 access 排到 user 之后都会让本用例失败。
func TestFullAssemblyPortFlow(t *testing.T) {
	require.NoError(t, assembly.ValidatePortFlow(fullAssembly()))
}

// TestFullAssemblyHasNoDuplicates 清单内不得重名。
func TestFullAssemblyHasNoDuplicates(t *testing.T) {
	seen := map[string]bool{}
	for _, c := range fullAssembly().Capabilities {
		require.False(t, seen[c.Descriptor.Name], "duplicate capability %q", c.Descriptor.Name)
		seen[c.Descriptor.Name] = true
	}
}

// TestFullAssemblyUngatedUnderEnabledSubset 回归（Task 3 评审 Finding 1；P2.4 裁定 7 /
// 设计裁定 B）：capabilities.enabled 为非空子集时，非 catalog 条目不得被裁剪 —— 它们由
// 形态清单决定，不由配置开关门控。encryption（其 Wire 注册字段级加密 hooks 并暴露 Cipher
// 端口）与 storage（uploadsec 消费的存储端口）是两个具体断言：二者仍在装配集内，端口仍
// 被提供；受门控条目仍按启用集闭包裁剪。
func TestFullAssemblyUngatedUnderEnabledSubset(t *testing.T) {
	res, err := assembly.ProbeAssembly(fullAssembly(), []string{"user", "auth", "uploadsec"})
	require.NoError(t, err)

	for _, name := range nonCatalogCapabilities {
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
	res, err := assembly.ProbeAssembly(fullAssembly(), nil)
	require.NoError(t, err)

	require.Contains(t, res.Capabilities, "queue", "queue 能力必须照常装配")
	require.NotContains(t, res.Provided, "queue", "event_bus 下不得构造队列客户端")
	// 对照：outbox 仍装配并提供端口（事件总线发布器路径）。
	require.Contains(t, res.Provided["outbox"], "outbox")
}

func descriptorNames(ds []contract.Descriptor) []string {
	out := make([]string, 0, len(ds))
	for _, d := range ds {
		out = append(out, d.Name)
	}
	return out
}
