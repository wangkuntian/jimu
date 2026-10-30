package generator

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"jimu/tools/generator/manifest"
	"jimu/tools/generator/workspace"
	"jimu/tools/internal/projectmetrics"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/mod/modfile"
)

type templateBudgets struct {
	SchemaVersion int `json:"schema_version"`
	Templates     map[string]struct {
		Limits           map[string]int `json:"limits"`
		ReadinessSeconds map[string]int `json:"readiness_seconds"`
	} `json:"templates"`
}

func TestOfficialTemplateBudgets(t *testing.T) {
	budgets := loadTemplateBudgets(t)
	assert.Equal(t, 1, budgets.SchemaVersion)

	for _, template := range []string{"basic-api", "auth-api"} {
		t.Run(template, func(t *testing.T) {
			dir := filepath.Join(t.TempDir(), template)
			result, err := newProjectForTest(t, NewOptions{
				Dir: dir, Template: template, Module: "example.com/" + template,
			})
			require.NoError(t, err)

			document, err := workspace.LoadProjectManifest(dir)
			require.NoError(t, err)
			files, lines, _, err := projectmetrics.ClosureSize(dir, "example.com/"+template, nil)
			require.NoError(t, err)

			metrics := map[string]int{
				"generated_files":     result.FileCount - 1,
				"direct_dependencies": directDependencyCount(t, dir),
				"go_closure_files":    files,
				"go_closure_lines":    lines,
				"external_services":   externalServiceCount(document),
				"business_routes":     document.Report.Routes,
				"health_routes":       healthRouteCount(t, dir),
				"tables":              len(document.Report.Tables),
				"migrations":          len(document.Report.Migrations),
				"non_code_assets":     len(document.Assets),
			}
			limits := budgets.Templates[template].Limits

			actualKeys := sortedKeys(metrics)
			limitKeys := sortedKeys(limits)
			assert.Equal(t, actualKeys, limitKeys, "budgets.json 与门禁必须使用同一组指标")
			for _, name := range actualKeys {
				t.Logf("%s: %d / %d", name, metrics[name], limits[name])
				assert.LessOrEqual(t, metrics[name], limits[name], "%s 超过发布阻断预算", name)
			}
		})
	}
}

func TestAuthAPIEmptyDatabaseEndToEnd(t *testing.T) {
	requireHeavyMatrix(t)
	requireDockerDaemon(t)

	started := time.Now()
	cache := newTestGoCache(t)
	dir := filepath.Join(t.TempDir(), "auth")
	_, err := newProjectForTest(t, NewOptions{
		Dir: dir, Template: "auth-api", Module: "example.com/auth",
	})
	require.NoError(t, err)

	project := fmt.Sprintf("jimu-auth-e2e-%d", time.Now().UnixNano())
	publicPort := freePort(t)
	managementPort := freePort(t)
	for managementPort == publicPort {
		managementPort = freePort(t)
	}
	env := dockerTestEnv(cache,
		"COMPOSE_PROJECT_NAME="+project,
		fmt.Sprintf("AUTH_API_PORT=%d", publicPort),
		fmt.Sprintf("AUTH_API_MANAGEMENT_PORT=%d", managementPort),
	)
	_ = runMakeForTemplate(t, dir, env, "test")
	_ = runMakeForTemplate(t, dir, env, "vet")
	t.Cleanup(func() {
		output, err := dockerCompose(t, dir, env, project, "down", "-v", "--remove-orphans")
		if err != nil {
			t.Logf("cleanup auth-api stack: %v\n%s", err, output)
		}
	})

	firstUp := runMakeForTemplate(t, dir, env, "up")
	require.Equal(t, 1, strings.Count(firstUp, "Administrator password:"),
		"first make up must print the generated administrator password exactly once:\n%s", firstUp)
	budgets := loadTemplateBudgets(t)
	limit := budgets.Templates["auth-api"].ReadinessSeconds["cold_images"]
	require.Positive(t, limit)
	require.LessOrEqual(t, time.Since(started), time.Duration(limit)*time.Second,
		"auth-api cold generate/up exceeded the %ds contract budget", limit)

	firstVerify := runMakeForTemplate(t, dir, env, "verify")
	require.Contains(t, firstVerify, "auth-api verification passed")

	for _, tc := range []struct {
		name  string
		env   []string
		error string
	}{
		{"mysql", []string{"DB_PORT=1"}, "connect mysql"},
		{"redis", []string{"REDIS_ADDR=redis:1"}, "ping redis"},
	} {
		t.Run("missing-"+tc.name, func(t *testing.T) {
			args := []string{"run", "--rm", "--no-deps"}
			for _, value := range tc.env {
				args = append(args, "-e", value)
			}
			args = append(args, "app")
			output, err := dockerCompose(t, dir, env, project, args...)
			require.Error(t, err, "auth-api must fail when %s is unavailable:\n%s", tc.name, output)
			require.Contains(t, output, tc.error)
		})
	}

	secondUp := runMakeForTemplate(t, dir, env, "up")
	require.Zero(t, strings.Count(secondUp, "Administrator password:"),
		"repeated make up must not print or regenerate the administrator password:\n%s", secondUp)
	secondVerify := runMakeForTemplate(t, dir, env, "verify")
	require.Contains(t, secondVerify, "auth-api verification passed")

	output := dockerComposeForTemplate(t, dir, env, project,
		"exec", "-T", "-e", "MYSQL_PWD=jimu", "mysql",
		"mysql", "--protocol=TCP", "-h127.0.0.1", "-ujimu", "-N", "-B", "jimu", "-e",
		`SELECT 'migrations', COUNT(*) FROM schema_migrations
		 UNION ALL SELECT 'tables', COUNT(*) FROM information_schema.tables
		   WHERE table_schema = DATABASE() AND table_name <> 'schema_migrations'
		 UNION ALL SELECT 'users', COUNT(*) FROM users
		 UNION ALL SELECT 'admins', COUNT(*) FROM users WHERE username = 'admin'`,
	)
	counts := parseSQLCounts(t, output)
	assert.Equal(t, 10, counts["migrations"])
	assert.Equal(t, 8, counts["tables"])
	assert.Equal(t, 2, counts["users"])
	assert.Equal(t, 1, counts["admins"])

	_ = runMakeForTemplate(t, dir, env, "down")
}

