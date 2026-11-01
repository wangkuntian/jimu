package frameworkmanifest

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/mod/modfile"

	"jimu/tools/generator/manifest"
)

type basicAPITemplateData struct {
	Module            string
	GoVersion         string
	GinVersion        string
	ViperVersion      string
	ZapVersion        string
	ValidatorVersion  string
	LumberjackVersion string
}

func basicAPIDocument(root, module, template string) (manifest.Document, error) {
	if template != "basic-api" {
		return manifest.Document{}, fmt.Errorf("unknown template %q (available: basic-api)", template)
	}
	framework, err := frameworkProvenance(root)
	if err != nil {
		return manifest.Document{}, err
	}
	data, err := loadBasicAPITemplateData(root, module)
	if err != nil {
		return manifest.Document{}, err
	}
	templateData, err := templateDataMap(data)
	if err != nil {
		return manifest.Document{}, err
	}
	templates := basicAPITemplates(templateData)
	files := []string{"go.sum"}
	for _, action := range templates {
		files = append(files, action.Destination)
	}
	return manifest.Document{
		SchemaVersion: manifest.CurrentSchemaVersion,
		Framework:     framework,
		Selection: manifest.Selection{
			Shape: "basic-api", Template: "basic-api", TemplateVersion: manifest.TemplateContractVersion, Support: "official",
			Capabilities: []string{}, Drivers: map[string][]string{},
		},
		Capabilities: []manifest.Capability{},
		Copy: []manifest.CopyAction{
			{Source: "go.sum", Destination: "go.sum", Include: []string{}, Exclude: []string{}},
		},
		Templates:      templates,
		Merges:         []manifest.MergeAction{},
		Rewrites:       []manifest.RewriteAction{},
		Assets:         []manifest.AssetAction{},
		Prune:          []manifest.PruneRule{},
		Report:         manifest.ReportSpec{Name: "basic-api", Capabilities: []string{}, Routes: 1},
		GeneratedFiles: files,
	}, nil
}

func loadBasicAPITemplateData(root, module string) (basicAPITemplateData, error) {
	content, err := os.ReadFile(filepath.Join(root, "go.mod"))
	if err != nil {
		return basicAPITemplateData{}, fmt.Errorf("read framework go.mod: %w", err)
	}
	file, err := modfile.Parse("go.mod", content, nil)
	if err != nil {
		return basicAPITemplateData{}, fmt.Errorf("parse framework go.mod: %w", err)
	}
	versions := make(map[string]string, len(file.Require))
	for _, requirement := range file.Require {
		versions[requirement.Mod.Path] = requirement.Mod.Version
	}
	data := basicAPITemplateData{Module: module}
	if file.Go != nil {
		data.GoVersion = file.Go.Version
	}
	lookup := func(path string) (string, error) {
		version, ok := versions[path]
		if !ok || version == "" {
			return "", fmt.Errorf("framework go.mod does not require %s", path)
		}
		return version, nil
	}
	assignments := []struct {
		path string
		dst  *string
	}{
		{"github.com/gin-gonic/gin", &data.GinVersion},
		{"github.com/spf13/viper", &data.ViperVersion},
		{"go.uber.org/zap", &data.ZapVersion},
		{"github.com/go-playground/validator/v10", &data.ValidatorVersion},
		{"gopkg.in/natefinch/lumberjack.v2", &data.LumberjackVersion},
	}
	for _, assignment := range assignments {
		version, err := lookup(assignment.path)
		if err != nil {
			return basicAPITemplateData{}, err
		}
		*assignment.dst = version
	}
	return data, nil
}

func basicAPITemplates(data map[string]any) []manifest.TemplateAction {
	files := []struct {
		source      string
		destination string
	}{
		{"go.mod.tmpl", "go.mod"},
		{"configs/app.yaml.tmpl", "configs/app.yaml"},
		{"README.md.tmpl", "README.md"},
		{"Makefile.tmpl", "Makefile"},
		{"Dockerfile.tmpl", "Dockerfile"},
		{".gitignore.tmpl", ".gitignore"},
		{".golangci.yml.tmpl", ".golangci.yml"},
		{"cmd/server/main.go.tmpl", "cmd/server/main.go"},
		{"internal/app/app.go.tmpl", "internal/app/app.go"},
		{"internal/app/router.go.tmpl", "internal/app/router.go"},
		{"internal/app/router_test.go.tmpl", "internal/app/router_test.go"},
		{"internal/config/config.go.tmpl", "internal/config/config.go"},
		{"internal/api/v1/routes.go.tmpl", "internal/api/v1/routes.go"},
		{"internal/api/v1/ping.go.tmpl", "internal/api/v1/ping.go"},
		{"internal/api/v1/ping_test.go.tmpl", "internal/api/v1/ping_test.go"},
		{"internal/platform/http/server.go.tmpl", "internal/platform/http/server.go"},
		{"internal/platform/http/middleware.go.tmpl", "internal/platform/http/middleware.go"},
		{"internal/platform/health/health.go.tmpl", "internal/platform/health/health.go"},
		{"internal/platform/logging/logger.go.tmpl", "internal/platform/logging/logger.go"},
		{"internal/shared/errorx/error.go.tmpl", "internal/shared/errorx/error.go"},
		{"internal/shared/response/response.go.tmpl", "internal/shared/response/response.go"},
	}
	actions := make([]manifest.TemplateAction, 0, len(files))
	for _, file := range files {
		actions = append(actions, manifest.TemplateAction{
			Source:      "tools/generator/render/templates/basicapi/" + file.source,
			Destination: file.destination,
			Kind:        "basic-api",
			Data:        data,
		})
	}
	return actions
}

func templateDataMap(value any) (map[string]any, error) {
	content, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("encode template data: %w", err)
	}
	var out map[string]any
	if err := json.Unmarshal(content, &out); err != nil {
		return nil, fmt.Errorf("decode template data: %w", err)
	}
	return out, nil
}
