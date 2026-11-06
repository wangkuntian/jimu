package generator

import (
	"net"
	"os"
	"path/filepath"
	"testing"

	"jimu/tools/generator/manifest"

	"github.com/stretchr/testify/require"
)

// withProtectedProvider returns a selectable capability expression that satisfies the
// preflight rule for every protected capability under test.
func withProtectedProvider(name string) string {
	switch name {
	case "auth", "apikey", "apidocs", "captcha", "oauth":
		return name
	default:
		return "apikey," + name
	}
}

func newTestRepository(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "internal/capabilities"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module testrepo\n\ngo 1.26\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return root
}

// writeManifestWithTemplate 造一份"由已退役模板生成"的最小 manifest，
// 用于验证 upgrade/capability add 对旧模板的拒绝路径（旧模板的渲染代码已删除，
// 不能再通过 NewProject 生成）。
func writeManifestWithTemplate(t *testing.T, template string) string {
	t.Helper()
	dir := t.TempDir()
	doc := manifest.Document{
		SchemaVersion: manifest.CurrentSchemaVersion,
		Framework:     manifest.Framework{Module: "jimu", Version: "v0.9.0", Commit: "deadbeef"},
		Selection: manifest.Selection{
			Shape:           template,
			Template:        template,
			TemplateVersion: manifest.TemplateContractVersion,
			Support:         "official",
			Capabilities:    []string{"tenant"},
		},
		Composition: &manifest.Composition{
			ContractVersion: manifest.CompositionContractVersion,
			Requested:       []string{"tenant"},
			Resolved:        []string{"tenant"},
			AutoAdded:       []string{},
			Drivers:         map[string][]string{},
			Validity:        manifest.CompositionValidityValid,
			Support:         manifest.CompositionSupportCertified,
		},
		Rewrites: []manifest.RewriteAction{{Kind: "module", From: "jimu", To: "example.com/legacy", Files: []string{}}},
	}
	if err := os.MkdirAll(filepath.Join(dir, ".jimu"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := manifest.Write(filepath.Join(dir, ".jimu", "manifest.json"), doc); err != nil {
		t.Fatal(err)
	}
	return dir
}

// freePort 返回一个当前空闲的本地端口（关闭监听后可能被抢占，测试内即用）。
func freePort(t *testing.T) int {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer listener.Close()
	return listener.Addr().(*net.TCPAddr).Port
}
