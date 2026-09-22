package assembly

import (
	"fmt"
	"strings"

	"jimu/internal/app"
	"jimu/internal/capability"
	"jimu/internal/config"
	"jimu/internal/contract"
)

// ValidatePortFlow 校验装配清单的端口流向：按清单顺序试运行每个能力的 Wire，任何经
// Context.Port 读取的端口都必须在读取发生前已提供 —— 内核容器桥接端口
// （provideContainerPorts）或更早能力经 Provide 注册的端口。
//
// 这是过渡期对「内联 Wire 闭包」的装配顺序护栏：闭包无法静态内省，因此在零值内核件
// 上真实执行各 Wire 的 Provide/Port 调用并观察读取结果（只加载配置段，不构造容器、
// 不连库、不产生 I/O）。完整形态（full）里每个被读取的端口都应有人提供，缺失即报错；
// 因此它不适用于「软依赖确实缺席」的裁剪形态。Task 3 把内联闭包逐个搬成 <name>.Wire
// 后，本函数对同一清单继续有效。
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

	container := &app.Container{Config: cfg, CapabilityConfigs: capCfgs}
	ctx := newContext(container, sections, capCfgs)
	if err := provideContainerPorts(ctx, container); err != nil {
		return err
	}

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
