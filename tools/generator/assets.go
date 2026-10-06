package generator

import (
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
//	声明集里每个能力的 Assets 并集 ∪ 全部内核资产组（P2.6 裁定 3：内核运维/观测资产全形态携带）。
//
// **口径是「声明集」，不是「形态」**：`--profile` 的清单在生成时已经展开进 set.Declared，故
// 形态生成与 `--with` 生成逐路径等价（TestAssetsForMatchesTheAssetTableTheGeneratedProjectDerives
// 钉住这一点）；而 `jimu capability add` 之后项目不再是任何形态，资产必须随声明集走 ——
// 否则新增能力（如 apidocs → docs/openapi）的资产不会被复制，生成项目第 5 条门禁会失真。
//
// 非 apidocs 形态不含 docs/openapi —— 这一步就是设计 §3.8「未选中资产不出现」的实际落点。
func AssetsFor(set CapabilitySet) ([]string, error) {
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
