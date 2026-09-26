package generator

import (
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"jimu/tools/internal/profileassets"
)

// 本文件实现生成项目的**资产复制**（T6）：deploy/** 与 docs/openapi 的复制集派生、逐文件
// 落地、以及 deploy/helm/values.yaml 的顶层键裁剪（S6②）。
//
// 归属与派生只有一份来源（tools/internal/profileassets，P2.6）：能力用
// contract.Descriptor.Assets 声明资产，内核运维/观测资产由具名内核资产组声明（全形态携带），
// 同一路径按最长前缀取胜。生成器不新增第二套资产表 —— 否则「复制的」与「门禁校验的」会漂移。

// AssetsFor 返回要复制到生成项目的资产路径（目录或单文件，仓库相对路径，升序去重）。
// 与 P2.6 的归属派生**同一来源**（tools/internal/profileassets）：
//
//	--profile：profileassets.ForProfile(name)（= 该形态能力 Assets 并集 ∪ 全部内核资产组）；
//	--with   ：profileassets.Declared() 里 cap:<选中能力> 的路径并集 ∪ profileassets.CoreGroups()。
//
// 非 apidocs 形态不含 docs/openapi —— 这一步就是设计 §3.8「未选中资产不出现」的实际落点。
func AssetsFor(set CapabilitySet) ([]string, error) {
	if set.Profile != "" {
		return profileassets.ForProfile(set.Profile)
	}
	known := make(map[string]bool, len(set.Known))
	for _, name := range set.Known {
		known[name] = true
	}
	declared := profileassets.Declared()
	selected := map[string]bool{}
	for _, name := range set.Declared {
		if len(known) > 0 && !known[name] {
			return nil, fmt.Errorf("capability %q in selection is not a known capability", name)
		}
		// 没有声明资产的能力在 Declared() 里没有条目（绝大多数能力如此），不是错误。
		for _, p := range declared["cap:"+name] {
			if canonical := profileassets.Canonical(p); canonical != "" {
				selected[canonical] = true
			}
		}
	}
	// 内核资产组全形态携带（P2.6 裁定 3）。
	for _, paths := range profileassets.CoreGroups() {
		for _, p := range paths {
			if canonical := profileassets.Canonical(p); canonical != "" {
				selected[canonical] = true
			}
		}
	}
	return slices.Sorted(maps.Keys(selected)), nil
}

// CopyAssets 把资产**文件**逐个复制到 dst（保留目录结构与权限位）：先由 assetFiles 求出
// 「会复制的文件清单」，再逐个复制 —— 两处口径不可能漂移（复制多少就是清单里多少）。
// 返回复制的相对路径（排序去重）。
func CopyAssets(root, dst string, assets []string) ([]string, error) {
	files, err := assetFiles(root, assets)
	if err != nil {
		return nil, err
	}
	for _, rel := range files {
		if err := copyOneFile(root, dst, rel); err != nil {
			return nil, err
		}
	}
	return files, nil
}

// assetFiles 返回 assets 前缀下会被复制的全部文件（仓库相对路径，升序去重）。
//
// 遍历口径与 profileassets.OwnershipIn **逐条一致**（跳过以 "." 开头的文件与目录），否则会出现
// 「复制了但门禁校验不到」或「门禁要求存在却没复制」的静默不一致 —— 这正是 P2.6 把归属做成
// 单一派生要避免的。符号链接既不跟随也不复制（CopyTree 同规则），但声明路径本身是符号链接时
// fail-closed 报错：静默产出空目录比失败更难排查。
func assetFiles(root string, assets []string) ([]string, error) {
	found := map[string]bool{}
	for _, raw := range assets {
		prefix := profileassets.Canonical(raw)
		if prefix == "" {
			return nil, fmt.Errorf("asset path %q is empty after canonicalization", raw)
		}
		if prefix == ".." || strings.HasPrefix(prefix, "../") || filepath.IsAbs(prefix) {
			return nil, fmt.Errorf("asset path %q escapes the repository root", raw)
		}
		src := filepath.Join(root, filepath.FromSlash(prefix))
		info, err := os.Lstat(src)
		if err != nil {
			return nil, fmt.Errorf("asset %s: %w", prefix, err)
		}
		if info.Mode()&fs.ModeSymlink != 0 {
			return nil, fmt.Errorf("asset %s is a symlink", prefix)
		}
		if !info.IsDir() {
			found[prefix] = true
			continue
		}
		err = filepath.WalkDir(src, func(p string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if strings.HasPrefix(d.Name(), ".") {
				if d.IsDir() {
					return fs.SkipDir
				}
				return nil
			}
			if d.IsDir() || d.Type()&fs.ModeSymlink != 0 {
				return nil
			}
			found[relPath(root, p)] = true
			return nil
		})
		if err != nil {
			return nil, fmt.Errorf("walk asset %s: %w", prefix, err)
		}
	}
	return slices.Sorted(maps.Keys(found)), nil
}

// valuesRelPath 是唯一需要按能力裁剪的内核资产（S6②：设计 §3.8 明列的 values.yaml）。
const valuesRelPath = "deploy/helm/values.yaml"

// renderValuesYAML 把已复制的 deploy/helm/values.yaml 顶层键按能力集裁剪后落盘。
// 键选择与渲染**复用 T4 的实现**（ValuesSections / RenderValuesYAML，裁定 ⑫），本函数只负责接线：
// 资产的落盘在 T6，段选择器在 T4。
//
// 文件不存在时跳过（该选择不含 helm chart）；存在则必须解析成功（SectionBlocks 遇到无法识别的
// 顶层行即报错，绝不静默丢内容）。
func renderValuesYAML(root, dst string, set CapabilitySet) error {
	target := filepath.Join(dst, filepath.FromSlash(valuesRelPath))
	if _, err := os.Stat(target); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("inspect %s: %w", valuesRelPath, err)
	}
	src, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(valuesRelPath)))
	if err != nil {
		return fmt.Errorf("read %s: %w", valuesRelPath, err)
	}
	keep, err := ValuesSections(src, set)
	if err != nil {
		return fmt.Errorf("select %s keys: %w", valuesRelPath, err)
	}
	out, err := RenderValuesYAML(src, keep)
	if err != nil {
		return fmt.Errorf("render %s: %w", valuesRelPath, err)
	}
	if err := writeFile(target, out, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", valuesRelPath, err)
	}
	return nil
}
