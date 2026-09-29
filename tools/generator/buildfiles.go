package generator

import (
	"slices"
	"strings"
)

// 单形态构建文件渲染（T5）：生成项目的 Makefile / Dockerfile / scripts/check_profiles.sh 都由
// templates/project/*.tmpl 渲染 —— 生成项目的产物里**没有任何形态分支**（形态在生成期固定）：
//
//	Makefile            无 PROFILE=、无 overlay 步骤，build 就是普通的 go build；
//	Dockerfile          无 ARG PROFILE、无 overlay 步骤（镜像/容器/用户名保持框架名，裁定 3）；
//	check_profiles.sh   只覆盖本项目唯一形态（PROFILES=(<shape>) + 该形态的闭包 golden）。
//
// 「无 apidocs 时不要 swagger/swagger-check」与「Dockerfile 不带 docs/openapi」是设计 §3.8
// 「未选中资产不出现」在构建文件上的落点；判定与 assets.go 的资产派生同一口径（含 apidocs 时
// docs/openapi 才是本次复制集里的一员）。

// buildData 是三份构建文件模板的数据。
type buildData struct {
	Shape      string // 本项目唯一形态名
	HasAPIdocs bool   // 选中集是否含 apidocs（catalog 之外、只在 full 清单里的 Ungated 能力）
	Expected   string // 本形态闭包 golden（能力根包，空格分隔）；逐值相等即「不含非预期能力」
}

// RenderMakefile 渲染生成项目的 Makefile。
func RenderMakefile(set CapabilitySet) ([]byte, error) {
	return renderBuildFile("Makefile", set)
}

// RenderDockerfile 渲染生成项目的 Dockerfile。
func RenderDockerfile(set CapabilitySet) ([]byte, error) {
	return renderBuildFile("Dockerfile", set)
}

// RenderCheckProfiles 渲染生成项目的 scripts/check_profiles.sh（单形态版门禁）。
func RenderCheckProfiles(set CapabilitySet) ([]byte, error) {
	return renderBuildFile("scripts/check_profiles.sh", set)
}

// renderBuildFile 取外置模板并渲染；模板名与产物相对路径同名（脚本多一层 scripts/ 前缀）。
func renderBuildFile(rel string, set CapabilitySet) ([]byte, error) {
	name := rel
	if i := strings.LastIndex(name, "/"); i >= 0 {
		name = name[i+1:]
	}
	text, err := Template("project/" + name + ".tmpl")
	if err != nil {
		return nil, err
	}
	return RenderText(name, text, newBuildData(set))
}

// newBuildData 把能力集折算成构建文件模板的数据（纯函数，便于单测）。
func newBuildData(set CapabilitySet) buildData {
	return buildData{
		Shape:      set.Shape,
		HasAPIdocs: slices.Contains(set.Copy, "apidocs"),
		Expected:   strings.Join(expectedRoots(set), " "),
	}
}

// expectedRoots 是生成项目出货二进制的能力**根包**闭包（check_profiles.sh 的 golden），
// 取 CapabilityRoots 算出的 `set.Roots`：Declared 的 **import 可达**根包集合（无子包，与
// scripts/check_profiles.sh 的 cap_roots 同口径），**不做** Requires 传递补齐。
//
// 为什么不能从 Copy 推（T8 裁定 17 的独立 oracle 发现的真实缺陷）：Copy 是**复制集**，含
// Requires 传递补齐 —— `--with=mfa` 的 `auth.Requires=[user,access]` 会把 access 放进 Copy
// （目录确实要复制），但生成项目的 assembly 只 import 声明集（user/mfa），出货二进制里**没有**
// access 根包；用 Copy 推 golden 会让生成项目自己的 `make profiles-check` 红。
// MigrationOnly（如 tenant）与 DomainOnly 天然不在 Roots 里：迁移携带能力只被生成的 catalog
// import（catalog 不进出货二进制），domain 子包不产生根包 import。
//
// 与本仓 scripts/check_profiles.sh 的 EXPECTED_<profile> 逐值一致（由
// TestProfileGoldensMatchTheRepoCheckProfilesScript 交叉钉住）。
func expectedRoots(set CapabilitySet) []string {
	return slices.Clone(set.Roots)
}
