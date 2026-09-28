package generator

import (
	"bytes"
	"fmt"
	"go/format"
	"io/fs"
	"sort"
	"strings"
	"text/template"
)

// Template 取外置模板的原始文本（name 相对 templates/，如 "module.go.tmpl"）。
func Template(name string) (string, error) {
	content, err := fs.ReadFile(templateFS, templateRoot+"/"+name)
	if err != nil {
		return "", fmt.Errorf("template %q: %w", name, err)
	}
	return string(content), nil
}

// TemplateNames 返回全部模板名（排序）；`TestTemplateNamesCoversEveryEmbeddedFile` 与
// `make check-templates` 都靠它枚举，避免模板文件加了却没人用。
func TemplateNames() []string {
	var names []string
	// embed.FS 是编译期快照，模式由编译器校验：这里的错误分支运行期不可能触发，
	// 而接口按约定只返回名字列表。
	_ = fs.WalkDir(templateFS, templateRoot, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		names = append(names, strings.TrimPrefix(p, templateRoot+"/"))
		return nil
	})
	sort.Strings(names)
	return names
}

// RenderText 渲染一个模板：text/template 执行 →（name 以 ".go" 或 ".go.tmpl" 结尾时）gofmt。
// 模板外置取代了 templates.go 的内嵌字符串常量：模板改动不再需要改 Go 代码，也不再与
// 真实能力结构无门禁地漂移（recon §9 风险 3）。
func RenderText(name, text string, data any) ([]byte, error) {
	// 空模板特例：migrations/postgres/.gitkeep 这类占位文件没有模板内容。
	if name == "" && text == "" {
		return []byte{}, nil
	}
	t, err := template.New(name).Parse(text)
	if err != nil {
		return nil, fmt.Errorf("parse template %q: %w", name, err)
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("execute template %q: %w", name, err)
	}
	content := buf.Bytes()
	if strings.HasSuffix(name, ".go") || strings.HasSuffix(name, ".go.tmpl") {
		formatted, err := format.Source(content)
		if err != nil {
			return nil, fmt.Errorf("format rendered template %q: %w", name, err)
		}
		content = formatted
	}
	return content, nil
}
