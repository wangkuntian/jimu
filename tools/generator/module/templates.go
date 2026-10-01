package module

import (
	"bytes"
	"embed"
	"fmt"
	"go/format"
	"io/fs"
	"strings"
	"text/template"
)

//go:embed templates
var templateFS embed.FS

func templateText(name string) (string, error) {
	content, err := fs.ReadFile(templateFS, "templates/"+name)
	if err != nil {
		return "", fmt.Errorf("template %q: %w", name, err)
	}
	return string(content), nil
}

func renderText(name, source string, data any) ([]byte, error) {
	if name == "" && source == "" {
		return []byte{}, nil
	}
	tmpl, err := template.New(name).Parse(source)
	if err != nil {
		return nil, fmt.Errorf("parse template %q: %w", name, err)
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
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
