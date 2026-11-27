package generator

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"

	"jimu/tools/generator/manifest"
	"jimu/tools/generator/workspace"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestOfficialPresetAudit 是一次可重复的生成产物审计：同时生成 minimal 与
// auth(compose+observability)，然后检查构建、测试、Makefile 接线、Compose 展开和
// Secret 初始化。它不启动长期容器；真实启动仍由 TestAuthPresetEmptyDatabaseEndToEnd
// 与 TestMinimalPresetStartsWithoutDependencies 覆盖。
//
// 目标是让「生成 → 检查 → 修复」的循环有固定入口，而不是依赖手工创建 bin/test。
func TestOfficialPresetAudit(t *testing.T) {
	requireHeavyMatrix(t)
	requireDockerCLI(t)

	cache := newTestGoCache(t)
	root := t.TempDir()
	minimalDir := filepath.Join(root, "minimal-audit-dir")
	authDir := filepath.Join(root, "auth-audit-dir")
	const (
		minimalModule = "example.com/audit-minimal"
		authModule    = "example.com/audit-auth"
		minimalName   = "audit-minimal"
		authName      = "audit-auth"
	)

	minimalResult, err := newProjectForTest(t, NewOptions{
		Dir: minimalDir, Template: "minimal", Module: minimalModule, NoTidy: true,
	})
	require.NoError(t, err)
	authResult, err := newProjectForTest(t, NewOptions{
		Dir: authDir, Template: "auth", Deploy: "compose,observability", Module: authModule, NoTidy: true,
	})
	require.NoError(t, err)

	t.Run("minimal", func(t *testing.T) {
		assert.Equal(t, "minimal", minimalResult.Template)
		assert.Empty(t, minimalResult.Resolved)

		document, err := workspace.LoadProjectManifest(minimalDir)
		require.NoError(t, err)
		assert.Equal(t, manifest.GenerationRuntimeLean, document.Generation.Runtime)
		assert.Empty(t, document.Generation.Deploy)

		assert.NoDirExists(t, filepath.Join(minimalDir, "deploy"))
		assert.NoDirExists(t, filepath.Join(minimalDir, "internal", "capabilities"))
		assert.NoFileExists(t, filepath.Join(minimalDir, "conf", "rbac_model.conf"))
		assert.FileExists(t, filepath.Join(minimalDir, ".dockerignore"))

		makefile := string(readFileForTest(t, filepath.Join(minimalDir, "Makefile")))
		for _, absent := range []string{"\nup:", "\nup-observability:", "\nmigrate:", "\nseed:", "prepare-secrets", "init-secrets.sh"} {
			assert.NotContains(t, makefile, absent, "minimal 不应生成 %s", absent)
		}
		readme := string(readFileForTest(t, filepath.Join(minimalDir, "README.md")))
		assert.Contains(t, readme, "# "+minimalName)
		assert.NotContains(t, readme, "make up")
		assert.NotContains(t, readme, "make migrate")
		assert.Contains(t, readText(t, filepath.Join(minimalDir, "internal/platform/http/serviceinfo.go")),
			`serviceName            = "`+minimalName+`"`)

		configKeys := topLevelConfigKeys(t, filepath.Join(minimalDir, "configs/app.yaml"))
		assert.Equal(t,
			[]string{"http", "log", "management", "security", "server"},
			configKeys,
			"minimal 配置应只保留有消费者的内核段")

		assertProjectBuilds(t, minimalDir, cache)
		assertProjectVets(t, minimalDir, cache)
		output, err := runGoInProjectOutput(t, minimalDir, cache, "test", "./...", "-count=1")
		require.NoError(t, err, "minimal go test 必须全绿:\n%s", output)
	})

	t.Run("auth", func(t *testing.T) {
		assert.Equal(t, "auth", authResult.Template)
		assert.Equal(t, "certified", authResult.Support)
		assert.ElementsMatch(t, []string{"user", "access", "auth"}, authResult.Resolved)

		document, err := workspace.LoadProjectManifest(authDir)
		require.NoError(t, err)
		assert.Equal(t, manifest.GenerationRuntimeStandard, document.Generation.Runtime,
			"observability 需要 standard 运行时的 OTel/reporter/metrics")
		assert.Equal(t, []string{"compose", "observability"}, document.Generation.Deploy)

		for _, rel := range []string{
			"conf/rbac_model.conf",
			"deploy/compose/docker-compose.yml",
			"deploy/compose/compose.observability.yml",
			"deploy/docker/Dockerfile",
			"deploy/observability/otel-collector.yaml",
			"deploy/scripts/init-secrets.sh",
			"deploy/scripts/observability.sh",
			"deploy/scripts/backup.sh",
			"deploy/backup/Dockerfile",
		} {
			assert.FileExists(t, filepath.Join(authDir, filepath.FromSlash(rel)), "auth 缺少 %s", rel)
		}
		assert.NoDirExists(t, filepath.Join(authDir, "deploy/k8s"))
		assert.NoDirExists(t, filepath.Join(authDir, "deploy/helm"))

		dockerfile := readText(t, filepath.Join(authDir, "deploy/docker/Dockerfile"))
		assert.Contains(t, dockerfile, "FROM golang:1.27.4-alpine")
		assert.Contains(t, dockerfile, "-o "+authName+" ./cmd/cli")
		assert.Contains(t, dockerfile, "COPY conf/")
		assert.NotContains(t, dockerfile, "COPY secrets")

		composeText := readText(t, filepath.Join(authDir, "deploy/compose/docker-compose.yml"))
		for _, needle := range []string{
			"jimu-init:",
			"./" + authName + " migrate up",
			authName + "-mariadb-data",
			authName + "-redis-data",
		} {
			assert.Contains(t, composeText, needle, "compose 缺少 %s", needle)
		}

		makefile := readText(t, filepath.Join(authDir, "Makefile"))
		for _, needle := range []string{
			"COMPOSE_PROJECT_NAME ?= " + authName,
			`COMPOSE_PROJECT_NAME="$(COMPOSE_PROJECT_NAME)"`,
			".NOTPARALLEL: up up-observability",
			"up: prepare-secrets",
			"prepare-observability-secrets:",
			"up-observability: prepare-secrets prepare-observability-secrets",
			"./deploy/scripts/init-secrets.sh --observability --no-legacy",
			"./deploy/scripts/observability.sh start",
		} {
			assert.Contains(t, makefile, needle, "生成的 Makefile 缺少 %s", needle)
		}
		assert.Contains(t, readText(t, filepath.Join(authDir, "README.md")), "make up-observability")
		assert.Contains(t, readText(t, filepath.Join(authDir, "README.md")), "make migrate")
		assert.Contains(t, readText(t, filepath.Join(authDir, "internal/platform/http/serviceinfo.go")),
			`serviceName            = "`+authName+`"`)
		assert.Contains(t, topLevelConfigKeys(t, filepath.Join(authDir, "configs/app.yaml")), "db")
		assert.Contains(t, topLevelConfigKeys(t, filepath.Join(authDir, "configs/app.yaml")), "otel")

		core := composeConfigForAudit(t, authDir, authName, cache)
		require.Contains(t, core.Services, "jimu-init")
		require.Contains(t, core.Services, "mariadb")
		require.Contains(t, core.Services, "redis")
		assert.Equal(t, []string{"/bin/sh", "-c", "./" + authName + " migrate up && ./" + authName + " seed"},
			core.Services["jimu-init"].Command)
		assert.Equal(t, "service_completed_successfully",
			core.Services["server"].DependsOn["jimu-init"].Condition)
		assert.Equal(t, "service_healthy", core.Services["jimu-init"].DependsOn["mariadb"].Condition)
		assert.Equal(t, "service_healthy", core.Services["jimu-init"].DependsOn["redis"].Condition)
		assert.Equal(t, authName+":latest", core.Services["server"].Image)
		assert.Contains(t, core.Volumes, authName+"-mariadb-data")
		assert.Contains(t, core.Volumes, authName+"-redis-data")

		all := composeConfigForAudit(t, authDir, authName, cache, "deploy/compose/compose.observability.yml")
		require.Contains(t, all.Services, "openobserve")
		require.Contains(t, all.Services, "otel-collector")
		assert.Contains(t, all.Volumes, authName+"-openobserve-data")
		assert.Contains(t, all.Secrets, "zo_auth_token")

		preparePlan := runMakePlan(t, authDir, cache, "prepare-secrets")
		assert.Contains(t, preparePlan, `COMPOSE_PROJECT_NAME="`+authName+`"`)
		assert.Contains(t, preparePlan, `SECRETS_DIR="`)
		observabilityPlan := runMakePlan(t, authDir, cache, "up-observability")
		assert.Contains(t, observabilityPlan, "--observability --no-legacy")
		assert.Contains(t, observabilityPlan, "init-secrets.sh --no-legacy")
		assert.Contains(t, observabilityPlan, "./deploy/scripts/observability.sh start")
		assert.Contains(t, observabilityPlan, "--project-name "+authName)

		assertProjectBuilds(t, authDir, cache)
		assertProjectVets(t, authDir, cache)
		output, err := runGoInProjectOutput(t, authDir, cache, "test", "./...", "-count=1")
		require.NoError(t, err, "auth go test 必须全绿:\n%s", output)
	})
}

