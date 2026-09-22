package assembly

import (
	"fmt"
	"os"
	"strings"

	"jimu/internal/app"
	"jimu/internal/config"
	"jimu/internal/contract"
	"jimu/internal/kernel/event"
	"jimu/internal/kernel/httpclient"
	"jimu/internal/kernel/logger"
)

// ProbeResult 是一次装配试运行的观测结果（零值内核件，不连库/Redis、不启动监听）。
type ProbeResult struct {
	// Capabilities 解析出的能力名，按装配顺序（含硬依赖闭包与非 catalog 条目）。
	Capabilities []string
	// Provided 各能力经 Context.Provide 注册的端口名，按注册顺序。
	Provided map[string][]string
	// Modules 按装配顺序登记的非空 Module 实例（Wire 返回 nil、只提供端口或仅参与迁移的
	// 条目不出现）。供「编译面」报告工具在不启动服务的前提下统计该形态的路由数。
	Modules []contract.Module
}

// ValidatePortFlow 校验装配清单的端口流向：按清单顺序试运行每个能力的 Wire，任何经
// Context.Port 读取的端口都必须在读取发生前已提供（更早的能力经 Provide 注册的端口）。
//
// 这是对 Wire 的装配顺序护栏：Wire 无法静态内省端口流向，因此在「零值内核件」上真实
// 执行各 Wire 的 Provide/Port 调用并观察读取结果。试运行只提供无 I/O 的内核件
// （Config/Logger/EventBus/HTTPClient），不连库、不连 Redis、不启动监听；为避免
// 本地存储等按相对路径建目录的构造污染仓库，试运行在临时工作目录内进行。
// 完整形态（full）里每个被读取的端口都应有人提供，缺失即报错；因此它不适用于
// 「软依赖确实缺席」的裁剪形态 —— 那类形态用 ProbeAssembly 观察装配产物。
func ValidatePortFlow(a Assembly) error {
	_, violations, err := probeWires(a, nil)
	if err != nil {
		return err
	}
	if len(violations) > 0 {
		return fmt.Errorf("assembly %q: port flow violations: %s", a.Name, strings.Join(violations, "; "))
	}
	return nil
}

// ProbeAssembly 在零值内核件上按 enabled 解析形态并试运行全部 Wire，返回解析出的能力名
// 与各能力实际提供的端口。解析走与 Run 相同的路径：受门控（catalog）条目按
// capabilities.enabled 裁剪并补齐硬依赖闭包，Ungated 条目恒装配。它不做读取顺序校验
// （那是 ValidatePortFlow 的职责），供启用子集/延迟构造的回归用例观察装配产物。
func ProbeAssembly(a Assembly, enabled []string) (ProbeResult, error) {
	result, _, err := probeWires(a, enabled)
	return result, err
}

// probeWires 是 ValidatePortFlow 与 ProbeAssembly 的共同实现：解析装配集 → 加载能力
// 配置段 → 在临时工作目录内的零值内核件上试运行 Wire，返回观测结果与端口读取违规。
func probeWires(a Assembly, enabled []string) (ProbeResult, []string, error) {
	if err := validateAssembly(a); err != nil {
		return ProbeResult{}, nil, err
	}

	cfg, sections, err := config.LoadWithSections()
	if err != nil {
		return ProbeResult{}, nil, fmt.Errorf("load config: %w", err)
	}
	byName := make(map[string]Capability, len(a.Capabilities))
	for _, c := range a.Capabilities {
		byName[c.Descriptor.Name] = c
	}
	caps, err := resolveCapabilities(a, enabled)
	if err != nil {
		return ProbeResult{}, nil, fmt.Errorf("resolve capabilities: %w", err)
	}
	capCfgs, err := app.LoadCapabilityConfigs(sections, caps, cfg.Environment)
	if err != nil {
		return ProbeResult{}, nil, fmt.Errorf("load capability configs: %w", err)
	}

	// 配置已加载，试运行不再依赖 cwd；临时目录吸收构造期的相对路径副作用。
	tmpDir, err := os.MkdirTemp("", "jimu-portflow-")
	if err != nil {
		return ProbeResult{}, nil, fmt.Errorf("create port flow probe dir: %w", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()
	workDir, err := os.Getwd()
	if err != nil {
		return ProbeResult{}, nil, fmt.Errorf("read working directory: %w", err)
	}
	if err := os.Chdir(tmpDir); err != nil {
		return ProbeResult{}, nil, fmt.Errorf("enter port flow probe dir: %w", err)
	}
	defer func() { _ = os.Chdir(workDir) }()

	container := &app.Container{
		Config:            cfg,
		CapabilityConfigs: capCfgs,
		Logger:            logger.New(cfg.Log),
		EventBus:          event.New(),
		HTTPClient:        httpclient.New(httpclient.Config{}),
	}
	ctx := newContext(container, sections, capCfgs)

	result := ProbeResult{
		Capabilities: make([]string, 0, len(caps)),
		Provided:     make(map[string][]string, len(caps)),
	}
	for _, d := range caps {
		result.Capabilities = append(result.Capabilities, d.Name)
	}

	var violations []string
	current := ""
	ctx.onPort = func(name string, provided bool) {
		if !provided {
			violations = append(violations,
				fmt.Sprintf("capability %q reads port %q before it is provided", current, name))
		}
	}
	ctx.onProvide = func(name string) {
		result.Provided[current] = append(result.Provided[current], name)
	}
	for _, d := range caps {
		current = d.Name
		if err := wireOne(ctx, d, byName); err != nil {
			return ProbeResult{}, nil, fmt.Errorf("assembly %q: %w", a.Name, err)
		}
	}
	result.Modules = ctx.modules
	return result, violations, nil
}
