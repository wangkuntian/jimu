package profiles_test

import (
	"sort"
	"testing"

	"jimu/internal/assembly"
	"jimu/internal/capabilities/catalog"
	"jimu/internal/contract"
	"jimu/internal/profiles/enterprise"
	"jimu/internal/profiles/full"
	"jimu/internal/profiles/machine"
	"jimu/internal/profiles/minimal"
	"jimu/internal/profiles/saas"

	"github.com/stretchr/testify/require"
)

// profileAssemblies 是设计 §2 的 5 个形态入口。
func profileAssemblies() []struct {
	name string
	a    assembly.Assembly
} {
	return []struct {
		name string
		a    assembly.Assembly
	}{
		{"full", full.Assembly()},
		{"minimal", minimal.Assembly()},
		{"saas", saas.Assembly()},
		{"enterprise", enterprise.Assembly()},
		{"machine", machine.Assembly()},
	}
}

// TestProfilePortFlow 每个形态的端口流向护栏（Task 3 评审：护栏此前只覆盖 full）。
//
// ValidatePortFlow 把「读取时尚未提供」一律视为违规，这对裁剪形态会误报：形态刻意排除
// 的能力所提供的端口（如 minimal 的 user 读 tenant/outbox）本就缺席，读取它们正是设计内的
// 软依赖降级。因此这里先用 ProbeAssembly 观测 full 提供的端口全集，把形态缺席的端口以
// **空值**前置注册（桩只登记端口名，消费方取回的值仍是 nil，降级行为不变），使护栏只对
// 形态**内部**真实的端口边报错——把 access 排到 user 之后、或把 grpc 排到 user 之前都会
// 让本用例失败。
func TestProfilePortFlow(t *testing.T) {
	universe := providedPorts(t, full.Assembly())
	for _, p := range profileAssemblies() {
		t.Run(p.name, func(t *testing.T) {
			require.NoError(t, assembly.ValidatePortFlow(withAbsentPortStubs(t, p.a, universe)))
		})
	}
}

// TestProfileNonCatalogEntriesAreUngated 每个形态里非 catalog 条目都必须标记 Ungated
// （Task 3 评审 Finding 1 的复发风险：形态若把 encryption/storage 当受门控条目，
// capabilities.enabled 的非空子集会把它们静默裁剪，字段加密/存储端口随之消失）；
// catalog 条目则必须受门控。
func TestProfileNonCatalogEntriesAreUngated(t *testing.T) {
	inCatalog := map[string]bool{}
	for _, n := range catalog.Names() {
		inCatalog[n] = true
	}
	for _, p := range profileAssemblies() {
		t.Run(p.name, func(t *testing.T) {
			for _, c := range p.a.Capabilities {
				if inCatalog[c.Descriptor.Name] {
					require.False(t, c.Ungated, "catalog 能力 %q 不得标记 Ungated", c.Descriptor.Name)
					continue
				}
				require.True(t, c.Ungated,
					"非 catalog 条目 %q 必须标记 Ungated，否则会被 capabilities.enabled 静默裁剪", c.Descriptor.Name)
			}
		})
	}
}

// withAbsentPortStubs 前置一个只登记「本形态缺席端口」的桩能力，见 TestProfilePortFlow。
func withAbsentPortStubs(t *testing.T, a assembly.Assembly, universe map[string]bool) assembly.Assembly {
	t.Helper()

	present := map[string]bool{}
	for name := range providedPorts(t, a) {
		present[name] = true
	}
	absent := make([]string, 0, len(universe))
	for name := range universe {
		if !present[name] {
			absent = append(absent, name)
		}
	}
	if len(absent) == 0 {
		return a
	}
	// 端口名排序：试运行结果与 map 迭代顺序无关。
	sort.Strings(absent)

	stub := assembly.Capability{
		Descriptor: contract.Descriptor{Name: "absent-ports-probe"},
		Wire: func(ctx *assembly.Context) (contract.Module, error) {
			for _, name := range absent {
				if err := ctx.Provide(name, nil); err != nil {
					return nil, err
				}
			}
			return nil, nil
		},
	}
	stubbed := a
	stubbed.Capabilities = append([]assembly.Capability{stub}, a.Capabilities...)
	return stubbed
}

// providedPorts 返回形态解析集里被任一能力 Provide 的端口名集合。
func providedPorts(t *testing.T, a assembly.Assembly) map[string]bool {
	t.Helper()

	res, err := assembly.ProbeAssembly(a, nil)
	require.NoError(t, err)

	out := map[string]bool{}
	for _, ports := range res.Provided {
		for _, name := range ports {
			out[name] = true
		}
	}
	return out
}