// auditComposeConfig 是 docker compose config --format json 的最小投影。
type auditComposeConfig struct {
	Services map[string]struct {
		Image     string   `json:"image"`
		Command   []string `json:"command"`
		DependsOn map[string]struct {
			Condition string `json:"condition"`
		} `json:"depends_on"`
	} `json:"services"`
	Volumes map[string]struct {
		Name string `json:"name"`
	} `json:"volumes"`
	Secrets map[string]json.RawMessage `json:"secrets"`
}

func requireDockerCLI(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("docker"); err != nil {
		t.Fatalf("生成产物审计需要 docker CLI（只执行 compose config）: %v", err)
	}
	command := exec.Command("docker", "compose", "version")
	output, err := command.CombinedOutput()
	require.NoError(t, err, "生成产物审计需要 docker compose 插件:\n%s", output)
}

func composeConfigForAudit(t *testing.T, dir, project, cache string, extraFiles ...string) auditComposeConfig {
	t.Helper()
	args := []string{
		"compose", "--project-name", project, "--project-directory", dir,
		"--env-file", "/dev/null", "-f", "deploy/compose/docker-compose.yml",
	}
	for _, file := range extraFiles {
		args = append(args, "-f", file)
	}
	args = append(args, "config", "--format", "json")
	command := exec.Command("docker", args...)
	command.Dir = dir
	command.Env = append(dockerTestEnv(cache), "JIMU_SECRETS_DIR="+filepath.Join(dir, "secrets"))
	var stdout, stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr
	require.NoError(t, command.Run(), "docker compose config 失败:\n%s", stderr.String())

	var config auditComposeConfig
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &config), "compose config 不是合法 JSON:\n%s", stdout.String())
	return config
}

