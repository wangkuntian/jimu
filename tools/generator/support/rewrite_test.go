package support

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRewriteModuleAppliesTheDocumentedRules(t *testing.T) {
	dir := t.TempDir()
	write := func(rel, content string) {
		t.Helper()
		p := filepath.Join(dir, filepath.FromSlash(rel))
		require.NoError(t, os.MkdirAll(filepath.Dir(p), 0o755))
		require.NoError(t, os.WriteFile(p, []byte(content), 0o644))
	}
	write("go.mod", "module jimu\n\ngo 1.26.6\n")
	write("internal/a/a.go", "package a\n\nimport \"jimu/internal/b\"\n")
	write("Makefile", "SERVER_BIN = $(BIN_DIR)/jimu-server\nCLI_BIN = $(BIN_DIR)/jimu-cli\n")
	write("scripts/x.sh", "grep 'jimu/internal/capabilities' go.mod\n")
	write("tools/composereport/main.go", "package main\n\nconst modulePath = \"jimu\"\n")
	write("Dockerfile", "go build -o jimu ./cmd/cli\nCOPY --from=builder /app/jimu .\nUSER jimu\n")

	changed, err := RewriteModule(dir, "jimu", "example.com/proj")
	require.NoError(t, err)
	assert.Contains(t, changed, "go.mod")

	read := func(rel string) string {
		t.Helper()
		b, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(rel)))
		require.NoError(t, err)
		return string(b)
	}
	assert.Contains(t, read("go.mod"), "module example.com/proj")
	assert.Contains(t, read("internal/a/a.go"), `"example.com/proj/internal/b"`)
	assert.Contains(t, read("Makefile"), "$(BIN_DIR)/proj-server")
	assert.NotContains(t, read("Makefile"), "jimu-server")
	assert.Contains(t, read("scripts/x.sh"), "'example.com/proj/internal/capabilities'")
	assert.Contains(t, read("tools/composereport/main.go"), `const modulePath = "example.com/proj"`)
	// 框架 CLI 二进制名与系统用户名**不改**（第 1 节裁定 3）。
	assert.Contains(t, read("Dockerfile"), "-o jimu ./cmd/cli")
	assert.Contains(t, read("Dockerfile"), "USER jimu")
}

func TestRewriteModuleIsIdempotent(t *testing.T) {
	dir := t.TempDir()
	write := func(rel, content string) {
		t.Helper()
		p := filepath.Join(dir, filepath.FromSlash(rel))
		require.NoError(t, os.MkdirAll(filepath.Dir(p), 0o755))
		require.NoError(t, os.WriteFile(p, []byte(content), 0o644))
	}
	write("go.mod", "module jimu\n\ngo 1.26.6\n")
	write("internal/a/a.go", "package a\n\nimport \"jimu/internal/b\"\n")
	write("Makefile", "SERVER_BIN = $(BIN_DIR)/jimu-server\n")

	if _, err := RewriteModule(dir, "jimu", "example.com/proj"); err != nil {
		t.Fatal(err)
	}
	changed, err := RewriteModule(dir, "jimu", "example.com/proj")
	require.NoError(t, err)
	assert.Empty(t, changed)
}

func TestRewriteModuleRejectsModulePathCollidingWithSourcePrefix(t *testing.T) {
	// 字面量替换在 to 自身仍含 "jimu/" 时不幂等：第二遍会再改写一次（.../jimu），
	// 或单遍即坏（.../jimu/v2 被 "jimu/" 规则二次插入前缀）。必须在写盘前 fail-closed。
	for _, module := range []string{"github.com/foo/jimu/v2", "github.com/foo/jimu", "github.com/foo/jimu/app"} {
		t.Run(module, func(t *testing.T) {
			dir := t.TempDir()
			original := "module jimu\n\ngo 1.26.6\n"
			goMod := filepath.Join(dir, "go.mod")
			require.NoError(t, os.WriteFile(goMod, []byte(original), 0o644))
			source := filepath.Join(dir, "internal", "a")
			require.NoError(t, os.MkdirAll(source, 0o755))
			require.NoError(t, os.WriteFile(filepath.Join(source, "a.go"), []byte("package a\n\nimport \"jimu/internal/b\"\n"), 0o644))

			_, err := RewriteModule(dir, "jimu", module)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "collides with source prefix")

			// 报错发生在写盘前：go.mod 与 .go 文件都保持原样。
			got, err := os.ReadFile(goMod)
			require.NoError(t, err)
			assert.Equal(t, original, string(got))
			got, err = os.ReadFile(filepath.Join(source, "a.go"))
			require.NoError(t, err)
			assert.Contains(t, string(got), `"jimu/internal/b"`)
		})
	}

	// 源前缀本身仍放行：探测串就是自身，不动点成立。
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module jimu\n\ngo 1.26.6\n"), 0o644))
	changed, err := RewriteModule(dir, "jimu", "jimu")
	require.NoError(t, err)
	assert.Empty(t, changed)
}

