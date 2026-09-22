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

// TestValidatePortFlowRejectsReadBeforeProvide 护栏必须观察到真实的 Port 读取：
// 消费方先读到尚无人提供的端口即报错并点名能力与端口；提供者排在前则通过。
// 这正是 full 清单里 wireUser 读 "access" 而 wireAccess 未 Provide 时的失败形态。
func TestValidatePortFlowRejectsReadBeforeProvide(t *testing.T) {
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
			_ = ctx.Port("access")
			return fakeModule{name: "user"}, nil
		},
	}

	err := ValidatePortFlow(Assembly{Name: "test", Capabilities: []Capability{consumer, provider}})
	require.ErrorContains(t, err, `capability "user" reads port "access" before it is provided`)

	require.NoError(t, ValidatePortFlow(Assembly{Name: "test", Capabilities: []Capability{provider, consumer}}))
}

// TestResolveCapabilitiesKeepsUngatedUnderEnabledSubset 非 catalog（Ungated）条目不受
// capabilities.enabled 门控（P2.4 裁定 7 / 设计裁定 B）：配置子集只裁剪受门控条目并补齐
// 其硬依赖闭包，Ungated 条目恒在结果内；结果按形态清单顺序排列。
func TestResolveCapabilitiesKeepsUngatedUnderEnabledSubset(t *testing.T) {
	a := Assembly{Name: "test", Capabilities: []Capability{
		{Descriptor: contract.Descriptor{Name: "encryption"}, Wire: nilWire, Ungated: true},
		{Descriptor: contract.Descriptor{Name: "storage"}, Wire: nilWire, Ungated: true},
		{Descriptor: contract.Descriptor{Name: "access"}, Wire: nilWire},
		{Descriptor: contract.Descriptor{Name: "user", Requires: []string{"access"}}, Wire: nilWire},
		{Descriptor: contract.Descriptor{Name: "apikey"}, Wire: nilWire},
	}}

	got, err := resolveCapabilities(a, []string{"user"})
	require.NoError(t, err)
	// access 由 user 的硬依赖闭包补入；apikey 被裁剪；顺序 = 形态清单顺序。
	require.Equal(t, []string{"encryption", "storage", "access", "user"}, descriptorNames(got))
}

// TestResolveCapabilitiesEmptyEnabledEnablesAllGated capabilities.enabled 为空表示全部
// 受门控条目启用（向后兼容），Ungated 条目照常在列。
func TestResolveCapabilitiesEmptyEnabledEnablesAllGated(t *testing.T) {
	a := Assembly{Name: "test", Capabilities: []Capability{
		{Descriptor: contract.Descriptor{Name: "storage"}, Wire: nilWire, Ungated: true},
		{Descriptor: contract.Descriptor{Name: "user"}, Wire: nilWire},
		{Descriptor: contract.Descriptor{Name: "apikey"}, Wire: nilWire},
	}}

	got, err := resolveCapabilities(a, nil)
	require.NoError(t, err)
	require.Equal(t, []string{"storage", "user", "apikey"}, descriptorNames(got))
}

// TestResolveCapabilitiesKeepsResolveErrors 未知名与缺失硬依赖的错误语义不被 Ungated
// 并集削弱。
func TestResolveCapabilitiesKeepsResolveErrors(t *testing.T) {
	a := Assembly{Name: "test", Capabilities: []Capability{
		{Descriptor: contract.Descriptor{Name: "storage"}, Wire: nilWire, Ungated: true},
		{Descriptor: contract.Descriptor{Name: "user", Requires: []string{"ghost"}}, Wire: nilWire},
		{Descriptor: contract.Descriptor{Name: "auth", Requires: []string{"user"}}, Wire: nilWire},
	}}

	_, err := resolveCapabilities(a, []string{"nope"})
	require.ErrorContains(t, err, `unknown capability "nope"`)
	_, err = resolveCapabilities(a, []string{"auth"})
	require.ErrorContains(t, err, `capability "user" requires unknown capability "ghost"`)
}

// TestUngatedCapabilitiesWireUnderEnabledSubset Ungated 条目在非空启用子集下仍然装配：
// 其 Wire 真被执行并注册端口，排在后面的受门控能力能消费到它们；被裁剪的受门控条目
// 不得装配。
func TestUngatedCapabilitiesWireUnderEnabledSubset(t *testing.T) {
	a := Assembly{Name: "test", Capabilities: []Capability{
		{
			Descriptor: contract.Descriptor{Name: "encryption"},
			Ungated:    true,
			Wire: func(ctx *Context) (contract.Module, error) {
				return nil, ctx.Provide("encryption", "cipher")
			},
		},
		{
			Descriptor: contract.Descriptor{Name: "storage"},
			Ungated:    true,
			Wire: func(ctx *Context) (contract.Module, error) {
				return nil, ctx.Provide("storage", "store")
			},
		},
		{
			Descriptor: contract.Descriptor{Name: "user"},
			Wire: func(ctx *Context) (contract.Module, error) {
				require.NotNil(t, ctx.Port("encryption"), "Ungated 端口必须在启用子集下仍被提供")
				require.NotNil(t, ctx.Port("storage"), "Ungated 端口必须在启用子集下仍被提供")
				return fakeModule{name: "user"}, nil
			},
		},
		{
			Descriptor: contract.Descriptor{Name: "apikey"},
			Wire: func(*Context) (contract.Module, error) {
				t.Error("被 capabilities.enabled 裁剪的受门控条目不得装配")
				return nil, nil
			},
		},
	}}
	caps, err := resolveCapabilities(a, []string{"user"})
	require.NoError(t, err)
	byName := make(map[string]Capability, len(a.Capabilities))
	for _, c := range a.Capabilities {
		byName[c.Descriptor.Name] = c
	}

	ctx := newTestContext(t)
	require.NoError(t, wireCapabilities(ctx, caps, byName))
	require.Equal(t, []string{"user"}, moduleNames(ctx))
}

// descriptorNames 返回描述符名（按顺序）。
func descriptorNames(ds []contract.Descriptor) []string {
	out := make([]string, 0, len(ds))
	for _, d := range ds {
		out = append(out, d.Name)
	}
	return out
}

// TestWireCapabilitiesRejectsResolvedCapabilityOutsideAssembly 解析结果必须都来自清单。
func TestWireCapabilitiesRejectsResolvedCapabilityOutsideAssembly(t *testing.T) {
	ctx := newTestContext(t)
	err := wireCapabilities(ctx, []contract.Descriptor{{Name: "ghost"}}, nil)
	require.ErrorContains(t, err, "not declared in the assembly")
}