func requireDockerDaemon(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("docker"); err != nil {
		t.Fatalf("auth-api 端到端门禁需要 Docker Compose: %v", err)
	}
	command := exec.Command("docker", "info", "--format", "{{.ServerVersion}}")
	command.Env = dockerTestEnv("")
	output, err := command.CombinedOutput()
	require.NoError(t, err, "auth-api 端到端门禁需要可用 Docker daemon:\n%s", output)
}

func dockerTestEnv(cache string, extra ...string) []string {
	env := serverEnvBaseForTest(cache)
	for _, key := range []string{
		"DOCKER_HOST", "DOCKER_CONTEXT", "DOCKER_CONFIG", "XDG_CONFIG_HOME", "TMPDIR",
		"USER", "LANG", "LC_ALL", "HTTP_PROXY", "HTTPS_PROXY", "NO_PROXY", "ALL_PROXY",
	} {
		if value := os.Getenv(key); value != "" {
			env = append(env, key+"="+value)
		}
	}
	return append(env, extra...)
}

func runMakeForTemplate(t *testing.T, dir string, env []string, target string) string {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Minute)
	defer cancel()
	command := exec.CommandContext(ctx, "make", target)
	command.Dir = dir
	command.Env = env
	output, err := command.CombinedOutput()
	require.NoError(t, err, "make %s failed:\n%s", target, output)
	return string(output)
}

func dockerComposeForTemplate(t *testing.T, dir string, env []string, project string, args ...string) string {
	t.Helper()
	output, err := dockerCompose(t, dir, env, project, args...)
	require.NoError(t, err, "docker compose %s failed:\n%s", strings.Join(args, " "), output)
	return output
}

func dockerCompose(t *testing.T, dir string, env []string, project string, args ...string) (string, error) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	commandArgs := append([]string{"compose", "-p", project}, args...)
	command := exec.CommandContext(ctx, "docker", commandArgs...)
	command.Dir = dir
	command.Env = env
	output, err := command.CombinedOutput()
	return string(output), err
}

func parseSQLCounts(t *testing.T, output string) map[string]int {
	t.Helper()
	counts := map[string]int{}
	for _, line := range strings.Split(strings.TrimSpace(output), "\n") {
		fields := strings.Fields(line)
		if len(fields) != 2 {
			continue
		}
		value, err := strconv.Atoi(fields[1])
		if err == nil {
			counts[fields[0]] = value
		}
	}
	require.Len(t, counts, 4, "unexpected mysql count output:\n%s", output)
	return counts
}

func loadTemplateBudgets(t *testing.T) templateBudgets {
	t.Helper()
	path := filepath.Join(frameworkRootForTest(t), "docs", "templates", "v1", "budgets.json")
	content, err := os.ReadFile(path)
	require.NoError(t, err)
	var budgets templateBudgets
	require.NoError(t, json.Unmarshal(content, &budgets))
	require.NotEmpty(t, budgets.Templates)
	return budgets
}

func directDependencyCount(t *testing.T, dir string) int {
	t.Helper()
	content, err := os.ReadFile(filepath.Join(dir, "go.mod"))
	require.NoError(t, err)
	file, err := modfile.Parse("go.mod", content, nil)
	require.NoError(t, err)
	count := 0
	for _, requirement := range file.Require {
		if !requirement.Indirect {
			count++
		}
	}
	return count
}

func externalServiceCount(document manifest.Document) int {
	services := map[string]bool{}
	for _, capability := range document.Capabilities {
		for _, service := range capability.RuntimeServices {
			services[service] = true
		}
	}
	return len(services)
}

func healthRouteCount(t *testing.T, dir string) int {
	t.Helper()
	content, err := os.ReadFile(filepath.Join(dir, "internal", "platform", "health", "health.go"))
	require.NoError(t, err)
	source := string(content)
	if strings.Contains(source, `HandleFunc("/livez"`) && strings.Contains(source, `HandleFunc("/readyz"`) {
		return 2
	}
	t.Fatalf("generated health handler does not expose both probes:\n%s", source)
	return 0
}

func sortedKeys[V any](values map[string]V) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	return keys
}