func TestRewriteModuleHandlesGoModLineEndings(t *testing.T) {
	// 规则表的 "module jimu\n" 字面量只覆盖 LF + 尾换行 + 无注释；CRLF、文件末尾无换行、
	// 尾空白、行尾注释（注释要保留）都必须改写，不能静默漏改（fail-open）。
	for name, tc := range map[string]struct {
		original string
		want     string
	}{
		"crlf":           {"module jimu\r\n\r\ngo 1.26.6\r\n", "module example.com/proj\r\n"},
		"no final eol":   {"module jimu", "module example.com/proj"},
		"lf":             {"module jimu\n\ngo 1.26.6\n", "module example.com/proj\n"},
		"module padded":  {"module   jimu\n\ngo 1.26.6\n", "module example.com/proj\n"},
		"trailing space": {"module jimu ", "module example.com/proj"},
		"trailing comment": {"module jimu // x\n\ngo 1.26.6\n",
			"module example.com/proj // x\n"},
	} {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			goMod := filepath.Join(dir, "go.mod")
			require.NoError(t, os.WriteFile(goMod, []byte(tc.original), 0o644))

			changed, err := RewriteModule(dir, "jimu", "example.com/proj")
			require.NoError(t, err)
			assert.Contains(t, changed, "go.mod")

			got, err := os.ReadFile(goMod)
			require.NoError(t, err)
			assert.Contains(t, string(got), tc.want)
			assert.NotContains(t, string(got), "module jimu")
		})
	}
}

func TestRewriteModuleFailsClosedWhenGoModCannotBeRewritten(t *testing.T) {
	// 反例：CR-only 行尾（`module jimu\rgo 1.26`）既不被规则表命中，也不被兜底正则命中；
	// module 指令的值仍是 "jimu" → 必须报错，而不是静默产出 module jimu 的项目。
	dir := t.TempDir()
	goMod := filepath.Join(dir, "go.mod")
	original := "module jimu\rgo 1.26\n"
	require.NoError(t, os.WriteFile(goMod, []byte(original), 0o644))

	_, err := RewriteModule(dir, "jimu", "example.com/proj")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "module directive still")

	got, err := os.ReadFile(goMod)
	require.NoError(t, err)
	assert.Equal(t, original, string(got))
}

func TestRewriteModuleAcceptsJimuLikeArtifactNames(t *testing.T) {
	// 裁定（Fix round 3）：合法的 to 一律放行，包括产物名以 "jimu-" 开头或含 "jimu" 子串的模块路径。
	// 单遍产物必须正确 —— 名为 jimu-app 的项目，产物名就该是 jimu-app-server。
	for _, tc := range []struct {
		module       string
		wantMakefile string
	}{
		{"example.com/jimu-app", "SERVER_BIN = $(BIN_DIR)/jimu-app-server\n"},
		{"github.com/foo/jimux", "SERVER_BIN = $(BIN_DIR)/jimux-server\n"},
		{"github.com/foo/my-jimu-app", "SERVER_BIN = $(BIN_DIR)/my-jimu-app-server\n"},
	} {
		t.Run(tc.module, func(t *testing.T) {
			dir := t.TempDir()
			write := func(rel, content string) {
				t.Helper()
				p := filepath.Join(dir, filepath.FromSlash(rel))
				require.NoError(t, os.MkdirAll(filepath.Dir(p), 0o755))
				require.NoError(t, os.WriteFile(p, []byte(content), 0o644))
			}
			write("go.mod", "module jimu\n\ngo 1.26.6\n")
			write("internal/a/a.go", "package a\n\nimport \"jimu/internal/b\"\n")
			write("Makefile", "SERVER_BIN = $(BIN_DIR)/jimu-server\n")
			// 框架自己的名字不改（第 1 节裁定 3）。
			write("Dockerfile", "go build -o jimu ./cmd/cli\nUSER jimu\n")

			changed, err := RewriteModule(dir, "jimu", tc.module)
			require.NoError(t, err)
			assert.Contains(t, changed, "go.mod")

			read := func(rel string) string {
				t.Helper()
				b, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(rel)))
				require.NoError(t, err)
				return string(b)
			}
			assert.Contains(t, read("go.mod"), "module "+tc.module)
			assert.Contains(t, read("internal/a/a.go"), `"`+tc.module+`/internal/b"`)
			assert.Equal(t, tc.wantMakefile, read("Makefile"))
			assert.Contains(t, read("Dockerfile"), "-o jimu ./cmd/cli")
			assert.Contains(t, read("Dockerfile"), "USER jimu")
		})
	}
}

