// Package profileassets 派生「谁拥有哪些非代码资产」（P2.6），供门禁
// （tools/checkcapabilities）与查询工具（tools/profileassets）共用，避免两处漂移
// （先例：tools/internal/heavydeps、tools/internal/profileoverlay）。
//
// 归属规则：能力用 contract.Descriptor.Assets 声明自己的资产；内核运维/观测资产由下面的
// 具名资产组声明（不属于任何能力，**全形态携带**）。同一路径被多处声明时按**最长前缀**取胜
// （具体文件赢过目录），因此每个资产文件恰有一个有效所有者。
// 所有者标识：能力 "cap:<name>"、内核组 "group:<name>"。
package profileassets

import (
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"

	"jimu/internal/capabilities/catalog"
	"jimu/internal/contract"
	"jimu/internal/profiles/registry"
)

// coreGroups 内核资产组：不属于任何能力，全形态携带。
// 生成项目是否携带由 P2.7 的生成器决定；本阶段只表达归属。
//
// observability 用更具体的前缀抢走 ops 目录里的观测清单（最长前缀匹配），
// 因此 deploy/k8s 整体归 ops，而其中的 openobserve/otel 两个文件归 observability。
var coreGroups = map[string][]string{
	"ops": {
		"deploy/k8s",
		"deploy/helm",
		"deploy/backup",
	},
	"observability": {
		"deploy/openobserve",
		"deploy/otel-collector.yaml",
		"deploy/k8s/openobserve.yaml",
		"deploy/k8s/otel-collector.yaml",
		"deploy/helm/templates/openobserve.yaml",
		"deploy/helm/templates/otel-collector.yaml",
	},
}

// AssetRoots 是门禁扫描的资产根。
// 不纳入 configs/：本仓 configs/*.yaml 逐字节不变（按能力渲染只对生成项目成立，归 P2.7）。
func AssetRoots() []string { return []string{"deploy", "docs/openapi"} }

// CoreGroups 返回内核资产组（副本，调用方修改不影响包级表）。
func CoreGroups() map[string][]string {
	out := make(map[string][]string, len(coreGroups))
	for name, paths := range coreGroups {
		out[name] = append([]string(nil), paths...)
	}
	return out
}

// Declared 返回「所有者 → 声明的前缀」全表（能力 ∪ 内核组），是归属判定的唯一来源。
func Declared() map[string][]string {
	out := map[string][]string{}
	for name, d := range capabilityDescriptors() {
		if len(d.Assets) == 0 {
			continue
		}
		paths := append([]string(nil), d.Assets...)
		slices.Sort(paths)
		out["cap:"+name] = paths
	}
	for name, paths := range coreGroups {
		out["group:"+name] = append([]string(nil), paths...)
	}
	return out
}

// Owner 用最长前缀匹配返回路径的有效所有者；不在任何声明前缀下时 ok=false。
func Owner(path string) (string, bool) { return OwnerIn(Declared(), path) }

// Canonical 归一化资产路径/前缀：去空白、转正斜杠、path.Clean（解析 "." 与 ".."、
// 折叠重复斜杠、去掉结尾 "/"）。
//
// **单一实现**：归属匹配、声明去重、资产根判断与形态资产集都必须走它，否则同一份声明在
// 不同环节会被解释成不同路径（"deploy/k8s/"、"deploy/k8s//"、"deploy/../configs" 这类变体
// 曾经能绕过门禁并静默误判归属）。
func Canonical(p string) string {
	p = strings.TrimSpace(filepath.ToSlash(p))
	if p == "" {
		return ""
	}
	p = path.Clean(p) // 同时去掉结尾 "/" 并解析 "."/".."
	if p == "." || p == "/" {
		return ""
	}
	return p
}

