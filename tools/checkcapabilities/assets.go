// 资产归属段（P2.6）：非代码资产必须「每个文件恰有一个有效所有者、无未声明资产」，
// 且声明的路径必须真实存在、落在资产根内。归属规则与派生逻辑在
// tools/internal/profileassets（与 tools/profileassets 共用同一份，避免两处漂移）。
package main

import (
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"jimu/internal/profiles/registry"
	"jimu/tools/internal/profileassets"
)

// checkAssets 用真实声明表校验仓库资产归属，并断言每个形态都覆盖全部内核资产组。
func checkAssets(root string) error {
	if err := checkAssetsIn(root, profileassets.Declared()); err != nil {
		return err
	}
	return checkProfileCoreCoverage(registry.Names(), profileassets.ForProfile, profileassets.CoreGroups())
}

// checkAssetsIn 校验资产归属（可传夹具，便于单测）：
//
//	① 每个声明的资产路径非空、存在（目录或文件），且**落在资产根内**（否则永远不会被扫描，
//	   等于把未经校验的路径交给 P2.7 生成器）；
//	② 同一路径不得被两个所有者声明（能力之间、能力与内核资产组之间）—— 比较用
//	   profileassets.Canonical 归一化后的路径，否则 "deploy/k8s/"、"deploy/k8s//"、
//	   "deploy/k8s/." 这类变体会绕过检查，并在等长匹配时静默误判归属；
//	③ 资产根下每个文件恰有一个有效所有者（最长前缀匹配），无未声明资产。
//
// 归一化只有 profileassets.Canonical 一份实现：去重键、资产根判断、前缀匹配与传给
// OwnershipIn 的表都用它，避免「门禁校验的形态」与「消费的形态」不一致。
func checkAssetsIn(root string, declared map[string][]string) error {
	roots := profileassets.AssetRoots()
	canonical := make(map[string][]string, len(declared))
	seen := map[string]string{} // 归一化声明路径 -> 所有者
	for _, owner := range slices.Sorted(maps.Keys(declared)) {
		for _, raw := range declared[owner] {
			p := profileassets.Canonical(raw)
			if p == "" {
				return fmt.Errorf("owner %q declares an empty asset path", owner)
			}
			if prev, dup := seen[p]; dup {
				return fmt.Errorf("asset path %q is declared by both %q and %q", p, prev, owner)
			}
			seen[p] = owner
			canonical[owner] = append(canonical[owner], p)
			if !underAnyRoot(p, roots) {
				return fmt.Errorf("asset path %q declared by %q is outside the asset roots %v", p, owner, roots)
			}
			if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(p))); err != nil {
				return fmt.Errorf("asset path %q declared by %q is missing: %w", p, owner, err)
			}
		}
	}
	_, err := profileassets.OwnershipIn(root, canonical)
	return err
}

// checkProfileCoreCoverage 断言每个形态的资产集都覆盖全部内核资产组（可注入便于测试）：
// 内核资产组是全形态携带的，任何形态漏掉它都意味着「不该被裁剪的资产随形态消失了」。
func checkProfileCoreCoverage(names []string, forProfile func(string) ([]string, error), core map[string][]string) error {
	var missing []string
	for _, name := range names {
		paths, err := forProfile(name)
		if err != nil {
			return err
		}
		have := make(map[string]bool, len(paths))
		for _, p := range paths {
			have[p] = true
		}
		for group, prefixes := range core {
			for _, p := range prefixes {
				if !have[p] {
					missing = append(missing, fmt.Sprintf("%s/%s:%s", group, name, p))
				}
			}
		}
	}
	slices.Sort(missing)
	if len(missing) > 0 {
		return fmt.Errorf("profiles missing kernel asset groups: %s", strings.Join(missing, ", "))
	}
	return nil
}

// underAnyRoot 判断（已归一化的）路径是否落在某个资产根之下（根自身算在内）。
func underAnyRoot(p string, roots []string) bool {
	for _, r := range roots {
		r = profileassets.Canonical(r)
		if p == r || strings.HasPrefix(p, r+"/") {
			return true
		}
	}
	return false
}
