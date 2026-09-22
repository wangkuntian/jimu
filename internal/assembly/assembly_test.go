package assembly

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"jimu/internal/contract"
)

// nilWire 是「无 Module 实例」能力的 Wire：只提供端口 / 仅参与迁移。
func nilWire(*Context) (contract.Module, error) { return nil, nil }

// TestRunRejectsInvalidAssemblyBeforeLoadingConfig 装配自检发生在加载配置/连库之前，
// 因此这些用例无需 configs/ 与 DB 即可覆盖。
func TestRunRejectsInvalidAssemblyBeforeLoadingConfig(t *testing.T) {
	cases := []struct {
		name string
		a    Assembly
	}{
		{"missing name", Assembly{}},
		{"empty capability name", Assembly{Name: "full", Capabilities: []Capability{{Wire: nilWire}}}},
		{"missing wire", Assembly{Name: "full", Capabilities: []Capability{{Descriptor: contract.Descriptor{Name: "user"}}}}},
		{"duplicate capability", Assembly{Name: "full", Capabilities: []Capability{
			{Descriptor: contract.Descriptor{Name: "user"}, Wire: nilWire},
			{Descriptor: contract.Descriptor{Name: "user"}, Wire: nilWire},
		}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			require.Error(t, Run(tc.a))
		})
	}
}

// TestWireCapabilitiesOrdersProvidersBeforeConsumers 端口按清单顺序可见：
// 排在后面的能力能取到前面能力注册的端口。
func TestWireCapabilitiesOrdersProvidersBeforeConsumers(t *testing.T) {
	ctx := newTestContext(t)
	provider := Capability{
		Descriptor: contract.Descriptor{Name: "access"},
		Wire: func(ctx *Context) (contract.Module, error) {
			require.NoError(t, ctx.Provide("access", "role-assigner"))
			return fakeModule{name: "access"}, nil
		},
	}
	consumer := Capability{
		Descriptor: contract.Descriptor{Name: "user"},
		Wire: func(ctx *Context) (contract.Module, error) {
			roles, _ := ctx.Port("access").(string)
			require.Equal(t, "role-assigner", roles)
			return fakeModule{name: "user"}, nil
		},
	}
	caps := []contract.Descriptor{provider.Descriptor, consumer.Descriptor}
	byName := map[string]Capability{"access": provider, "user": consumer}

	require.NoError(t, wireCapabilities(ctx, caps, byName))
	require.Equal(t, []string{"access", "user"}, moduleNames(ctx))
}

// TestWireCapabilitiesMissingPortStaysAbsent 端口缺失只降级：消费方取回 nil，
// Wire 返回 nil 的能力不注册模块。
func TestWireCapabilitiesMissingPortStaysAbsent(t *testing.T) {
	ctx := newTestContext(t)
	consumer := Capability{
		Descriptor: contract.Descriptor{Name: "auth"},
		Wire: func(ctx *Context) (contract.Module, error) {
			require.Nil(t, ctx.Port("captcha"), "未注册的端口必须返回 nil")
			return nil, nil
		},
	}
	require.NoError(t, wireCapabilities(ctx, []contract.Descriptor{consumer.Descriptor},
		map[string]Capability{"auth": consumer}))
	require.Empty(t, moduleNames(ctx))
}

// TestWireCapabilitiesPropagatesWireError Wire 失败必须带上能力名上抛。
func TestWireCapabilitiesPropagatesWireError(t *testing.T) {
	ctx := newTestContext(t)
	boom := errors.New("boom")
	c := Capability{
		Descriptor: contract.Descriptor{Name: "auth"},
		Wire:       func(*Context) (contract.Module, error) { return nil, boom },
	}
	err := wireCapabilities(ctx, []contract.Descriptor{c.Descriptor}, map[string]Capability{"auth": c})
	require.ErrorIs(t, err, boom)
	require.ErrorContains(t, err, `wire capability "auth"`)
}

// TestWireCapabilitiesRejectsResolvedCapabilityOutsideAssembly 解析结果必须都来自清单。
func TestWireCapabilitiesRejectsResolvedCapabilityOutsideAssembly(t *testing.T) {
	ctx := newTestContext(t)
	err := wireCapabilities(ctx, []contract.Descriptor{{Name: "ghost"}}, nil)
	require.ErrorContains(t, err, "not declared in the assembly")
}
