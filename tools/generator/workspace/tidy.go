package workspace

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func Tidy(dir string) error {
	return runGo(dir, []string{"GOFLAGS=-mod=mod", "GOWORK=off"}, "mod", "tidy")
}

func runGo(dir string, env []string, args ...string) error {
	cmd := exec.CommandContext(context.Background(), "go", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), env...)
	var output bytes.Buffer
	cmd.Stdout = &output
	cmd.Stderr = &output
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("go %s: %w\n%s", strings.Join(args, " "), err, strings.TrimRight(output.String(), "\n"))
	}
	return nil
}
