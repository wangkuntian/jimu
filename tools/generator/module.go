package generator

import (
	"fmt"
	"os"
	"path/filepath"
	"text/template"
)

const moduleTemplate = `package {{.Name}}

import (
	"jimu/internal/contract"
)

type Module struct {}

func New() *Module {
	return &Module{}
}

func (m *Module) Name() string {
	return "{{.Name}}"
}

func (m *Module) RegisterHTTP(r contract.Router) {
	// TODO: register routes
}

func (m *Module) RegisterJobs(j contract.JobRegistry) {}

func (m *Module) RegisterEvents(e contract.EventBus) {}
`

func GenerateModule(name string) error {
	dir := filepath.Join("internal", "modules", name)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	tmpl, err := template.New("module").Parse(moduleTemplate)
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}

	f, err := os.Create(filepath.Join(dir, "module.go"))
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer f.Close()

	if err := tmpl.Execute(f, map[string]string{"Name": name}); err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	fmt.Printf("Module '%s' created at %s\n", name, dir)
	return nil
}
