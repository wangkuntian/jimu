package assembly_test

import (
	"path/filepath"
	"sort"
	"testing"

	"jimu/internal/app"
	"jimu/internal/assembly"
	"jimu/internal/config"
	"jimu/internal/contract"
	"jimu/internal/kernel/event"
	"jimu/internal/kernel/httpclient"
	"jimu/internal/kernel/logger"
	"jimu/internal/profiles/full"

	"github.com/stretchr/testify/require"
)

// TestWireForMatchesProbeAssembly 钉住测试缝与端口流向试运行的同构性：同一形态下，
// WireFor（Run 与 e2e 共用的接线）装配出的模块集合必须与 ProbeAssembly（composereport/
// 门禁使用的零值内核件试运行路径）一致 —— 两者若分歧，e2e 与报告度量的就不是同一个东西。
func TestWireForMatchesProbeAssembly(t *testing.T) {
	t.Chdir(filepath.Join("..", "..")) // configs/ 相对仓库根

	a := full.Assembly()
	caps, err := assembly.Resolve(a, nil)
	require.NoError(t, err)

	cfg, sections, err := config.LoadWithSections()
	require.NoError(t, err)

	// 先取 ProbeAssembly 的结果（它内部自行加载配置，要求 cwd = 仓库根），再切到临时目录：
	// 接线本身不再依赖 cwd，而临时目录能吸收构造期副作用 —— storage/local 会 MkdirAll("storage")，
	// 留在仓库根会污染工作区（读法同 portflow.Wire）。
	probe, err := assembly.ProbeAssembly(a, nil)
	require.NoError(t, err)

	t.Chdir(t.TempDir())
	capCfgs, err := app.LoadCapabilityConfigs(sections, caps, cfg.Environment)
	require.NoError(t, err)

	container := &app.Container{
		Config:            cfg,
		Sections:          sections,
		CapabilityConfigs: capCfgs,
		Logger:            logger.New(cfg.Log),
		EventBus:          event.New(),
		HTTPClient:        httpclient.New(httpclient.Config{}),
	}
	ctx, modules, err := assembly.WireFor(container, sections, capCfgs, a, caps)
	require.NoError(t, err)
	require.NotNil(t, ctx)

	names := func(ms []contract.Module) []string {
		out := make([]string, 0, len(ms))
		for _, m := range ms {
			out = append(out, contract.Describe(m).Name)
		}
		sort.Strings(out)
		return out
	}
	require.Equal(t, names(probe.Modules), names(modules),
		"WireFor 装配的模块集合必须与 ProbeAssembly 一致")
	require.NotEmpty(t, modules)
}

// 形态名写错时 Resolve 必须报错（fail-closed），不能静默装配出一个空形态。
func TestResolveRejectsUnknownCapability(t *testing.T) {
	_, err := assembly.Resolve(full.Assembly(), []string{"ghost"})
	require.ErrorContains(t, err, "ghost")
}
