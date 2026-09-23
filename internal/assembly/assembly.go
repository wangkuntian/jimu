// Package assembly 是组合根驱动：把「形态（profile）」声明的能力清单装配成运行中的应用。
//
// 依赖方向（防环）：assembly → app → kernel；能力 wire.go → assembly + kernel + contract；
// assembly 本身不 import 任何能力包，能力件经 Context 的端口注册表互相消费。
package assembly

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"jimu/internal/app"
	"jimu/internal/capability"
	"jimu/internal/config"
	"jimu/internal/contract"
)

// Capability 是一个能力在装配期的形态：静态声明（Descriptor）+ 自装配函数（Wire）。
// Wire 构造能力的 Module 实例（无实例时返回 nil, nil）并把它对外提供的端口经
// Context.Provide 注册，供排在后面的能力消费。
type Capability struct {
	Descriptor contract.Descriptor
	Wire       func(*Context) (contract.Module, error)
	// Drivers 本形态为这个能力选中的驱动包名（子集；空 = 不选任何驱动）。
	// 选中集必须 ⊆ Descriptor.Drivers，且与 profiles/<name>/drivers.go 的 blank import
	// 逐值一致（make check-capabilities 静态校验，设计 §3.7 的两道保险之二）。
	Drivers []string
	// Ungated 标记非 catalog 条目：由形态清单决定是否装配，不受 capabilities.enabled
	// 门控（P2.4 裁定 7 / 设计裁定 B：非 catalog 包不由启用集门控）。catalog 条目为
	// false，随 capabilities.enabled 的解析集裁剪与补齐。
	Ungated bool
}

// Assembly 描述一个形态的完整装配。
type Assembly struct {
	// Name 形态名（如 full），用于错误与日志。
	Name string
	// Version 构建版本：config 里没有该字段，由各形态的 main 包经 ldflags 注入后传入；
	// 为空时保持配置加载结果不变。
	Version string
	// Capabilities 顺序即解析与装配顺序：提供端口的能力必须排在消费它的能力之前。
	Capabilities []Capability
	// Seed 在全部能力装配完成、Bootstrap 之前执行；nil 表示不在启动时播种。
	Seed func(*Context) error
}

// Run 驱动一个形态的完整生命周期：
// 加载配置 → 解析启用集 → 加载能力配置段 → 构造内核容器 → 建装配上下文 →
// 按解析顺序 Wire/Register → Seed → Bootstrap → 运行直到收到退出信号。
func Run(a Assembly) error {
	if err := validateAssembly(a); err != nil {
		return err
	}

	cfg, sections, err := config.LoadWithSections()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	if a.Version != "" {
		cfg.Version = a.Version
	}
	cfg.Environment = os.Getenv("APP_ENV")

	byName := make(map[string]Capability, len(a.Capabilities))
	for _, c := range a.Capabilities {
		byName[c.Descriptor.Name] = c
	}
	// 能力开关：capabilities.enabled 为空表示全部启用（向后兼容）；非 catalog（Ungated）
	// 条目不受门控（P2.4 裁定 7）。在构建容器前解析，因为能力配置段按启用集加载（设计 §8）。
	caps, err := resolveCapabilities(a, cfg.Capabilities.Enabled)
	if err != nil {
		return fmt.Errorf("resolve capabilities: %w", err)
	}
	enabled := make(map[string]bool, len(caps))
	for _, d := range caps {
		enabled[d.Name] = true
	}

	// 能力配置段：按启用集解码 → 默认值 → 校验（prod 下追加加严校验）。
	// 未启用的能力不在 caps 内，其配置段既不出现也不校验（设计 §8）。
	capCfgs, err := app.LoadCapabilityConfigs(sections, caps, cfg.Environment)
	if err != nil {
		return fmt.Errorf("load capability configs: %w", err)
	}

	container, err := app.NewContainer(cfg, sections, capCfgs, caps, enabled)
	if err != nil {
		return fmt.Errorf("create container: %w", err)
	}
	// 容器构造成功后，任何失败都必须回收 DB/Redis/追踪出口。
	stop := func() { _ = container.Stop(context.Background()) }

	// 配置文件热更新：仅应用运行时安全项（log.level）。
	// 结构类配置（DB/Redis 连接池、监听端口等）变更需重启进程生效。
	if err := config.Watch(func(newCfg *config.Config) error {
		container.Logger.Infow("config file changed, applying runtime settings", "level", newCfg.Log.Level)
		if err := container.Logger.SetLevel(newCfg.Log.Level); err != nil {
			container.Logger.Errorw("apply new log level failed", "error", err.Error())
			return err
		}
		return nil
	}); err != nil {
		container.Logger.Warnw("config file watch disabled", "error", err.Error())
	}

	ctx := newContext(container, sections, capCfgs)
	// 解析集写回上下文：Seed 等装配期钩子据本形态的能力（而非全量清单）决策。
	ctx.caps = caps

	if err := wireCapabilities(ctx, caps, byName); err != nil {
		stop()
		return fmt.Errorf("assembly %q: %w", a.Name, err)
	}

	if a.Seed != nil {
		if err := a.Seed(ctx); err != nil {
			stop()
			return fmt.Errorf("seed assembly %q: %w", a.Name, err)
		}
	}

	application, err := app.Bootstrap(container, ctx.Components(), ctx.Jobs(), ctx.modules...)
	if err != nil {
		stop()
		return fmt.Errorf("bootstrap application: %w", err)
	}

	signalCtx, stopSignals := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stopSignals()
	if err := application.Run(signalCtx); err != nil {
		return fmt.Errorf("run application: %w", err)
	}
	return nil
}

