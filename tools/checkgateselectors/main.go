// Command checkgateselectors 守住 `-run` 选择器与真实测试名的一致性。
//
// `go test -list` 在选择器零匹配时退出码仍为 0，于是删掉或改名测试后，
// 门禁会静默变成"什么都没跑"。本命令要求每个 SCAFFOLD_*_TESTS 至少匹配一个测试。
package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

var selectorLine = regexp.MustCompile(`^(SCAFFOLD_[A-Z_]+_TESTS)\s*:=\s*(.*\S)\s*$`)

func parseSelectors(makefile string) (map[string]string, error) {
	selectors := map[string]string{}
	scanner := bufio.NewScanner(strings.NewReader(makefile))
	for scanner.Scan() {
		match := selectorLine.FindStringSubmatch(scanner.Text())
		if match == nil {
			continue
		}
		selectors[match[1]] = match[2]
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	if len(selectors) == 0 {
		return nil, fmt.Errorf("no SCAFFOLD_*_TESTS selector found in Makefile")
	}
	return selectors, nil
}

func requireMatches(selector, output string) error {
	matched := 0
	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		if strings.HasPrefix(scanner.Text(), "Test") {
			matched++
		}
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	if matched == 0 {
		return fmt.Errorf("selector %q matches no test; the gate would silently pass", selector)
	}
	return nil
}

func repositoryRoot(start string) (string, error) {
	dir, err := filepath.Abs(start)
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("no go.mod above %s", start)
		}
		dir = parent
	}
}

func run() error {
	root, err := repositoryRoot(".")
	if err != nil {
		return err
	}
	content, err := os.ReadFile(filepath.Join(root, "Makefile"))
	if err != nil {
		return err
	}
	selectors, err := parseSelectors(string(content))
	if err != nil {
		return err
	}
	names := make([]string, 0, len(selectors))
	for name := range selectors {
		names = append(names, name)
	}
	sort.Strings(names)

	goBinary := os.Getenv("GO")
	if goBinary == "" {
		goBinary = "go"
	}
	failed := false
	for _, name := range names {
		selector := selectors[name]
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		cmd := exec.CommandContext(ctx, goBinary, "test", "./tools/generator/", "-list", selector)
		cmd.Dir = root
		output, err := cmd.CombinedOutput()
		cancel()
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s: go test -list failed: %v\n%s", name, err, output)
			failed = true
			continue
		}
		if err := requireMatches(selector, string(output)); err != nil {
			fmt.Fprintf(os.Stderr, "%s: %v\n%s", name, err, output)
			failed = true
			continue
		}
		fmt.Printf("%s: ok (%s)\n", name, selector)
	}
	if failed {
		return fmt.Errorf("gate selectors do not match the test suite")
	}
	return nil
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
