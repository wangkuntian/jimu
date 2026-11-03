package main

import (
	"fmt"
	"maps"
	"slices"

	"jimu/internal/assembly"
	"jimu/internal/capabilities/catalog"
	"jimu/internal/capability"
	"jimu/internal/contract"
)

// compositionDescriptors 返回 catalog 与全部形态清单的并集（按能力名去重，catalog 优先），
// 以及全部已知能力名。非 catalog 能力（storage/ws/encryption/notification/grpc/apidocs）
// 不在 catalog 里，但同样参与组合契约校验。
func compositionDescriptors(asms map[string]assembly.Assembly) ([]contract.Descriptor, []string) {
	out := catalog.All()
	known := catalog.KnownNames()
	seen := make(map[string]bool, len(out))
	for _, d := range out {
		seen[d.Name] = true
	}
	for _, name := range slices.Sorted(maps.Keys(asms)) {
		for _, item := range asms[name].Capabilities {
			descriptor := item.Descriptor
			if seen[descriptor.Name] {
				continue
			}
			seen[descriptor.Name] = true
			out = append(out, descriptor)
			known = append(known, descriptor.Name)
		}
	}
	return out, known
}

// checkCompositionContract 校验全量声明（catalog ∪ 形态清单）的组合契约：
// 未知/自引用/重复冲突、重复提供方、必需消费者缺失和非法驱动声明。
// 除全仓并集外，每个形态单独再校验一次，确保 Required consumer 在该形态内可满足。
func checkCompositionContract(asms map[string]assembly.Assembly) error {
	caps, known := compositionDescriptors(asms)
	if err := capability.ValidateDeclarations(caps, known); err != nil {
		return err
	}
	for _, name := range slices.Sorted(maps.Keys(asms)) {
		profileCaps := make([]contract.Descriptor, 0, len(asms[name].Capabilities))
		for _, item := range asms[name].Capabilities {
			profileCaps = append(profileCaps, item.Descriptor)
		}
		if err := capability.ValidateDeclarations(profileCaps, known); err != nil {
			return fmt.Errorf("profile %q: %w", name, err)
		}
	}
	return nil
}
