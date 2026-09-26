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
// 「未选中资产不出现」在构建文件上的落点；判定与 RenderDocs 同一口径（set.Copy 含 apidocs）。

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

// expectedRoots 是生成项目出货二进制的能力**根包**闭包（check_profiles.sh 的 golden）：
// 编译闭包 —— 即 Copy ∖ MigrationOnly ∖ DomainOnly。
//
//	MigrationOnly：迁移携带能力（如 tenant）的根包只被**生成的 catalog** import，而 catalog
//	               正是「生产包不得 import catalog」这条断言要挡住的东西（cmd/server 的闭包
//	               里没有它）—— 它们随 `jimu migrate`/`seed` 进 CLI，不进出货二进制；
//	DomainOnly：  只带 domain/ 子包的内核编译期依赖，不产生对能力根包的 import。
//
// 因此 golden 与本仓 scripts/check_profiles.sh 的 EXPECTED_<profile> 逐值一致（实测 minimal 的
// 两处都是 access auth encryption notification outbox queue user）。
func expectedRoots(set CapabilitySet) []string {
	out := make([]string, 0, len(set.Copy))
	for _, name := range set.Copy {
		if slices.Contains(set.DomainOnly, name) || slices.Contains(set.MigrationOnly, name) {
			continue
		}
		out = append(out, name)
	}
	slices.Sort(out)
	return out
}
