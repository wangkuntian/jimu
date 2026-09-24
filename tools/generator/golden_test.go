package generator

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestGenerateModuleProducesSameTree 是模板外置（templates/** + go:embed）的逐字节回归网：
// testdata/golden/module/** 由**迁移前**（HEAD 98186be 的 CLI）对同一 fixture 生成并入库，
// 当前实现对同一输入生成的能力目录必须与黄金树逐文件相同（既不多也不少）。
func TestGenerateModuleProducesSameTree(t *testing.T) {
	root := newTestRepository(t)
	if err := GenerateModuleAt(root, "product"); err != nil {
		t.Fatal(err)
	}
	goldenRoot := filepath.Join("testdata", "golden", "module")
	generatedRoot := filepath.Join(root, "internal", "capabilities", "product")

	want := readTree(t, goldenRoot)
	require.NotEmpty(t, want, "golden tree is empty")
	got := readTree(t, generatedRoot)

	for rel, wantContent := range want {
		gotContent, ok := got[rel]
		if !ok {
			t.Errorf("missing generated file %s", rel)
			continue
		}
		if gotContent != wantContent {
			t.Errorf("%s differs from golden\ngot:  %q\nwant: %q", rel, gotContent, wantContent)
		}
		delete(got, rel)
	}
	for rel := range got {
		t.Errorf("unexpected generated file %s", rel)
	}
}

func readTree(t *testing.T, root string) map[string]string {
	t.Helper()
	files := map[string]string{}
	err := filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		rel, err := filepath.Rel(root, p)
		if err != nil {
			return err
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		files[filepath.ToSlash(rel)] = string(b)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return files
}
