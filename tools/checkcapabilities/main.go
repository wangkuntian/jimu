// Command checkcapabilities 校验能力自描述与实际迁移一致（P2.8 门禁的第一块）。
// 当前范围：① Owns 的表必须由且仅由该能力的 mysql 迁移 CREATE（表归属唯一）；
// ② 驱动可用集/选中集/import 闭包一致、形态只 import 已声明驱动（见 drivers.go）。
// 完整门禁（跨能力 import 一致性、internal 越界、kernel→capabilities 反向依赖）留待 P2.8。
package main

import (
	"fmt"
	"io/fs"
	"maps"
	"os"
	"regexp"
	"slices"
	"strings"

	"jimu/internal/capabilities/catalog"
	"jimu/internal/contract"
)

// createTableRe 提取 CREATE TABLE 的目标表名；容忍任意空白、可选 TEMPORARY 与
// IF NOT EXISTS，并对表名做大小写归一（to lower）后再比较。
var createTableRe = regexp.MustCompile(`(?i)CREATE\s+(?:TEMPORARY\s+)?TABLE(?:\s+IF\s+NOT\s+EXISTS)?\s+[` + "`" + `"]?([a-z0-9_]+)`)

// checkOwnership 比较「声明拥有的表」与「迁移实际建的表」，返回首个违规说明。
// 遍历顺序按能力名排序，保证同一份清单每次报出的违规确定。
func checkOwnership(declared, created map[string][]string) error {
	owners := map[string]string{} // 表 -> 能力
	for _, capName := range slices.Sorted(maps.Keys(declared)) {
		for _, t := range declared[capName] {
			if prev, dup := owners[t]; dup {
				return fmt.Errorf("table %q owned by both %q and %q", t, prev, capName)
			}
			owners[t] = capName
		}
	}
	for _, capName := range slices.Sorted(maps.Keys(created)) {
		for _, t := range created[capName] {
			owner, ok := owners[t]
			if !ok {
				return fmt.Errorf("table %q created by %q but not declared in its Owns", t, capName)
			}
			if owner != capName {
				return fmt.Errorf("table %q created by %q but owned by %q", t, capName, owner)
			}
		}
	}
	for _, t := range slices.Sorted(maps.Keys(owners)) {
		if !slices.Contains(created[owners[t]], t) {
			return fmt.Errorf("capability %q declares Owns %q but its migrations do not create it", owners[t], t)
		}
	}
	return nil
}

func main() {
	if err := catalog.ValidateDeclarations(); err != nil {
		fmt.Fprintln(os.Stderr, "❌ check-capabilities:", err)
		os.Exit(1)
	}
	declared := map[string][]string{}
	created := map[string][]string{}
	for _, d := range catalog.All() {
		if len(d.Owns) > 0 {
			declared[d.Name] = d.Owns
		}
		tables, err := createdTables(d)
		if err != nil {
			fmt.Fprintln(os.Stderr, "❌ check-capabilities:", err)
			os.Exit(1)
		}
		if len(tables) > 0 {
			created[d.Name] = tables
		}
	}
	if err := checkOwnership(declared, created); err != nil {
		fmt.Fprintln(os.Stderr, "❌ check-capabilities:", err)
		os.Exit(1)
	}
	root, err := os.Getwd()
	if err != nil {
		fmt.Fprintln(os.Stderr, "❌ check-capabilities:", err)
		os.Exit(1)
	}
	if err := checkDrivers(root, profileAssemblies()); err != nil {
		fmt.Fprintln(os.Stderr, "❌ check-capabilities:", err)
		os.Exit(1)
	}
	fmt.Println("✅ check-capabilities: 能力自描述与迁移归属一致")
	fmt.Println("✅ check-capabilities: 驱动可用集/选中集/import 闭包一致")
	fmt.Println("✅ check-capabilities: 形态生产代码只 import 已声明的驱动")
}

// createdTables 从能力嵌入的 mysql 迁移里提取 CREATE TABLE 的表名（去重排序）。
// mysql 与 postgres 迁移集保持同名表，本门禁以 mysql 为准。
func createdTables(d contract.Descriptor) ([]string, error) {
	if d.Migrations == nil {
		return nil, nil
	}
	set := map[string]bool{}
	err := fs.WalkDir(d.Migrations, "migrations/mysql", func(path string, e fs.DirEntry, err error) error {
		if err != nil || e.IsDir() || !strings.HasSuffix(path, ".sql") {
			return err
		}
		b, err := fs.ReadFile(d.Migrations, path)
		if err != nil {
			return err
		}
		for _, m := range createTableRe.FindAllStringSubmatch(string(b), -1) {
			set[strings.ToLower(m[1])] = true
		}
		return nil
	})
	out := slices.Sorted(maps.Keys(set))
	return out, err
}
