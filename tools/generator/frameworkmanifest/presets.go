package frameworkmanifest

import (
	"fmt"
	"slices"

	"jimu/internal/profiles/full"
)

// Preset 是 jimu new 的一个官方起点：能力集 + 驱动集 + 支持级别。
//
// 预设只是现有裁剪引擎的入口，不引入第二套物化逻辑 —— 它们最终都走
// completeSelection 的「声明集 ∪ 编译闭包 ∪ 迁移携带」裁剪。
type Preset struct {
	Name         string
	Capabilities []string
	Drivers      map[string][]string
	// Support 沿用旧模板的词汇（official/validated），由 compositionSupport 映射为
	// manifest 的 certified/validated/experimental。
	Support string
}

// presetOrder 固定对外顺序：CLI 帮助、预算表与门禁都依赖它，不要按 map 迭代。
var presetOrder = []string{"minimal", "auth", "full"}

// PresetNames 返回全部预设名，顺序固定。
func PresetNames() []string { return slices.Clone(presetOrder) }

// LookupPreset 按名解析预设；未知名字列出全部可用预设。
func LookupPreset(name string) (Preset, error) {
	switch name {
	case "minimal":
		return Preset{Name: "minimal", Support: "official"}, nil
	case "auth":
		return Preset{
			Name:         "auth",
			Capabilities: []string{"user", "access", "auth"},
			Drivers:      map[string][]string{"auth": {"redis"}},
			Support:      "official",
		}, nil
	case "full":
		return fullPreset(), nil
	default:
		return Preset{}, fmt.Errorf("unknown template %q (available: %s)", name, joinNames(PresetNames()))
	}
}

// fullPreset 从框架 full 形态逐项派生，而不是硬编码能力名与驱动。
//
// 关键在驱动：--with 的缺省规则是「取每个能力的首个驱动」，会丢掉
// storage:s3、queue:kafka/rabbitmq、dataops:excel。full 必须带全集，
// 否则产物与框架 full 二进制不等价。
func fullPreset() Preset {
	preset := Preset{Name: "full", Drivers: map[string][]string{}, Support: "validated"}
	for _, item := range full.Assembly().Capabilities {
		preset.Capabilities = append(preset.Capabilities, item.Descriptor.Name)
		if len(item.Drivers) > 0 {
			preset.Drivers[item.Descriptor.Name] = slices.Clone(item.Drivers)
		}
	}
	return preset
}

func joinNames(names []string) string {
	out := ""
	for i, name := range names {
		if i > 0 {
			out += ", "
		}
		out += name
	}
	return out
}
