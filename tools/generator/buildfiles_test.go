package generator

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 本文件是 Task 5 的测试：生成项目的三份构建文件（Makefile/Dockerfile/scripts/check_profiles.sh）
// 必须是**单形态**口径（无 PROFILE=、无构建期叠加、无 overlay），且 tools/** 复制必须带上
// patches.go 的定点补丁。

func TestGeneratedMakefileHasNoProfileBranch(t *testing.T) {
	set, err := ParseCapabilitySet("minimal", "", "")
	require.NoError(t, err)
	out, err := RenderMakefile(set)
	require.NoError(t, err)
	src := string(out)
	assert.NotContains(t, src, "PROFILE ?=")
	assert.NotContains(t, src, "profileoverlay")
	assert.NotContains(t, src, "-overlay=")
	assert.Contains(t, src, "go build -ldflags", "普通 go build")
	assert.Contains(t, src, "\ncheck-capabilities:")
	assert.Contains(t, src, "\nprofiles-check:")
	assert.Contains(t, src, "\ncompose-report:")
	assert.NotContains(t, src, "\nswagger:") // minimal 不含 apidocs → swagger 目标一并去掉（设计 §3.8）
	assert.NotContains(t, src, "docs/openapi")
	// 被删掉的目标：它们依赖未复制的脚本/资产（scripts/{smoke,backup,restore}*.sh、
	// docker-compose.yml、deploy/backup、proto/）。
	for _, gone := range []string{"\nsmoke-check:", "\ncompose-check:", "\nbackup:", "\nrestore:",
		"\ngovulncheck:", "\nproto:", "\nbench:", "\nloadtest:", "\nsecrets:", "\ndocker-build:"} {
		assert.NotContains(t, src, gone, "生成版 Makefile 不应保留 %q", gone)
	}
}

func TestGeneratedMakefileKeepsSwaggerWhenApidocsSelected(t *testing.T) {
	set, err := ParseCapabilitySet("full", "", "")
	require.NoError(t, err)
	out, err := RenderMakefile(set)
	require.NoError(t, err)
	assert.Contains(t, string(out), "\nswagger:")
	assert.Contains(t, string(out), "\nswagger-check:")
	assert.Contains(t, string(out), "docs/openapi")
	assert.NotContains(t, string(out), "PROFILE ?=")
}

func TestGeneratedDockerfileHasNoOverlayAndConditionalOpenapi(t *testing.T) {
	min, err := ParseCapabilitySet("minimal", "", "")
	require.NoError(t, err)
	out, err := RenderDockerfile(min)
	require.NoError(t, err)
	assert.NotContains(t, string(out), "ARG PROFILE")
	assert.NotContains(t, string(out), "overlay")
	assert.NotContains(t, string(out), "docs/openapi")

	full, err := ParseCapabilitySet("full", "", "")
	require.NoError(t, err)
	out, err = RenderDockerfile(full)
	require.NoError(t, err)
	assert.Contains(t, string(out), "COPY docs/openapi/ ./docs/openapi/")
}

func TestGeneratedCheckProfilesCoversExactlyOneShape(t *testing.T) {
	set, err := ParseCapabilitySet("minimal", "", "")
	require.NoError(t, err)
	out, err := RenderCheckProfiles(set)
	require.NoError(t, err)
	src := string(out)
	assert.Contains(t, src, `PROFILES=(minimal)`)
	assert.Contains(t, src, `EXPECTED_minimal=`)
	assert.NotContains(t, src, "EXPECTED_full=")
	assert.Contains(t, src, "no capabilities/catalog")
	// 单形态的 golden = 生成项目出货二进制的真实闭包（编译闭包），与本仓
	// scripts/check_profiles.sh 的 EXPECTED_minimal 逐值一致：迁移携带的 tenant 只进
	// catalog（CLI），不进 cmd/server 的闭包。
	assert.Contains(t, src, `EXPECTED_minimal="access auth encryption notification outbox queue user"`)
	// 逐值相等已蕴含「不含非预期能力」：单形态脚本不再持有 FORBIDDEN/ALLOWED 清单
	// （两者按构造不相交，比对恒为 no-op）。
	assert.NotContains(t, src, "FORBIDDEN_")
	assert.NotContains(t, src, "ALLOWED_")
	assert.NotContains(t, src, "subtract_words")
	assert.NotContains(t, src, "ALL_CAPS")
	// 单形态脚本不得再有叠加逻辑。
	assert.NotContains(t, src, "-overlay=")
	assert.NotContains(t, src, "profileoverlay \"$p\"")
}

