package render

import (
	"fmt"
	"os"
	"path/filepath"
	"text/template"

	"jimu/tools/generator/manifest"
)

func renderTemplate(sourceRoot, destinationRoot string, action manifest.TemplateAction) error {
	if action.Kind == "cli" {
		return renderCLI(sourceRoot, destinationRoot, action.Source, action.Destination, action.Data)
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
	return nil
}
