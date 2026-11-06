package generator

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
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

// waitHTTP 轮询 url 直到 2xx，返回响应体；超时则带上进程日志失败。
func waitHTTP(t *testing.T, url, logPath string) string {
	t.Helper()
	deadline := time.Now().Add(60 * time.Second)
	client := &http.Client{Timeout: 2 * time.Second}
	var last error
	for time.Now().Before(deadline) {
		response, err := client.Get(url)
		if err == nil {
			body, readErr := io.ReadAll(response.Body)
			_ = response.Body.Close()
			if readErr == nil && response.StatusCode >= 200 && response.StatusCode < 300 {
				return string(body)
			}
			last = fmt.Errorf("status %d: %s", response.StatusCode, body)
		} else {
			last = err
		}
		time.Sleep(200 * time.Millisecond)
	}
	logs, _ := os.ReadFile(logPath)
	t.Fatalf("%s did not become healthy: %v\nserver logs:\n%s", url, last, logs)
	return ""
}

type templateBudgets struct {
	SchemaVersion int `json:"schema_version"`
	Templates     map[string]struct {
		Limits           map[string]int `json:"limits"`
		ReadinessSeconds map[string]int `json:"readiness_seconds"`
	} `json:"templates"`
}

func TestPresetBudgets(t *testing.T) {
	budgets := loadTemplateBudgets(t)
	assert.Equal(t, 1, budgets.SchemaVersion)

	for _, template := range []string{"minimal", "auth", "full"} {
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

// TestMinimalPresetStartsWithoutDependencies 是 minimal 的产品承诺：0 能力、0 外部服务，
// `go run ./cmd/server` 直接可用。两种 env 都要起得来：不设 ADMIN_PASSWORD（跳过种子），
// 以及设置 ADMIN_PASSWORD（0 能力时 DB 为 nil，种子必须被护栏跳过而不是 panic）。
func TestMinimalPresetStartsWithoutDependencies(t *testing.T) {
	for _, tc := range []struct {
		name string
		env  []string
	}{
		{"without-admin-password", nil},
		{"with-admin-password", []string{"ADMIN_PASSWORD=minimal-smoke-password"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := filepath.Join(t.TempDir(), "minimal")
			_, err := newProjectForTest(t, NewOptions{
				Dir: dir, Template: "minimal", Module: "example.com/minimal", NoTidy: true,
			})
			require.NoError(t, err)

			publicPort, managementPort := freePort(t), freePort(t)
			for managementPort == publicPort {
				managementPort = freePort(t)
			}
			env := append(serverEnvBaseForTest(newTestGoCache(t)),
				"APP_ENV=dev",
				fmt.Sprintf("HTTP_PORT=%d", publicPort),
				fmt.Sprintf("MANAGEMENT_PORT=%d", managementPort),
			)
			env = append(env, tc.env...)

			binary := filepath.Join(t.TempDir(), "server")
			build := exec.Command("go", "build", "-o", binary, "./cmd/server")
			build.Dir = dir
			build.Env = env
			buildOutput, err := build.CombinedOutput()
			require.NoError(t, err, "generated server must build:\n%s", buildOutput)

			logPath := filepath.Join(t.TempDir(), "server.log")
			logFile, err := os.Create(logPath)
			require.NoError(t, err)
			t.Cleanup(func() { _ = logFile.Close() })
			command := exec.Command(binary)
			command.Dir = dir
			command.Env = env
			command.Stdout = logFile
			command.Stderr = logFile
			require.NoError(t, command.Start())
			stopped := false
			t.Cleanup(func() {
				if stopped {
					return
				}
				_ = command.Process.Kill()
				_ = command.Wait()
			})

			waitHTTP(t, fmt.Sprintf("http://127.0.0.1:%d/livez", managementPort), logPath)
			waitHTTP(t, fmt.Sprintf("http://127.0.0.1:%d/readyz", managementPort), logPath)
			body := waitHTTP(t, fmt.Sprintf("http://127.0.0.1:%d/api/v1/ping", publicPort), logPath)
			assert.Contains(t, body, "pong")

			stopped = true
			_ = command.Process.Kill()
			_ = command.Wait()
		})
	}
}

func TestAuthPresetEmptyDatabaseEndToEnd(t *testing.T) {
	requireHeavyMatrix(t)
	requireDockerDaemon(t)

	dir := filepath.Join(t.TempDir(), "auth")
	_, err := newProjectForTest(t, NewOptions{
		Dir: dir, Template: "auth", Module: "example.com/auth",
	})
	require.NoError(t, err)

	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	mysqlPort, redisPort := freePort(t), freePort(t)
	publicPort := freePort(t)
	managementPort := freePort(t)
	for managementPort == publicPort || managementPort == mysqlPort || managementPort == redisPort {
		managementPort = freePort(t)
	}

	mysqlName := "jimu-auth-mysql-" + suffix
	redisName := "jimu-auth-redis-" + suffix
	dockerRun(t, "run", "-d", "--name", mysqlName,
		"-e", "MYSQL_ROOT_PASSWORD=root", "-e", "MYSQL_DATABASE=jimu",
		"-e", "MYSQL_USER=jimu", "-e", "MYSQL_PASSWORD=jimu",
		"-p", fmt.Sprintf("127.0.0.1:%d:3306", mysqlPort), "mysql:8.4")
	dockerRun(t, "run", "-d", "--name", redisName,
		"-p", fmt.Sprintf("127.0.0.1:%d:6379", redisPort), "redis:7-alpine")
	t.Cleanup(func() {
		_ = exec.Command("docker", "rm", "-f", mysqlName, redisName).Run()
	})
	waitMySQL(t, mysqlName)
	waitRedis(t, redisName)

	adminPassword := "{{SECRET_admin_e2e_pass}}"
	env := append(serverEnvBaseForTest(newTestGoCache(t)),
		"APP_ENV=dev",
		"DB_HOST=127.0.0.1",
		fmt.Sprintf("DB_PORT=%d", mysqlPort),
		"DB_USER=jimu",
		"DB_PASSWORD=jimu",
		"DB_NAME=jimu",
		fmt.Sprintf("REDIS_ADDR=127.0.0.1:%d", redisPort),
		fmt.Sprintf("HTTP_PORT=%d", publicPort),
		fmt.Sprintf("MANAGEMENT_PORT=%d", managementPort),
		"ADMIN_PASSWORD="+adminPassword,
	)

	binary := filepath.Join(t.TempDir(), "server")
	build := exec.Command("go", "build", "-o", binary, "./cmd/server")
	build.Dir = dir
	build.Env = env
	buildOutput, err := build.CombinedOutput()
	require.NoError(t, err, "generated auth server must build:\n%s", buildOutput)

	logPath := filepath.Join(t.TempDir(), "server.log")
	logFile, err := os.Create(logPath)
	require.NoError(t, err)
	t.Cleanup(func() { _ = logFile.Close() })
	command := exec.Command(binary)
	command.Dir = dir
	command.Env = env
	command.Stdout = logFile
	command.Stderr = logFile
	require.NoError(t, command.Start())
	t.Cleanup(func() {
		_ = command.Process.Kill()
		_ = command.Wait()
	})

	waitHTTP(t, fmt.Sprintf("http://127.0.0.1:%d/readyz", managementPort), logPath)
	body := waitHTTP(t, fmt.Sprintf("http://127.0.0.1:%d/api/v1/ping", publicPort), logPath)
	assert.Contains(t, body, "pong")
	assertAdminLogin(t, publicPort, adminPassword, logPath)
}

// dockerRun 执行 docker 子命令并在失败时带上输出。
func dockerRun(t *testing.T, args ...string) string {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	command := exec.CommandContext(ctx, "docker", args...)
	command.Env = dockerTestEnv("")
	output, err := command.CombinedOutput()
	require.NoError(t, err, "docker %s failed:\n%s", strings.Join(args, " "), output)
	return string(output)
}

func waitMySQL(t *testing.T, container string) {
	t.Helper()
	waitForContainer(t, container, func() bool {
		command := exec.Command("docker", "exec", container,
			"mysqladmin", "ping", "-h127.0.0.1", "-uroot", "-proot", "--silent")
		command.Env = dockerTestEnv("")
		return command.Run() == nil
	})
}

func waitRedis(t *testing.T, container string) {
	t.Helper()
	waitForContainer(t, container, func() bool {
		command := exec.Command("docker", "exec", container, "redis-cli", "ping")
		command.Env = dockerTestEnv("")
		output, err := command.Output()
		return err == nil && strings.Contains(string(output), "PONG")
	})
}

func waitForContainer(t *testing.T, container string, ready func() bool) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Minute)
	for time.Now().Before(deadline) {
		if ready() {
			return
		}
		time.Sleep(500 * time.Millisecond)
	}
	logs := dockerRun(t, "logs", "--tail", "50", container)
	t.Fatalf("container %s did not become ready:\n%s", container, logs)
}

