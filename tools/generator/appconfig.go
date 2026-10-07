package generator

import (
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"

	"jimu/internal/capabilities/catalog"
	"jimu/internal/config"
)

// 本文件实现生成项目 configs/*.yaml 的**段渲染**（S6①）。段集合由三类来源派生：
//
//	① 内核段   —— internal/config.Config 的 mapstructure 顶层字段（unmarshalConfig 覆盖的全部段）；
//	② 声明段   —— 选中能力的 contract.Descriptor.Configs（只有 7 个能力声明）；
//	③ 自读段   —— 能力在自己的 Load() 里 config.LoadSection 读、但**未**在 Configs 声明的段
//	              （storage/notification，见 selfReadSections）。
//
// 只对**生成项目**成立：本仓 configs/*.yaml 是逐字节冻结的蓝本，生成器只读不改。

// KernelSections 返回内核段（升序）：直接反射 internal/config.Config 的顶层 mapstructure
// tag —— 与 config.go 的 unmarshalConfig 覆盖范围同源，不写死清单（漂移护栏见
// TestKernelSectionsMatchConfigStruct）。内核段在生成项目里恒保留。
func KernelSections() []string {
	typ := reflect.TypeOf(config.Config{})
	out := make([]string, 0, typ.NumField())
	for i := range typ.NumField() {
		name, _, _ := strings.Cut(typ.Field(i).Tag.Get("mapstructure"), ",")
		if name == "" || name == "-" {
			continue // "-" 是运行时元数据（Version/Environment），不是配置段
		}
		out = append(out, name)
	}
	slices.Sort(out)
	return out
}

// selfReadSections 是「能力自读但未在 Descriptor.Configs 声明」的段映射（第三类来源）：
// 非 catalog（Ungated）能力在自己的 Load() 里直接 config.LoadSection(dec, <段>, …)，只靠
// Descriptor.Configs 派生段集合会漏掉它们（storage/config.go、notification/config.go），
// 生成项目的 configs/app.yaml 会缺段。
//
// 键是能力名，值是该能力拥有的段（按源码顺序）。漂移护栏：
// TestSelfReadSectionsMatchTheFrameworkSources 扫描 internal/capabilities/**/config.go 的
// LoadSection 实参逐值比对 —— 这张表与能力源码必须始终一致。
var selfReadSections = map[string][]string{
	"storage":      {"storage"},
	"notification": {"email", "sms", "notification"},
}

// SectionsFor 返回生成项目 configs/*.yaml 该保留的顶层段（升序去重）：
// 内核段 ∪ 选中能力的 Descriptor.Configs 段 ∪ 选中能力的 selfReadSections 段。
//
// 「选中能力」= set.Declared（声明集，不含只作迁移/编译携带的能力）：未声明的能力其段既不
// 出现也不校验（设计 §8）。set.Known 非空时对声明集做一次成员校验（fail-closed）。
func SectionsFor(set CapabilitySet) ([]string, error) {
	known := make(map[string]bool, len(set.Known))
	for _, name := range set.Known {
		known[name] = true
	}
	declared := make(map[string]bool, len(set.Declared))
	for _, name := range set.Declared {
		if len(known) > 0 && !known[name] {
			return nil, fmt.Errorf("capability %q in selection is not a known capability", name)
		}
		declared[name] = true
	}

	keep := make(map[string]bool, 32)
	for _, section := range KernelSections() {
		keep[section] = true
	}
	// ② 声明段：catalog 的 Descriptor.Configs（非 catalog 能力当前都不声明 Configs；
	// 若将来声明，TestSectionsSourcesExplainEveryAppYAMLKey 会因段集合对不上而红）。
	for _, desc := range catalog.All() {
		if !declared[desc.Name] {
			continue
		}
		for _, spec := range desc.Configs {
			keep[spec.Section] = true
		}
	}
	// ③ 自读段。
	for name := range declared {
		for _, section := range selfReadSections[name] {
			keep[section] = true
		}
	}
	return slices.Sorted(maps.Keys(keep)), nil
}

