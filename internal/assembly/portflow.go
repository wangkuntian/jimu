package assembly

import (
	"fmt"
	"os"
	"strings"

	"jimu/internal/app"
	"jimu/internal/capability"
	"jimu/internal/config"
	"jimu/internal/contract"
	"jimu/internal/kernel/event"
	"jimu/internal/kernel/httpclient"
	"jimu/internal/kernel/logger"
)

// ValidatePortFlow 校验装配清单的端口流向：按清单顺序试运行每个能力的 Wire，任何经
// Context.Port 读取的端口都必须在读取发生前已提供（更早的能力经 Provide 注册的端口）。
//
// 这是对 Wire 的装配顺序护栏：Wire 无法静态内省端口流向，因此在「零值内核件」上真实
// 执行各 Wire 的 Provide/Port 调用并观察读取结果。试运行只提供无 I/O 的内核件
// （Config/Logger/EventBus/HTTPClient），不连库、不连 Redis、不启动监听；为避免
// 本地存储等按相对路径建目录的构造污染仓库，试运行在临时工作目录内进行。
// 完整形态（full）里每个被读取的端口都应有人提供，缺失即报错；因此它不适用于
// 「软依赖确实缺席」的裁剪形态。
func ValidatePortFlow(a Assembly) error {
	if err := validateAssembly(a); err != nil {
		return err
	}

	cfg, sections, err := config.LoadWithSections()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	descriptors := make([]contract.Descriptor, 0, len(a.Capabilities))
	byName := make(map[string]Capability, len(a.Capabilities))
	for _, c := range a.Capabilities {
		descriptors = append(descriptors, c.Descriptor)
		byName[c.Descriptor.Name] = c
	}
	// 校验的是清单声明的装配顺序本身，故不按 capabilities.enabled 裁剪（enabled 为空
	// 时二者一致；非空子集下运行期行为由 Run 与 profile 各自的用例覆盖）。
	caps, err := capability.Resolve(descriptors, nil)
	if err != nil {
		return fmt.Errorf("resolve capabilities: %w", err)
	}
	capCfgs, err := app.LoadCapabilityConfigs(sections, caps, cfg.Environment)
	if err != nil {
		return fmt.Errorf("load capability configs: %w", err)
	}

	// 配置已加载，试运行不再依赖 cwd；临时目录吸收构造期的相对路径副作用。
	tmpDir, err := os.MkdirTemp("", "jimu-portflow-")
	if err != nil {
		return fmt.Errorf("create port flow probe dir: %w", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()
	workDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("read working directory: %w", err)
	}
	if err := os.Chdir(tmpDir); err != nil {
		return fmt.Errorf("enter port flow probe dir: %w", err)
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

	var violations []string
	current := ""
	ctx.onPort = func(name string, provided bool) {
		if !provided {
			violations = append(violations,
				fmt.Sprintf("capability %q reads port %q before it is provided", current, name))
		}
	}
	for _, d := range caps {
		current = d.Name
		if err := wireOne(ctx, d, byName); err != nil {
			return fmt.Errorf("assembly %q: %w", a.Name, err)
		}
	}
	if len(violations) > 0 {
		return fmt.Errorf("assembly %q: port flow violations: %s", a.Name, strings.Join(violations, "; "))
	}
	return nil
}