func runMakePlan(t *testing.T, dir, cache, target string) string {
	t.Helper()
	command := exec.Command("make", "-n", target)
	command.Dir = dir
	command.Env = dockerTestEnv(cache)
	output, err := command.CombinedOutput()
	require.NoError(t, err, "make -n %s 失败:\n%s", target, output)
	return string(output)
}

func readText(t *testing.T, path string) string {
	t.Helper()
	return string(readFileForTest(t, path))
}

var topLevelConfigKeyPattern = regexp.MustCompile(`(?m)^([a-z_][a-z0-9_]*):`)

func topLevelConfigKeys(t *testing.T, path string) []string {
	t.Helper()
	matches := topLevelConfigKeyPattern.FindAllStringSubmatch(readText(t, path), -1)
	keys := make([]string, 0, len(matches))
	for _, match := range matches {
		keys = append(keys, match[1])
	}
	slices.Sort(keys)
	return slices.Compact(keys)
}

// auditDockerStub 返回一个只记录 volume inspect 的假 docker。init-secrets.sh
// 只会在卷探测阶段调用它；返回 "No such volume" 让脚本走首次初始化路径。
func auditDockerStub(t *testing.T) (binDir, logPath string) {
	t.Helper()
	binDir = filepath.Join(t.TempDir(), "bin")
	require.NoError(t, os.MkdirAll(binDir, 0o755))
	logPath = filepath.Join(t.TempDir(), "docker.log")
	script := `#!/usr/bin/env bash
set -euo pipefail
printf '%s\n' "$*" >> "$JIMU_AUDIT_DOCKER_LOG"
if [ "${1:-}" = "volume" ] && [ "${2:-}" = "inspect" ]; then
  printf 'Error response from daemon: No such volume: %s\n' "${3:-}" >&2
  exit 1
fi
exit 0
`
	require.NoError(t, os.WriteFile(filepath.Join(binDir, "docker"), []byte(script), 0o755))
	return binDir, logPath
}