// TestApplyPatchFileFailsClosedOnDrift 钉住 patches.go 的 fail-closed 语义：原文不存在即报错，
// 绝不静默跳过（文案漂移而没人同步补丁时，生成项目会带着错误口径继续跑）。
func TestApplyPatchFileFailsClosedOnDrift(t *testing.T) {
	dir := t.TempDir()
	ok := filepath.Join(dir, "drivers.go")
	require.NoError(t, os.WriteFile(ok, []byte("package main\n\nconst modulePath = \"jimu\"\n"), 0o644))
	require.NoError(t, applyPatchFile(ok, [][2]string{{`const modulePath = "jimu"`, `const modulePath = "{{module}}"`}}, "example.com/proj"))
	content, err := os.ReadFile(ok)
	require.NoError(t, err)
	assert.Contains(t, string(content), `const modulePath = "example.com/proj"`)

	drifted := filepath.Join(dir, "drifted.go")
	require.NoError(t, os.WriteFile(drifted, []byte("package main\n"), 0o644))
	err = applyPatchFile(drifted, [][2]string{{`const modulePath = "jimu"`, `const modulePath = "{{module}}"`}}, "example.com/proj")
	require.ErrorContains(t, err, "patch")
}

// TestFilePatchesTargetExistingFrameworkText fail-closed 的前提自检：表里每条补丁的**原文**
// 必须真的存在于框架仓对应文件里；否则 applyFilePatches 会在生成时把每个项目都打挂。
func TestFilePatchesTargetExistingFrameworkText(t *testing.T) {
	root := FrameworkRoot()
	require.NotEmpty(t, root)
	require.NotEmpty(t, filePatches)
	for rel, patches := range filePatches {
		content, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
		require.NoError(t, err, "filePatches 指向的文件必须存在：%s", rel)
		for _, p := range patches {
			assert.Contains(t, string(content), p[0], "%s 的补丁原文必须存在（fail-closed 前提）", rel)
		}
	}
}

// TestCopyToolsCoversEveryBlueprintTree 复制集必须覆盖 brief 列出的全部蓝本目录树：
// 生成项目里这些工具要能**在该项目内**工作（单形态），少一类即 check-capabilities/
// profiles-check/compose-report 直接不可用。
func TestCopyToolsCoversEveryBlueprintTree(t *testing.T) {
	assert.Equal(t, []string{
		"tools/internal/profileoverlay",
		"tools/internal/profileassets",
		"tools/internal/heavydeps",
		"tools/profileoverlay",
		"tools/profileassets",
		"tools/checkcapabilities",
		"tools/composereport",
		"tools/logcheck",
	}, copyTools)
	root := FrameworkRoot()
	for _, rel := range copyTools {
		info, err := os.Stat(filepath.Join(root, filepath.FromSlash(rel)))
		require.NoError(t, err, "蓝本目录缺失：%s", rel)
		assert.True(t, info.IsDir(), "%s 必须是目录", rel)
	}
	// tools/generator 是生成器自身，**不得**复制进生成项目。
	assert.NotContains(t, copyTools, "tools/generator")
}

