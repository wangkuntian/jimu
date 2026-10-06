package generator

import "jimu/tools/generator/render"

// Template returns a raw embedded generator template by its relative name.
func Template(name string) (string, error) { return render.Template(name) }

// TemplateNames returns the sorted names of all embedded generator templates.
func TemplateNames() []string { return render.TemplateNames() }

// RenderText executes a template and formats generated Go source.
func RenderText(name, text string, data any) ([]byte, error) {
	return render.RenderText(name, text, data)
}