func TestRewriteModuleSecondPassKeepsModuleAndImportsStable(t *testing.T) {
	// 精确幂等契约（见 rewrite.go 的 rewriteRules/RewriteModule 注释）：
	//   (a) 被放行的 to ⇒ go.mod 的 module 指令与 .go import 前缀替换**幂等**（第二遍不再改动它们）；
	//   (b) `jimu-*` 产物名规则**不保证**对同一棵已重写树幂等：产物名自身以 "jimu-" 开头时，
	//       第二遍 `$(BIN_DIR)/jimu-` 规则会再次命中（只在 Makefile 这类产物名上漂移）。
	// 这里把 (b) 的漂移显式断言出来，避免它被误当成契约；T2/T8 只在「从框架仓新复制出来的树」上
	// 调用一次 RewriteModule，所以这条限制不影响真实生成路径。
	for _, tc := range []struct {
		module         string
		wantMakefile   string   // 第一遍之后
		wantSecondPass []string // 第二遍 changed（产物名漂移时非空）
	}{
		{"example.com/proj", "SERVER_BIN = $(BIN_DIR)/proj-server\n", nil},
		{"example.com/my-app", "SERVER_BIN = $(BIN_DIR)/my-app-server\n", nil},
		{"jimu", "SERVER_BIN = $(BIN_DIR)/jimu-server\n", nil},
		{"example.com/jimu-app", "SERVER_BIN = $(BIN_DIR)/jimu-app-server\n", []string{"Makefile"}},
	} {
		t.Run(tc.module, func(t *testing.T) {
			dir := t.TempDir()
			write := func(rel, content string) {
				t.Helper()
				p := filepath.Join(dir, filepath.FromSlash(rel))
				require.NoError(t, os.MkdirAll(filepath.Dir(p), 0o755))
				require.NoError(t, os.WriteFile(p, []byte(content), 0o644))
			}
			write("go.mod", "module jimu\n\ngo 1.26.6\n")
			write("internal/a/a.go", "package a\n\nimport \"jimu/internal/b\"\n")
			write("Makefile", "SERVER_BIN = $(BIN_DIR)/jimu-server\n")
			write("Dockerfile", "go build -o jimu ./cmd/cli\nUSER jimu\n")

			read := func(rel string) string {
				t.Helper()
				b, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(rel)))
				require.NoError(t, err)
				return string(b)
			}
			if _, err := RewriteModule(dir, "jimu", tc.module); err != nil {
				t.Fatal(err)
			}
			assert.Equal(t, tc.wantMakefile, read("Makefile"))
			goModFirst, goFileFirst := read("go.mod"), read("internal/a/a.go")

			second, err := RewriteModule(dir, "jimu", tc.module)
			require.NoError(t, err)
			if tc.wantSecondPass == nil {
				assert.Empty(t, second)
			} else {
				assert.Equal(t, tc.wantSecondPass, second)
			}
			// 契约 (a)：go.mod 与 .go 的内容在两遍之间稳定。
			assert.Equal(t, goModFirst, read("go.mod"))
			assert.Equal(t, goFileFirst, read("internal/a/a.go"))
			// 框架自己的名字在两遍之后仍不改。
			assert.Contains(t, read("Dockerfile"), "-o jimu ./cmd/cli")
		})
	}
}