// resolveCapabilities 解析一个形态的实际装配集，是 Run 与端口流向试运行的共用入口。
//
// 受门控条目走 capability.Resolve：configured 为空表示全部受门控条目启用；非空时未知名
// 报错、硬依赖自动补齐闭包（错误语义不变）。Ungated 条目（非 catalog）由形态清单决定，
// capabilities.enabled 不得移除，故直接并入结果（P2.4 裁定 7）。最终按形态清单顺序
// 排列，即装配顺序。
func resolveCapabilities(a Assembly, configured []string) ([]contract.Descriptor, error) {
	all := make([]contract.Descriptor, 0, len(a.Capabilities))
	gated := make([]string, 0, len(a.Capabilities))
	selected := make(map[string]contract.Descriptor, len(a.Capabilities))
	for _, c := range a.Capabilities {
		all = append(all, c.Descriptor)
		if c.Ungated {
			selected[c.Descriptor.Name] = c.Descriptor
			continue
		}
		gated = append(gated, c.Descriptor.Name)
	}
	if len(configured) == 0 {
		configured = gated
	}
	resolved, err := capability.Resolve(all, configured)
	if err != nil {
		return nil, err
	}
	for _, d := range resolved {
		selected[d.Name] = d
	}
	out := make([]contract.Descriptor, 0, len(selected))
	for _, c := range a.Capabilities {
		if d, ok := selected[c.Descriptor.Name]; ok {
			out = append(out, d)
		}
	}
	return out, nil
}

// wireCapabilities 按解析顺序调用每个能力的 Wire，并注册非空 Module 实例。
// 顺序即清单顺序：提供端口的能力排在消费方之前，消费方经 Context.Port 取回
// （端口缺失即软依赖降级）。
func wireCapabilities(ctx *Context, caps []contract.Descriptor, byName map[string]Capability) error {
	for _, d := range caps {
		if err := wireOne(ctx, d, byName); err != nil {
			return err
		}
	}
	return nil
}

// wireOne 装配单个能力：Wire 构造实例并注册端口，非空 Module 交给 Bootstrap。
func wireOne(ctx *Context, d contract.Descriptor, byName map[string]Capability) error {
	c, ok := byName[d.Name]
	if !ok {
		return fmt.Errorf("capability %q resolved but not declared in the assembly", d.Name)
	}
	module, err := c.Wire(ctx)
	if err != nil {
		return fmt.Errorf("wire capability %q: %w", d.Name, err)
	}
	if module == nil {
		return nil // 无 Module 实例：只提供端口 / 仅参与迁移
	}
	return ctx.Register(module)
}

// validateAssembly 在触碰配置/DB 之前自检清单：名字必填、Wire 必备、不得重名，
// 且每个能力的驱动选中集必须是其 Descriptor.Drivers 的子集（不得选中未声明的驱动）。
func validateAssembly(a Assembly) error {
	if a.Name == "" {
		return fmt.Errorf("assembly: name is required")
	}
	seen := make(map[string]bool, len(a.Capabilities))
	for _, c := range a.Capabilities {
		name := c.Descriptor.Name
		if name == "" {
			return fmt.Errorf("assembly %q: capability with an empty name", a.Name)
		}
		if c.Wire == nil {
			return fmt.Errorf("assembly %q: capability %q has no Wire", a.Name, name)
		}
		if seen[name] {
			return fmt.Errorf("assembly %q: capability %q declared twice", a.Name, name)
		}
		available := make(map[string]bool, len(c.Descriptor.Drivers))
		for _, d := range c.Descriptor.Drivers {
			available[d] = true
		}
		selected := make(map[string]bool, len(c.Drivers))
		for _, d := range c.Drivers {
			if !available[d] {
				return fmt.Errorf("assembly %q: capability %q selects driver %q which is not declared in its Descriptor.Drivers %v",
					a.Name, name, d, c.Descriptor.Drivers)
			}
			if selected[d] {
				return fmt.Errorf("assembly %q: capability %q selects driver %q twice", a.Name, name, d)
			}
			selected[d] = true
		}
		seen[name] = true
	}
	return nil
}