// assertAdminLogin 验证种子真的写入了管理员账号：登录成功且返回 access_token。
func assertAdminLogin(t *testing.T, port int, password, logPath string) {
	t.Helper()
	payload, err := json.Marshal(map[string]string{"username": "admin", "password": password})
	require.NoError(t, err)
	url := fmt.Sprintf("http://127.0.0.1:%d/api/v1/auth/login", port)
	deadline := time.Now().Add(60 * time.Second)
	var last string
	for time.Now().Before(deadline) {
		response, err := http.Post(url, "application/json", strings.NewReader(string(payload)))
		if err == nil {
			body, _ := io.ReadAll(response.Body)
			_ = response.Body.Close()
			if response.StatusCode == http.StatusOK {
				var result struct {
					Data struct {
						AccessToken string `json:"access_token"`
					} `json:"data"`
				}
				require.NoError(t, json.Unmarshal(body, &result), "unexpected login response: %s", body)
				assert.NotEmpty(t, result.Data.AccessToken, "login must return an access token: %s", body)
				return
			}
			last = fmt.Sprintf("status %d: %s", response.StatusCode, body)
		} else {
			last = err.Error()
		}
		time.Sleep(500 * time.Millisecond)
	}
	logs, _ := os.ReadFile(logPath)
	t.Fatalf("admin login did not succeed: %s\nserver logs:\n%s", last, logs)
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
	env = append(env,
		fmt.Sprintf("APP_UID=%d", os.Getuid()),
		fmt.Sprintf("APP_GID=%d", os.Getgid()),
	)
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
	content, err := os.ReadFile(filepath.Join(dir, "internal", "kernel", "observability", "health.go"))
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