// SectionBlocks 把一份 YAML 的顶层键切成有序块：块首的连续注释/空行归属**紧随其后**的段
// （块首注释随该段一起保留或一起丢弃）；段内注释（其后仍有缩进行）留在本段。
// 返回段键顺序（纯注释文件返回单个空键块）与段原文；拼接所有块逐字节等于 src
// （TestAppConfigSectionBlocksRoundTripsRepoFiles 钉住这一点）。
// 遇到无法识别的顶层行（缩进异常/不是 `key:`）即报错，绝不静默丢内容。
func SectionBlocks(src []byte) (keys []string, blocks []string, err error) {
	var pending []string
	for i, line := range splitKeepEOL(src) {
		body := strings.TrimRight(line, "\r\n")
		switch {
		case strings.TrimSpace(body) == "", strings.HasPrefix(strings.TrimLeft(body, " \t"), "#"):
			pending = append(pending, line)
		case body[0] == ' ' || body[0] == '\t':
			if len(blocks) == 0 {
				return nil, nil, fmt.Errorf("section parser: line %d: indented line before any top-level key: %q", i+1, body)
			}
			// 段内注释后仍有缩进行：按原顺序并回本段（不能把它推给下一段）。
			blocks[len(blocks)-1] += strings.Join(pending, "")
			pending = pending[:0]
			blocks[len(blocks)-1] += line
		default:
			key, ok := topLevelKey(body)
			if !ok {
				return nil, nil, fmt.Errorf("section parser: line %d: not a top-level `key:` line: %q", i+1, body)
			}
			keys = append(keys, key)
			blocks = append(blocks, strings.Join(pending, "")+line)
			pending = pending[:0]
		}
	}
	// 文件尾部的注释/空行：挂在最后一个段之后（纯注释文件则自成一块），不丢。
	if len(pending) > 0 {
		if len(blocks) == 0 {
			keys = append(keys, "")
			blocks = append(blocks, strings.Join(pending, ""))
		} else {
			blocks[len(blocks)-1] += strings.Join(pending, "")
		}
	}
	return keys, blocks, nil
}

// RenderAppConfig 按 keep 选择段并拼接（保持源文件段序）：只对**生成项目**成立；
// 本仓 configs/*.yaml 逐字节不变（Global Constraints）。app.yaml 与 app.prod.yaml 用同一份
// keep（S6①），否则 prod 下未选中能力的段会重新出现；keep 里有而源文件没有的段（app.prod.yaml
// 无 capabilities）不算错，只是无处可留。
func RenderAppConfig(src []byte, keep []string) ([]byte, error) {
	return renderSections(src, keep)
}

// RenderConfigs 渲染生成项目的 configs/*.yaml（替换 T2 的「原样复制」占位）：
// configs/app.yaml 与 configs/app.prod.yaml 同口径段过滤。
func RenderConfigs(root, dst string, set CapabilitySet) error {
	sections, err := SectionsFor(set)
	if err != nil {
		return err
	}
	for _, rel := range configFiles() {
		src, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
		if err != nil {
			return fmt.Errorf("read %s: %w", rel, err)
		}
		out, err := RenderAppConfig(src, sections)
		if err != nil {
			return fmt.Errorf("render %s: %w", rel, err)
		}
		target := filepath.Join(dst, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return fmt.Errorf("create directory for %s: %w", rel, err)
		}
		if err := writeFile(target, out, 0o644); err != nil {
			return fmt.Errorf("write %s: %w", rel, err)
		}
	}
	return nil
}

// configFiles 返回 configs 渲染产物的相对路径。
func configFiles() []string {
	return []string{"configs/app.yaml", "configs/app.prod.yaml"}
}

