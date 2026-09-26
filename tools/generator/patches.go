package generator

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// copyTools 是复制进生成项目的工具树（**不含** tools/generator —— 生成器自身不进产物）。
// 生成项目里这些工具必须能**在该项目内**工作（单形态）：checkcapabilities / profiles-check /
// compose-report / logcheck 的目标直接依赖它们，缺一个对应 make 目标就不可用。
var copyTools = []string{
	"tools/internal/profileoverlay",
	"tools/internal/profileassets",
	"tools/internal/heavydeps",
	"tools/profileoverlay",
	"tools/profileassets",
	"tools/checkcapabilities",
	"tools/composereport",
	"tools/logcheck",
}

// modulePlaceholder 是补丁替换文本里的模块路径占位符：filePatches 的替换值是**编译期常量
// 文本**，只有模块路径要到生成时才确定，故用占位符而不是散落的 if。
const modulePlaceholder = "{{module}}"

// filePatches 是复制文件的**定点改写表**（键 = 相对路径，值 = 有序字面量替换：
// {原文, 替换}，替换文本里的 {{module}} 会换成 --module 的值）。
//
// 只改「复制到生成项目后不再成立」的东西，不改逻辑、不改行为方向：
//
//	modulePath 常量 —— tools/{checkcapabilities,composereport} 的 `const modulePath = "jimu"`
//	                  是裸字符串（没有 "jimu/" 前缀），受控重写的 import 规则抓不到；
//	                  不改则生成项目的驱动归属判定与闭包度量按 jimu/... 统计，恒为空。
//	                  （RewriteModule 也有一条同名规则；本表显式列出后先落地，重写对其幂等。）
//	logcheck 自跳过 —— `p.PkgPath == "jimu/tools/logcheck"` 同样是裸字符串；不改则生成项目里
//	                  logcheck 不再跳过自己。
//	profileoverlay 替换模板 —— overlay 替换内容的 import 写在 fmt.Sprintf 的原始字符串里
//	                  （不是 ImportSpec），import 重写规则抓不到；不改则生成项目里
//	                  profileoverlay 产出仍 import jimu/... 的选点文件，checkcapabilities /
//	                  profiles-check 的 packages.Load 直接失败（这是审查实测的必改点之一）。
//	composereport 口径文案 —— 报告表格/说明写死「各形态（profile）」与框架 module path 示例，
//	                  单形态项目里不成立。
//
// 命中失败即报错（fail-closed：文案改了却没人同步补丁，好过静默跳过）。
var filePatches = map[string][][2]string{
	"tools/checkcapabilities/drivers.go": {
		// 冗余但**有意保留**（belt & braces）：rewrite.go 的 `.go` 重写也认这条声明名 + 值，
		// 本表先落地、随后的重写对它幂等。显式登记便于审查「生成项目里的裸 module path」时
		// 一眼看全，请勿以「重写已覆盖」为由删除。
		{`const modulePath = "jimu"`, `const modulePath = "{{module}}"`},
	},
	"tools/composereport/main.go": {
		// 同上：冗余但有意保留（belt & braces）。
		{`const modulePath = "jimu"`, `const modulePath = "{{module}}"`},
		{
			"// Command composereport 生成各形态（profile）的「编译面」报告。",
			"// Command composereport 生成本项目唯一形态的「编译面」报告。",
		},
		{
			"报告回答「层②（形态选点）究竟改变了什么」",
			"报告回答「本项目编进了什么」",
		},
		{
			"数在各形态间**完全相同**",
			"数不再随形态变化",
		},
		{
			// 报告正文里的模块路径示例同样是**框架** module path（.go 的 import 重写只动
			// ImportSpec，不动字符串字面量）；不改则生成项目的报告写「本模块（jimu/...）」。
			"（`jimu/...`）",
			"（`{{module}}/...`）",
		},
	},
	"tools/logcheck/main.go": {
		{`"jimu/tools/logcheck"`, `"{{module}}/tools/logcheck"`},
	},
	"tools/internal/profileoverlay/profileoverlay.go": {
		// overlay 替换内容的 import 写在 fmt.Sprintf 的**原始字符串**里（不是 ImportSpec），
		// 受控重写的 import 规则抓不到；不改则生成项目里 profileoverlay 生成出仍 import
		// jimu/... 的选点文件，checkcapabilities / profiles-check 的 packages.Load 直接失败。
		{`"jimu/internal/assembly"`, `"{{module}}/internal/assembly"`},
		{`"jimu/internal/profiles/%s"`, `"{{module}}/internal/profiles/%s"`},
	},
}

// applyFilePatches 在复制完成后的生成树上应用 filePatches（按路径排序，保证确定性）。
func applyFilePatches(dst, module string) error {
	for _, rel := range sortedPatchPaths() {
		path := filepath.Join(dst, filepath.FromSlash(rel))
		if err := applyPatchFile(path, filePatches[rel], module); err != nil {
			return err
		}
	}
	return nil
}

// applyPatchFile 对单个文件应用有序字面量替换；任一条原文未命中即报错（fail-closed，绝不静默
// 跳过：文案漂移而没人同步补丁时，生成项目会带着错误口径继续跑）。全部替换完成后一次写盘。
func applyPatchFile(path string, patches [][2]string, module string) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("patch %s: %w", filepathSlash(path), err)
	}
	text := string(content)
	for _, p := range patches {
		if !strings.Contains(text, p[0]) {
			return fmt.Errorf("patch %s: 原文未命中（补丁表与蓝本漂移）: %q", filepathSlash(path), p[0])
		}
		text = strings.Replace(text, p[0], strings.ReplaceAll(p[1], modulePlaceholder, module), 1)
	}
	if text == string(content) {
		return nil
	}
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("patch %s: %w", filepathSlash(path), err)
	}
	if err := os.WriteFile(path, []byte(text), info.Mode().Perm()); err != nil {
		return fmt.Errorf("patch %s: %w", filepathSlash(path), err)
	}
	return nil
}

// skipToolTestFile 判定工具树里不复制框架侧的 `_test.go`：它们断言的是**框架**口径
// （多形态 registry 清单、框架 module path 字面量、框架资产表、5 形态 golden），在单形态
// 生成项目里既不成立也没有意义；生成项目的门禁是 make check-capabilities / profiles-check。
// 生产文件一律复制（工具必须能在生成项目内工作）。
func skipToolTestFile(base string) bool {
	return strings.HasSuffix(base, "_test.go")
}

// sortedPatchPaths 返回 filePatches 的键（排序）：map 遍历顺序随机，补丁应用必须确定。
func sortedPatchPaths() []string {
	out := make([]string, 0, len(filePatches))
	for rel := range filePatches {
		out = append(out, rel)
	}
	sort.Strings(out)
	return out
}
