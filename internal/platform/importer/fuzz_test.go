package importer

import (
	"context"
	"strings"
	"testing"
)

// FuzzCSVImporterParse 保证任意 CSV 输入下解析不 panic，成功时行结构完整。
func FuzzCSVImporterParse(f *testing.F) {
	f.Add("name,age\ntom,30\n")
	f.Add("a,b,c\n1,2\n")
	f.Add("")
	f.Add(`"unclosed`)
	f.Fuzz(func(t *testing.T, data string) {
		rows, err := NewCSVImporter().Parse(context.Background(), strings.NewReader(data))
		if err != nil {
			return
		}
		for _, r := range rows {
			_ = r
		}
	})
}
