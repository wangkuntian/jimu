package render

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"jimu/tools/generator/manifest"
)

func renderTemplate(sourceRoot, destinationRoot string, action manifest.TemplateAction) error {
	if action.Kind == "cli" {
		return renderCLI(sourceRoot, destinationRoot, action.Source, action.Destination, action.Data)
	}
	if action.Kind == "sections" {
		return renderSectionTemplate(sourceRoot, destinationRoot, action)
	}
	source, err := safeJoin(sourceRoot, action.Source)
	if err != nil {
		return err
	}
	destination, err := safeJoin(destinationRoot, action.Destination)
	if err != nil {
		return err
	}
	content, err := os.ReadFile(source)
	if err != nil {
		return fmt.Errorf("read template %s: %w", action.Source, err)
	}
	tmpl, err := template.New(action.Source).Parse(string(content))
	if err != nil {
		return fmt.Errorf("parse template %s: %w", action.Source, err)
	}
	if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
		return err
	}
	file, err := os.Create(destination)
	if err != nil {
		return fmt.Errorf("create template destination %s: %w", action.Destination, err)
	}
	defer func() { _ = file.Close() }()
	if err := tmpl.Execute(file, action.Data); err != nil {
		return fmt.Errorf("execute template %s: %w", action.Source, err)
	}
	mode := os.FileMode(0o644)
	if strings.HasSuffix(action.Destination, ".sh") {
		mode = 0o755
	}
	if err := file.Chmod(mode); err != nil {
		return fmt.Errorf("chmod template destination %s: %w", action.Destination, err)
	}
	return nil
}
