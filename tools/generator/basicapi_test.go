package generator

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"golang.org/x/mod/modfile"
)

func TestBasicAPIBudgetAndDirectDependencies(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "basic")
	result, err := newProjectForTest(t, NewOptions{
		Dir: dir, Template: "basic-api", Module: "example.com/basic",
	})
	require.NoError(t, err)
	require.LessOrEqual(t, result.FileCount-1, 40, "generated files excluding manifest")

	content, err := os.ReadFile(filepath.Join(dir, "go.mod"))
	require.NoError(t, err)
	file, err := modfile.Parse("go.mod", content, nil)
	require.NoError(t, err)
	direct := 0
	for _, requirement := range file.Require {
		if !requirement.Indirect {
			direct++
		}
	}
	require.LessOrEqual(t, direct, 8)
}

func TestBasicAPIStartsWithoutExternalServices(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "basic")
	_, err := newProjectForTest(t, NewOptions{
		Dir: dir, Template: "basic-api", Module: "example.com/basic",
	})
	require.NoError(t, err)

	binary := filepath.Join(dir, "bin", "server")
	env := serverEnvBaseForTest(newTestGoCache(t))
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
	require.Eventually(t, func() bool {
		return statusOK(client, readyURL) && statusOK(client, publicURL)
	}, 15*time.Second, 100*time.Millisecond, "basic-api did not become ready:\n%s", &logs)

	require.NoError(t, command.Process.Signal(os.Interrupt))
	select {
	case err := <-waitCh:
		require.NoError(t, err, "basic-api logs:\n%s", &logs)
	case <-time.After(5 * time.Second):
		t.Fatalf("basic-api did not exit after interrupt:\n%s", &logs)
	}
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