// TestNewProjectCopiesToolsAndRendersBuildFiles 是 Task 5 的落地验收：工具树 + 三份构建文件
// 都在生成项目里，且 modulePath 定点改写已生效、框架侧 tools 测试未被复制。
func TestNewProjectCopiesToolsAndRendersBuildFiles(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "proj")
	_, err := NewProject(NewOptions{Dir: dir, Profile: "minimal", Module: "example.com/proj", NoTidy: true})
	require.NoError(t, err)

	for _, rel := range []string{
		"Makefile", "Dockerfile", "scripts/check_profiles.sh",
		"tools/checkcapabilities/main.go", "tools/checkcapabilities/drivers.go",
		"tools/checkcapabilities/assets.go",
		"tools/composereport/main.go", "tools/logcheck/main.go",
		"tools/profileassets/main.go", "tools/profileoverlay/main.go",
		"tools/internal/profileassets/profileassets.go",
		"tools/internal/profileoverlay/profileoverlay.go",
		"tools/internal/heavydeps/heavydeps.go",
	} {
		assert.FileExists(t, filepath.Join(dir, filepath.FromSlash(rel)), rel)
	}
	assert.NoDirExists(t, filepath.Join(dir, "tools", "generator"), "tools/generator 不得进生成项目")
	// 框架侧 tools 测试断言框架口径（多形态清单/框架 module path/框架资产表），不复制。
	assert.NoFileExists(t, filepath.Join(dir, "tools/checkcapabilities/drivers_test.go"))
	assert.NoFileExists(t, filepath.Join(dir, "tools/internal/profileoverlay/profileoverlay_test.go"))

	drivers, err := os.ReadFile(filepath.Join(dir, "tools/checkcapabilities/drivers.go"))
	require.NoError(t, err)
	assert.Contains(t, string(drivers), `const modulePath = "example.com/proj"`)
	assert.NotContains(t, string(drivers), `const modulePath = "jimu"`)

	logcheck, err := os.ReadFile(filepath.Join(dir, "tools/logcheck/main.go"))
	require.NoError(t, err)
	assert.Contains(t, string(logcheck), `"example.com/proj/tools/logcheck"`)
	assert.NotContains(t, string(logcheck), `"jimu/tools/logcheck"`)

	// profileoverlay 的替换模板是原始字符串里的 import：必须一并定点改写，否则生成项目里
	// profileoverlay 产出仍 import jimu/... 的选点文件（checkcapabilities 的 load 会直接失败）。
	overlay, err := os.ReadFile(filepath.Join(dir, "tools/internal/profileoverlay/profileoverlay.go"))
	require.NoError(t, err)
	assert.Contains(t, string(overlay), `"example.com/proj/internal/assembly"`)
	assert.Contains(t, string(overlay), `"example.com/proj/internal/profiles/%s"`)
	assert.NotContains(t, string(overlay), `"jimu/internal/assembly"`)
	assert.NotContains(t, string(overlay), `"jimu/internal/profiles/%s"`)

	// compose-report 的口径文案已按单形态改写（recon §8 第 4 点）。
	report, err := os.ReadFile(filepath.Join(dir, "tools/composereport/main.go"))
	require.NoError(t, err)
	assert.Contains(t, string(report), "生成本项目唯一形态的「编译面」报告")
	assert.NotContains(t, string(report), "生成各形态（profile）的「编译面」报告")

	info, err := os.Stat(filepath.Join(dir, "scripts/check_profiles.sh"))
	require.NoError(t, err)
	assert.NotZero(t, info.Mode().Perm()&0o100, "check_profiles.sh 必须可执行")

	// 复制/渲染进生成项目的构建文件不得残留框架 module path（Makefile 的产物名按项目名改写）。
	makefile, err := os.ReadFile(filepath.Join(dir, "Makefile"))
	require.NoError(t, err)
	assert.Contains(t, string(makefile), "$(BIN_DIR)/proj-server")
	assert.NotContains(t, string(makefile), "$(BIN_DIR)/jimu-")
}

// TestToolCopyFilterDropsFrameworkTestsOnly 复制过滤器只丢 `_test.go`，其它文件一律放行。
func TestToolCopyFilterDropsFrameworkTestsOnly(t *testing.T) {
	root := FrameworkRoot()
	entries, err := os.ReadDir(filepath.Join(root, "tools", "checkcapabilities"))
	require.NoError(t, err)
	for _, e := range entries {
		want := !strings.HasSuffix(e.Name(), "_test.go")
		assert.Equal(t, want, !skipToolTestFile(e.Name()), "%s 的过滤判定", e.Name())
	}
}

// TestBuildFilesRenderForEveryCapabilitySelection 是「每个选择都能渲染出单形态构建文件」的
// 廉价系统网（不构建）：golden 必须等于该选择的编译闭包，且不得混入迁移携带 / 只带 domain 的能力
// —— 它们要么只进 catalog/CLI，要么只贡献 domain 子包，都不该出现在出货二进制的根包闭包里。
// 新增 MigrationSchemaDeps 边或改闭包算法时，这条会先于 25 次真实构建指出问题。
func TestBuildFilesRenderForEveryCapabilitySelection(t *testing.T) {
	for _, name := range allCapabilityNames(t) {
		t.Run(name, func(t *testing.T) {
			set, err := ParseCapabilitySet("", name, "app")
			require.NoError(t, err)
			scripts, err := RenderCheckProfiles(set)
			require.NoError(t, err)
			src := string(scripts)
			assert.Contains(t, src, "PROFILES=(app)")
			assert.Contains(t, src, `EXPECTED_app="`+strings.Join(expectedRoots(set), " ")+`"`)
			for _, only := range append(append([]string{}, set.MigrationOnly...), set.DomainOnly...) {
				assert.NotContains(t, expectedRoots(set), only,
					"%s 只随迁移/domain 携带，不得进出货二进制的根包闭包", only)
			}
			// apidocs 是唯一带 swagger 的选择（未经 profile 也可选，Important 3）。
			makefile, err := RenderMakefile(set)
			require.NoError(t, err)
			if slices.Contains(set.Copy, "apidocs") {
				assert.Contains(t, string(makefile), "\nswagger:")
			} else {
				assert.NotContains(t, string(makefile), "\nswagger:")
			}
			if _, err := RenderDockerfile(set); err != nil {
				t.Fatalf("render Dockerfile for %s: %v", name, err)
			}
		})
	}
}