func TestAuditInitSecretsUsesComposeProjectName(t *testing.T) {
	requireHeavyMatrix(t)

	dir := filepath.Join(t.TempDir(), "auth-audit-dir")
	const project = "audit-auth"
	_, err := newProjectForTest(t, NewOptions{
		Dir: dir, Template: "auth", Deploy: "compose,observability", Module: "example.com/" + project, NoTidy: true,
	})
	require.NoError(t, err)

	binDir, logPath := auditDockerStub(t)
	secretsDir := filepath.Join(dir, "secrets")
	command := exec.Command("bash", "deploy/scripts/init-secrets.sh", "--observability", "--no-legacy")
	command.Dir = dir
	command.Env = []string{
		"PATH=" + binDir + string(os.PathListSeparator) + os.Getenv("PATH"),
		"HOME=" + os.Getenv("HOME"),
		"TMPDIR=" + os.Getenv("TMPDIR"),
		"USER=" + os.Getenv("USER"),
		"LANG=" + os.Getenv("LANG"),
		"LC_ALL=" + os.Getenv("LC_ALL"),
		"SECRETS_DIR=" + secretsDir,
		"COMPOSE_PROJECT_NAME=" + project,
		"JIMU_AUDIT_DOCKER_LOG=" + logPath,
	}
	output, err := command.CombinedOutput()
	require.NoError(t, err, "init-secrets.sh 失败:\n%s", output)

	calls := strings.Split(strings.TrimSpace(readText(t, logPath)), "\n")
	assert.ElementsMatch(t, []string{
		"volume inspect " + project + "-mariadb-data",
		"volume inspect " + project + "-openobserve-data",
	}, calls, "卷探测必须使用 Compose 项目名，而不是目录名")
	assert.NotContains(t, calls, "volume inspect "+filepath.Base(dir)+"-mariadb-data")

	for _, name := range []string{
		"db_root_password.txt",
		"db_password.txt",
		"jwt_secret.txt",
		"encryption_key.txt",
		"audit_hash_secret.txt",
		"admin_password.txt",
		"otel_auth_password.txt",
		"openobserve.env",
		"zo_auth_token.txt",
	} {
		assert.FileExists(t, filepath.Join(secretsDir, name), "缺少 secret %s", name)
	}
	assert.Contains(t, readText(t, filepath.Join(secretsDir, "openobserve.env")), "ZO_ROOT_USER_EMAIL=")
	assert.NotEmpty(t, readText(t, filepath.Join(secretsDir, "zo_auth_token.txt")))
}