// OwnerIn 与 Owner 同义，但用调用方提供的声明表（测试与自定义场景）。
//
// 等长并列（两个不同所有者声明了同一个归一化前缀）视为**无有效所有者**：fail-closed，
// 不按所有者名排序抽一个。门禁的「重复声明」检查会先一步报出更明确的错误。
func OwnerIn(declared map[string][]string, path string) (string, bool) {
	path = Canonical(path)
	if path == "" {
		return "", false
	}
	best, bestLen, tied := "", -1, false
	for _, owner := range slices.Sorted(maps.Keys(declared)) {
		for _, prefix := range declared[owner] {
			prefix = Canonical(prefix)
			if prefix == "" {
				continue
			}
			if path == prefix || strings.HasPrefix(path, prefix+"/") {
				switch {
				case len(prefix) > bestLen:
					best, bestLen, tied = owner, len(prefix), false
				case len(prefix) == bestLen && owner != best:
					tied = true
				}
			}
		}
	}
	if tied {
		return "", false
	}
	return best, bestLen >= 0
}

// Ownership 遍历资产根下的所有文件（跳过以 "." 开头的文件与目录），返回 path → owner。
// 资产根不存在时跳过；有文件没有有效所有者时报错。
func Ownership(root string) (map[string]string, error) {
	return OwnershipIn(root, Declared())
}

// OwnershipIn 与 Ownership 同义，但用调用方提供的声明表（测试与自定义场景）。
func OwnershipIn(root string, declared map[string][]string) (map[string]string, error) {
	out := map[string]string{}
	for _, rel := range AssetRoots() {
		base := filepath.Join(root, filepath.FromSlash(rel))
		if _, err := os.Stat(base); err != nil {
			// 资产根本次不存在（如生成项目的裁剪结果）→ 无可校验；
			// 其它错误（如权限）不吞掉，交给调用方 fail-closed。
			if errors.Is(err, fs.ErrNotExist) {
				continue
			}
			return nil, err
		}
		err := filepath.WalkDir(base, func(p string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if strings.HasPrefix(d.Name(), ".") {
				if d.IsDir() {
					return fs.SkipDir
				}
				return nil
			}
			if d.IsDir() {
				return nil
			}
			asset, err := filepath.Rel(root, p)
			if err != nil {
				return err
			}
			asset = filepath.ToSlash(asset)
			owner, ok := OwnerIn(declared, asset)
			if !ok {
				return fmt.Errorf("asset %q has no effective owner", asset)
			}
			out[asset] = owner
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	return out, nil
}

// ForProfile 返回该形态的资产路径（该形态能力的 Assets 并集 + 全部内核资产组），升序去重。
// 未知形态返回错误（文案与 registry 一致）。
func ForProfile(profile string) ([]string, error) {
	a, err := registry.Lookup(profile)
	if err != nil {
		return nil, err
	}
	descs := capabilityDescriptors()
	set := map[string]bool{}
	// 入集合前一律 Canonical：派生输出必须与归属判定同一口径，
	// 否则消费方（Makefile / P2.7 生成器）会拿到与校验过的声明不一致的路径。
	for _, c := range a.Capabilities {
		for _, p := range descs[c.Descriptor.Name].Assets {
			if canonical := Canonical(p); canonical != "" {
				set[canonical] = true
			}
		}
	}
	// 内核资产组全形态携带（P2.6 裁定 3）。
	for _, paths := range coreGroups {
		for _, p := range paths {
			if canonical := Canonical(p); canonical != "" {
				set[canonical] = true
			}
		}
	}
	return slices.Sorted(maps.Keys(set)), nil
}

// capabilityDescriptors 汇总「能力名 → Descriptor」：catalog 全量 ∪ 各形态清单的并集。
// 非 catalog 的 Ungated 条目（如 apidocs）只出现在形态清单里，必须一并取到。
func capabilityDescriptors() map[string]contract.Descriptor {
	out := map[string]contract.Descriptor{}
	for _, d := range catalog.All() {
		out[d.Name] = d
	}
	for _, a := range registry.All() {
		for _, c := range a.Capabilities {
			out[c.Descriptor.Name] = c.Descriptor
		}
	}
	return out
}
