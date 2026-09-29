package app

import (
	"fmt"

	"jimu/internal/config"
	"jimu/internal/contract"
)

// CapabilityConfigs 已解码并校验的能力配置段集合，按 YAML 段键索引。
type CapabilityConfigs struct {
	bySection map[string]any
}

// Section 取某配置段实例（键为 Descriptor.Configs 中声明的 Section）。
// 未声明该段或对应能力未启用时返回 nil。
func (c *CapabilityConfigs) Section(key string) any {
	if c == nil {
		return nil
	}
	return c.bySection[key]
}

// Len 已加载的配置段数量（用于装配自检）。
func (c *CapabilityConfigs) Len() int {
	if c == nil {
		return 0
	}
	return len(c.bySection)
}

// LoadCapabilityConfigs 按启用集加载能力配置段：对每个启用能力的每个声明段
// 依次执行 解码 → ApplyDefaults → Validate（env=prod 时追加 ValidateProd）。
//
// 设计 §8：未启用的能力不在 caps 内，因此其配置段既不出现也不校验。
// 能力段必须实现 config.SectionConfig，否则立即报错（避免校验被静默跳过）。
func LoadCapabilityConfigs(dec config.SectionDecoder, caps []contract.Descriptor, env string) (*CapabilityConfigs, error) {
	out := &CapabilityConfigs{bySection: make(map[string]any)}
	for _, d := range caps {
		for _, spec := range d.Configs {
			if spec.Section == "" {
				return nil, fmt.Errorf("capability %s declares a config section with an empty key", d.Name)
			}
			if spec.New == nil {
				return nil, fmt.Errorf("capability %s config %q has no constructor", d.Name, spec.Section)
			}
			v := spec.New()
			sc, ok := v.(config.SectionConfig)
			if !ok {
				return nil, fmt.Errorf("capability %s config %q (%T) does not implement config.SectionConfig", d.Name, spec.Section, v)
			}
			if err := dec.UnmarshalKey(spec.Section, v); err != nil {
				return nil, fmt.Errorf("decode capability config %q: %w", spec.Section, err)
			}
			sc.ApplyDefaults()
			if err := sc.Validate(); err != nil {
				return nil, fmt.Errorf("invalid capability config %q: %w", spec.Section, err)
			}
			if env == "prod" {
				if pv, ok := v.(config.ProdConfigValidator); ok {
					if err := pv.ValidateProd(); err != nil {
						return nil, fmt.Errorf("invalid capability config %q: %w", spec.Section, err)
					}
				}
			}
			out.bySection[spec.Section] = v
		}
	}
	return out, nil
}

// SectionOf 取某配置段并断言为其具体类型（T 为指针类型，如 *captcha.Config）。
// 段不存在或类型不符时返回 (零值, false)。
func SectionOf[T any](c *CapabilityConfigs, key string) (T, bool) {
	var zero T
	v := c.Section(key)
	if v == nil {
		return zero, false
	}
	t, ok := v.(T)
	return t, ok
}