// valuesCapabilityKeys 是 deploy/helm/values.yaml 里「与能力同名」的顶层键 → 能力名（S6②）。
// 本仓 24 个顶层键里只有这三个与能力同名（其余 21 个是内核键）；漂移护栏见
// TestValuesCapabilityKeysMatchRepo。
var valuesCapabilityKeys = map[string]string{
	"audit":   "audit",
	"storage": "storage",
	"auth":    "auth",
}

// ValuesSections 返回生成项目 deploy/helm/values.yaml 该保留的顶层键（按源文件顺序）：
// 全部内核键（源文件里不与能力同名的键）∪ 选中能力同名的键。未选中能力的同名键不出现。
//
// 注意：本函数只做**键裁剪**；values.yaml 的落盘属 T6 的资产复制范围（RenderValuesYAML 已备好）。
func ValuesSections(src []byte, set CapabilitySet) ([]string, error) {
	keys, _, err := SectionBlocks(src)
	if err != nil {
		return nil, err
	}
	declared := make(map[string]bool, len(set.Declared))
	for _, name := range set.Declared {
		declared[name] = true
	}
	out := make([]string, 0, len(keys))
	for _, key := range keys {
		if name, ok := valuesCapabilityKeys[key]; ok && !declared[name] {
			continue
		}
		out = append(out, key)
	}
	return out, nil
}

// RenderValuesYAML 按 keep 裁剪 values.yaml 的顶层键（与 RenderAppConfig 同一段选择器）。
func RenderValuesYAML(src []byte, keep []string) ([]byte, error) {
	return renderSections(src, keep)
}

// renderSections 按 keep 拼接被保留的段原文（空键块 = 纯注释块，恒保留）。
func renderSections(src []byte, keep []string) ([]byte, error) {
	keys, blocks, err := SectionBlocks(src)
	if err != nil {
		return nil, err
	}
	want := make(map[string]bool, len(keep))
	for _, key := range keep {
		want[key] = true
	}
	var out strings.Builder
	out.Grow(len(src))
	for i, key := range keys {
		if key != "" && !want[key] {
			continue
		}
		out.WriteString(blocks[i])
	}
	return []byte(out.String()), nil
}

// splitKeepEOL 按行切分并保留行尾（含最后一个无换行的行），保证拼接可逐字节还原。
func splitKeepEOL(src []byte) []string {
	if len(src) == 0 {
		return nil
	}
	lines := strings.SplitAfter(string(src), "\n")
	if last := len(lines) - 1; lines[last] == "" {
		lines = lines[:last]
	}
	return lines
}

// topLevelKey 判断一行是否是顶层 `key:` 行并取键名（支持 "quoted" / 'quoted' 键）。
// YAML 里键与值之间的冒号后必须跟空白行尾或注释，故 `http://x` 这类标量不会被误认成键。
// 顶层块序列项（`- name: x`）不是映射键，必须返回 false（否则会被当成键名 "- name" 而静默丢弃，
// 与「绝不静默丢内容」相悖）。
func topLevelKey(line string) (string, bool) {
	if line == "" {
		return "", false
	}
	if line[0] == '-' && (len(line) == 1 || line[1] == ' ' || line[1] == '\t') {
		return "", false
	}
	if line[0] == '"' || line[0] == '\'' {
		quote := line[0]
		end := strings.IndexByte(line[1:], quote)
		if end < 0 {
			return "", false
		}
		rest := strings.TrimLeft(line[end+2:], " \t")
		if !strings.HasPrefix(rest, ":") {
			return "", false
		}
		if after := rest[1:]; after != "" && after[0] != ' ' && after[0] != '\t' {
			return "", false
		}
		return line[1 : end+1], true
	}
	idx := strings.IndexByte(line, ':')
	if idx <= 0 {
		return "", false
	}
	if after := line[idx+1:]; after != "" && after[0] != ' ' && after[0] != '\t' {
		return "", false
	}
	return line[:idx], true
}
