package generator

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestBasicAPIStartsWithoutExternalServices(t *testing.T) {
	started := time.Now()
	dir := filepath.Join(t.TempDir(), "basic")
	_, err := newProjectForTest(t, NewOptions{
		Dir: dir, Template: "basic-api", Module: "example.com/basic",
	})
	require.NoError(t, err)

	binary := filepath.Join(dir, "bin", "server")
	cache := newTestGoCache(t)
	env := serverEnvBaseForTest(cache)
	assertProjectBuilds(t, dir, cache)
	assertProjectVets(t, dir, cache)
	runGoInProject(t, dir, cache, "test", "./...")
	build := exec.Command("go", "build", "-o", binary, "./cmd/server")
	build.Dir = dir
	build.Env = env
	output, err := build.CombinedOutput()
	require.NoError(t, err, "build basic-api:\n%s", output)

	publicPort := freePort(t)
	managementPort := freePort(t)
	command := exec.Command(binary)
	command.Dir = dir
	command.Env = append(append([]string{}, env...),
		"APP_ENV=dev",
		"HTTP_HOST=127.0.0.1",
		fmt.Sprintf("HTTP_PORT=%d", publicPort),
		"MANAGEMENT_HOST=127.0.0.1",
		fmt.Sprintf("MANAGEMENT_PORT=%d", managementPort),
	)
	var logs bytes.Buffer
	command.Stdout = &logs
	command.Stderr = &logs
	require.NoError(t, command.Start())
	waitCh := make(chan error, 1)
	go func() { waitCh <- command.Wait() }()
	t.Cleanup(func() {
		if command.Process != nil {
			_ = command.Process.Kill()
		}
	})

	client := &http.Client{Timeout: time.Second}
	publicURL := fmt.Sprintf("http://127.0.0.1:%d/api/v1/ping", publicPort)
	readyURL := fmt.Sprintf("http://127.0.0.1:%d/readyz", managementPort)
	liveURL := fmt.Sprintf("http://127.0.0.1:%d/livez", managementPort)
	require.Eventually(t, func() bool {
		return statusOK(client, liveURL) && statusOK(client, readyURL) && statusOK(client, publicURL)
	}, 15*time.Second, 100*time.Millisecond, "basic-api did not become ready:\n%s", &logs)
	limit := loadTemplateBudgets(t).Templates["basic-api"].ReadinessSeconds["cold_generate_and_start"]
	require.Positive(t, limit)
	require.LessOrEqual(t, time.Since(started), time.Duration(limit)*time.Second,
		"basic-api cold generate/start exceeded the %ds contract budget", limit)

	status, body := requestJSON(t, client, http.MethodGet, fmt.Sprintf("http://127.0.0.1:%d/api/v1/missing", publicPort))
	require.Equal(t, http.StatusNotFound, status)
	require.Equal(t, 1004, body.Code)
	status, body = requestJSON(t, client, http.MethodPost, publicURL)
	require.Equal(t, http.StatusMethodNotAllowed, status)
	require.Equal(t, 1005, body.Code)

	require.NoError(t, command.Process.Signal(syscall.SIGTERM))
	select {
	case err := <-waitCh:
		require.NoError(t, err, "basic-api logs:\n%s", &logs)
	case <-time.After(5 * time.Second):
		t.Fatalf("basic-api did not exit after interrupt:\n%s", &logs)
	}
}

func TestAuthAPIComposeConfigIsValid(t *testing.T) {
	if _, err := exec.LookPath("docker"); err != nil {
		t.Skip("docker CLI is not installed")
	}
	dir := filepath.Join(t.TempDir(), "auth")
	_, err := newProjectForTest(t, NewOptions{
		Dir: dir, Template: "auth-api", Module: "example.com/auth",
	})
	require.NoError(t, err)
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "secrets"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "secrets", "admin_password.txt"), []byte("admin12345678"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "secrets", "jwt_secret.txt"), []byte("compose-config-secret"), 0o600))

	command := exec.Command("docker", "compose", "config", "--format", "json")
	command.Dir = dir
	command.Env = append(serverEnvBaseForTest(newTestGoCache(t)),
		"AUTH_API_PORT=18080",
		"AUTH_API_MANAGEMENT_PORT=19090",
	)
	output, err := command.CombinedOutput()
	require.NoError(t, err, "docker compose config:\n%s", output)

	var config struct {
		Secrets map[string]struct {
			File string `json:"file"`
		} `json:"secrets"`
		Services map[string]struct {
			Secrets []struct {
				Source string `json:"source"`
				Target string `json:"target"`
			} `json:"secrets"`
			Ports []struct {
				Target    int    `json:"target"`
				Published string `json:"published"`
			} `json:"ports"`
		} `json:"services"`
	}
	require.NoError(t, json.Unmarshal(output, &config), "docker compose config JSON:\n%s", output)
	require.Equal(t, "admin_password.txt", filepath.Base(config.Secrets["admin_password"].File))
	require.Equal(t, "jwt_secret.txt", filepath.Base(config.Secrets["jwt_secret"].File))
	for _, service := range []string{"init", "app"} {
		mounted := map[string]string{}
		for _, secret := range config.Services[service].Secrets {
			mounted[secret.Source] = secret.Target
		}
		require.Equal(t, map[string]string{
			"admin_password": "/run/secrets/admin_password",
			"jwt_secret":     "/run/secrets/jwt_secret",
		}, mounted, "%s must receive the files through Compose secrets", service)
	}
	published := map[int]string{}
	for _, port := range config.Services["app"].Ports {
		published[port.Target] = port.Published
	}
	require.Equal(t, "18080", published[8080], "public API host port must be overridable")
	require.Equal(t, "19090", published[9090], "management host port must be overridable")

	dockerfile, err := os.ReadFile(filepath.Join(dir, "Dockerfile"))
	require.NoError(t, err)
	require.Contains(t, string(dockerfile), "http://127.0.0.1:9090/readyz",
		"make up --wait must wait for readiness, not only process liveness")
}

func freePort(t *testing.T) int {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer listener.Close()
	return listener.Addr().(*net.TCPAddr).Port
}

func statusOK(client *http.Client, url string) bool {
	response, err := client.Get(url)
	if err != nil {
		return false
	}
	defer response.Body.Close()
	_, _ = io.Copy(io.Discard, response.Body)
	return response.StatusCode == http.StatusOK
}

func requestJSON(t *testing.T, client *http.Client, method, url string) (int, struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}) {
	t.Helper()
	request, err := http.NewRequest(method, url, nil)
	require.NoError(t, err)
	response, err := client.Do(request)
	require.NoError(t, err)
	defer response.Body.Close()
	var body struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	}
	require.NoError(t, json.NewDecoder(response.Body).Decode(&body))
	require.NotEmpty(t, body.Message)
	return response.StatusCode, body
}
