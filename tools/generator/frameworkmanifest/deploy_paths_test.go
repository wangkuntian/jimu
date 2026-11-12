package frameworkmanifest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDeploymentUsesGroupedPaths(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	files := []string{
		"Makefile",
		"scripts/dev.sh",
		"scripts/ensure-image.sh",
		"deploy/compose/docker-compose.yml",
		"deploy/compose/compose.observability.yml",
		"deploy/scripts/observability.sh",
		"deploy/scripts/init-secrets.sh",
		"tools/generator/render/templates/project/Makefile.tmpl",
	}
	forbidden := []string{
		"-f docker-compose.yml",
		"-f compose.observability.yml",
		"./scripts/init-secrets.sh",
		"deploy/openobserve",
		"deploy/otel-collector.yaml",
		"dockerfile: Dockerfile",
	}
	for _, rel := range files {
		content, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
		if err != nil {
			t.Fatalf("read %s: %v", rel, err)
		}
		text := string(content)
		for _, value := range forbidden {
			if strings.Contains(text, value) {
				t.Errorf("%s still references %q", rel, value)
			}
		}
	}

	compose, err := os.ReadFile(filepath.Join(root, "deploy", "compose", "docker-compose.yml"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"dockerfile: deploy/docker/Dockerfile", "./deploy/scripts"} {
		if !strings.Contains(string(compose), want) {
			t.Errorf("deploy/compose/docker-compose.yml missing %q", want)
		}
	}
}
