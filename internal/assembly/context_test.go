package assembly

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"jimu/internal/app"
	"jimu/internal/contract"
)

// fakeModule 最小 contract.Module。
type fakeModule struct{ name string }

func (m fakeModule) Name() string                      { return m.name }
func (m fakeModule) RegisterHTTP(contract.Router)      {}
func (m fakeModule) RegisterJobs(contract.JobRegistry) {}
func (m fakeModule) RegisterEvents(contract.EventBus)  {}

// newTestContext 构造一个只带内核访问器的装配上下文（不触库/Redis）。
func newTestContext(t *testing.T) *Context {
	t.Helper()
	return newContext(&app.Container{}, nil, nil)
}

// newTestContextWithConfigs 构造带已解码能力配置段的上下文。
func newTestContextWithConfigs(t *testing.T, capCfgs *app.CapabilityConfigs) *Context {
	t.Helper()
	return newContext(&app.Container{}, nil, capCfgs)
}

func TestRegisterRejectsDuplicateModuleName(t *testing.T) {
	ctx := newTestContext(t)
	require.NoError(t, ctx.Register(fakeModule{name: "user"}))
	require.Error(t, ctx.Register(fakeModule{name: "user"}))
}

// TestRegisterKeepsDeclarationOrder 已注册模块按注册顺序排列，供 Bootstrap 使用。
func TestRegisterKeepsDeclarationOrder(t *testing.T) {
	ctx := newTestContext(t)
	require.NoError(t, ctx.Register(fakeModule{name: "user"}))
	require.NoError(t, ctx.Register(fakeModule{name: "auth"}))
	require.Equal(t, []string{"user", "auth"}, moduleNames(ctx))
}

// fakeSectionConfig 最小 SectionConfig，用于 MustSection 用例。
type fakeSectionConfig struct {
	value string
}

func (c *fakeSectionConfig) ApplyDefaults()  {}
func (c *fakeSectionConfig) Validate() error { return nil }

// fakeDecoder 按段键填充 *fakeSectionConfig。
type fakeDecoder struct{ values map[string]string }

func (d fakeDecoder) UnmarshalKey(key string, rawVal any) error {
	if c, ok := rawVal.(*fakeSectionConfig); ok {
		c.value = d.values[key]
	}
	return nil
}

// TestMustSectionReturnsTypedConfig 便利入口：能力 Wire 取配置段无需手写类型断言；
// 段不存在（能力未启用）时返回 T 的零值。
func TestMustSectionReturnsTypedConfig(t *testing.T) {
	caps := []contract.Descriptor{{
		Name:    "captcha",
		Configs: []contract.ConfigSpec{{Section: "captcha", New: func() any { return &fakeSectionConfig{} }}},
	}}
	capCfgs, err := app.LoadCapabilityConfigs(fakeDecoder{values: map[string]string{"captcha": "from-yaml"}}, caps, "dev")
	require.NoError(t, err)

	ctx := newTestContextWithConfigs(t, capCfgs)
	got := MustSection[*fakeSectionConfig](ctx, "captcha")
	require.NotNil(t, got)
	assert.Equal(t, "from-yaml", got.value)
	assert.Nil(t, MustSection[*fakeSectionConfig](ctx, "absent"))
	assert.Nil(t, newTestContext(t).CapabilityConfigs())
}

// moduleNames 返回上下文已注册模块名（按注册顺序）。
func moduleNames(ctx *Context) []string {
	out := make([]string, 0, len(ctx.modules))
	for _, m := range ctx.modules {
		out = append(out, m.Name())
	}
	return out
}
