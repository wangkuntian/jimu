package generator

import (
	"os"
	"os/exec"
	"path/filepath"
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
	assert.Contains(t, src, "-w -s", "生成项目默认剥离符号表")
	assert.Contains(t, src, "\ncheck-capabilities:")
	// 单形态项目不需要多形态资产工具：这两个目标依赖已不随生成项目发布的工具树。
	assert.NotContains(t, src, "\nprofiles-check:")
	assert.NotContains(t, src, "\ncompose-report:")
	assert.NotContains(t, src, "\nswagger:") // minimal 不含 apidocs → swagger 目标一并去掉（设计 §3.8）
	assert.NotContains(t, src, "docs/openapi")
	// 被删掉的目标：它们依赖未复制的脚本/资产（scripts/{smoke,backup,restore}*.sh、
	// docker-compose.yml、deploy/backup、proto/）。
	for _, gone := range []string{"\nsmoke-check:", "\ncompose-check:", "\nbackup:", "\nrestore:",
		"\ngovulncheck:", "\nproto:", "\nbench:", "\nloadtest:", "\nsecrets:", "\ndocker-build:"} {
		assert.NotContains(t, src, gone, "生成版 Makefile 不应保留 %q", gone)
	}
}

func TestGeneratedMakefileUsesSharedLifecycleContract(t *testing.T) {
	set, err := ParseCapabilitySet("minimal", "", "")
	require.NoError(t, err)
	out, err := RenderMakefile(set)
	require.NoError(t, err)
	src := string(out)
	assert.NotContains(t, src, "include .env")
	assert.NotContains(t, src, "\nrun:")
	assert.Contains(t, src, "\ndev:")
	assert.Contains(t, src, "\nimage:")
	assert.Contains(t, src, "IMAGE_REPO ?= jimu")
	assert.Contains(t, src, "IMAGE_TAG = $(IMAGE_REPO):$(VERSION)")
	assert.Contains(t, src, "docker tag $(IMAGE_TAG) $(LATEST_TAG)")
	assert.Contains(t, src, "## help:")
	assert.Contains(t, src, "VERSION ?= dev")
	assert.Contains(t, src, "APP_VERSION ?= dev")
	assert.Contains(t, src, "LDFLAGS := -X main.version=$(APP_VERSION)")
	for _, gone := range []string{"\nup:", "\ndown:", "\nup-observability:"} {
		assert.NotContains(t, src, gone)
	}
}

func TestGeneratedMakefileHelpIncludesWrappedDescriptions(t *testing.T) {
	set, err := ParseCapabilitySet("minimal", "", "")
	require.NoError(t, err)
	out, err := RenderMakefile(set)
	require.NoError(t, err)

	dir := t.TempDir()
	makefile := filepath.Join(dir, "Makefile")
	require.NoError(t, os.WriteFile(makefile, out, 0o644))
	cmd := exec.Command("make", "-f", makefile, "help")
	cmd.Dir = dir
	help, err := cmd.CombinedOutput()
	require.NoError(t, err, "make help failed:\n%s", help)
	assert.Contains(t, string(help), "禁动态 key R2 / 字段词汇表 R3 / 禁嵌套对象 R4")
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
		"tools/internal/projectmetrics",
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
	_, err := newProjectForTest(t, NewOptions{Dir: dir, Profile: "minimal", Module: "example.com/proj", NoTidy: true})
	require.NoError(t, err)

	for _, rel := range []string{
		"Makefile", "Dockerfile",
		"tools/checkcapabilities/main.go", "tools/checkcapabilities/drivers.go",
		"tools/checkcapabilities/assets.go",
		"tools/logcheck/main.go",
		"tools/internal/profileassets/profileassets.go",
		"tools/internal/heavydeps/heavydeps.go",
	} {
		assert.FileExists(t, filepath.Join(dir, filepath.FromSlash(rel)), rel)
	}
	// 多形态资产工具不随生成项目发布。
	assert.NoDirExists(t, filepath.Join(dir, "tools", "composereport"))
	assert.NoDirExists(t, filepath.Join(dir, "tools", "profileassets"))
	assert.NoDirExists(t, filepath.Join(dir, "tools", "profileoverlay"))
	assert.NoDirExists(t, filepath.Join(dir, "tools", "internal", "profileoverlay"))
	assert.NoFileExists(t, filepath.Join(dir, "scripts", "check_profiles.sh"))
	assert.NoDirExists(t, filepath.Join(dir, "tools", "generator"), "tools/generator 不得进生成项目")
	// 框架侧 tools 测试断言框架口径（多形态清单/框架 module path/框架资产表），不复制。
	assert.NoFileExists(t, filepath.Join(dir, "tools/checkcapabilities/drivers_test.go"))
	drivers, err := os.ReadFile(filepath.Join(dir, "tools/checkcapabilities/drivers.go"))
	require.NoError(t, err)
	assert.Contains(t, string(drivers), `const modulePath = "example.com/proj"`)
	assert.NotContains(t, string(drivers), `const modulePath = "jimu"`)

	logcheck, err := os.ReadFile(filepath.Join(dir, "tools/logcheck/main.go"))
	require.NoError(t, err)
	assert.Contains(t, string(logcheck), `"example.com/proj/tools/logcheck"`)
	assert.NotContains(t, string(logcheck), `"jimu/tools/logcheck"`)

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
