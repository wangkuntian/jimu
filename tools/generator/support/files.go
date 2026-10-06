package support

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
)

// CopyTree 把 src 目录树下的文件复制到 dst（保留相对路径、文件权限位；目录统一 0o755）。
// filter != nil 时对每个**相对路径**（斜杠分隔）与 DirEntry 询问一次：目录返回 false 表示整棵
// 子树不进（只在 skipped 里登记该目录本身），文件返回 false 表示该文件不复制。
// 返回复制的文件数、被跳过条目（相对 src 的斜杠路径，排序去重：被过滤的条目与符号链接）与错误。
// src 不存在、不是目录或是符号链接即报错（fail-closed：绝不静默产出空项目）；dst 由本函数按需创建。
//
// 为什么需要它：本仓所有工具都是「读本仓」或「往本仓写模板」（tools/generator/module.go 的
// preflight 对任何已存在目标一律报错），没有「复制一棵目录树到另一个根」的原语，而生成项目
// 的本质正是按选择复制（recon §7「必须新写」第 1 项）。
func CopyTree(src, dst string, filter func(rel string, d fs.DirEntry) bool) (int, []string, error) {
	linfo, err := os.Lstat(src)
	if err != nil {
		return 0, nil, fmt.Errorf("copy tree source: %w", err)
	}
	if linfo.Mode()&fs.ModeSymlink != 0 {
		// WalkDir 对 root 用 Lstat：跟随的符号链接会被当成非目录静默跳过，复制 0 个文件。
		return 0, nil, fmt.Errorf("copy tree source %s is a symlink", src)
	}
	info, err := os.Stat(src)
	if err != nil {
		return 0, nil, fmt.Errorf("copy tree source: %w", err)
	}
	if !info.IsDir() {
		return 0, nil, fmt.Errorf("copy tree source %s is not a directory", src)
	}
	copied := 0
	skipped := make(map[string]bool)
	finish := func(err error) (int, []string, error) {
		names := make([]string, 0, len(skipped))
		for name := range skipped {
			names = append(names, name)
		}
		sort.Strings(names)
		return copied, names, err
	}
	err = filepath.WalkDir(src, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, p)
		if err != nil {
			return fmt.Errorf("relative path of %s: %w", p, err)
		}
		rel = filepath.ToSlash(rel)
		if rel == "." {
			// 根目录不参与过滤：filter 只对 src 之下的条目裁决。
			return nil
		}
		if filter != nil && !filter(rel, d) {
			skipped[rel] = true
			if d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		if d.IsDir() {
			return nil
		}
		if d.Type()&fs.ModeSymlink != 0 {
			// 符号链接既不跟随也不复制（相对链接在目标树里可能悬空），也不计入返回的文件数。
			skipped[rel] = true
			return nil
		}
		entryInfo, err := d.Info()
		if err != nil {
			return fmt.Errorf("stat %s: %w", rel, err)
		}
		content, err := os.ReadFile(p)
		if err != nil {
			return fmt.Errorf("read %s: %w", rel, err)
		}
		target := filepath.Join(dst, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return fmt.Errorf("create directory for %s: %w", rel, err)
		}
		if err := os.WriteFile(target, content, entryInfo.Mode().Perm()); err != nil {
			return fmt.Errorf("write %s: %w", rel, err)
		}
		copied++
		return nil
	})
	return finish(err)
}
