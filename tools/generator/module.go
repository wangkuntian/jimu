package generator

import (
	"bytes"
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"text/template"
)

// GenerateModule 生成完整的模块骨架（Clean Architecture 分层）
func GenerateModule(name string) error {
	root, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get working directory: %w", err)
	}
	return GenerateModuleAt(root, name)
}

func GenerateModuleAt(root, name string) error {
	data, targets, err := preflight(root, name)
	if err != nil {
		return err
	}
	files, err := renderAll(data, targets)
	if err != nil {
		return err
	}
	if err := writeAll(root, files); err != nil {
		return err
	}
	fmt.Printf("Module '%s' created at internal/capabilities/%s/\n", name, name)
	return nil
}

type templateData struct {
	Name            string
	NameCamel       string
	VarName         string
	TableName       string
	RouteName       string
	MigrationNumber string
}

type targetFile struct {
	path     string
	template string
}

var validModuleName = regexp.MustCompile(`^[a-z][a-z0-9]*(?:_[a-z0-9]+)*$`)

var goKeywords = map[string]bool{
	"break": true, "default": true, "func": true, "interface": true, "select": true,
	"case": true, "defer": true, "go": true, "map": true, "struct": true,
	"chan": true, "else": true, "goto": true, "package": true, "switch": true,
	"const": true, "fallthrough": true, "if": true, "range": true, "type": true,
	"continue": true, "for": true, "import": true, "return": true, "var": true,
}

func preflight(root, name string) (templateData, []targetFile, error) {
	if !validModuleName.MatchString(name) || goKeywords[name] {
		return templateData{}, nil, fmt.Errorf("invalid module name: %q", name)
	}
	capMigDir := filepath.Join("internal", "capabilities", name, "migrations", "mysql")
	for _, rel := range []string{"go.mod", filepath.Join("internal", "capabilities")} {
		if _, err := os.Stat(filepath.Join(root, rel)); err != nil {
			return templateData{}, nil, fmt.Errorf("repository missing %s: %w", rel, err)
		}
	}
	// 能力内自行编号：读该能力迁移目录取最大编号 +1；目录不存在（新能力）从 001 起。
	migrationNumber, err := nextMigrationNumber(filepath.Join(root, capMigDir))
	if err != nil {
		return templateData{}, nil, err
	}
	data := templateData{
		Name:            name,
		NameCamel:       camel(name),
		VarName:         lowerCamel(name),
		TableName:       name + "s",
		RouteName:       strings.ReplaceAll(name, "_", "-") + "s",
		MigrationNumber: migrationNumber,
	}
	targets := []targetFile{
		{filepath.Join("internal", "capabilities", name, "module.go"), moduleTemplate},
		{filepath.Join("internal", "capabilities", name, "domain", "entity.go"), entityTemplate},
		{filepath.Join("internal", "capabilities", name, "domain", "repository.go"), repositoryTemplate},
		{filepath.Join("internal", "capabilities", name, "application", "dto.go"), dtoTemplate},
		{filepath.Join("internal", "capabilities", name, "application", "errors.go"), errorsTemplate},
		{filepath.Join("internal", "capabilities", name, "application", "service.go"), serviceTemplate},
		{filepath.Join("internal", "capabilities", name, "application", "service_test.go"), serviceTestTemplate},
		{filepath.Join("internal", "capabilities", name, "infrastructure", "mysql_repository.go"), mysqlRepoTemplate},
		{filepath.Join("internal", "capabilities", name, "interfaces", "handler.go"), handlerTemplate},
		{filepath.Join("internal", "capabilities", name, "interfaces", "handler_test.go"), handlerTestTemplate},
		{filepath.Join("internal", "capabilities", name, "interfaces", "router.go"), routerTemplate},
		{filepath.Join("internal", "capabilities", name, "interfaces", "router.go"), routerTemplate},
		// 迁移写进能力目录：两个方言都生成（postgres 版由生成者手调方言差异），
		// postgres/.gitkeep 占位避免能力无 postgres 迁移时 embed 缺目录编译失败。
		{filepath.Join("internal", "capabilities", name, "migrations", "mysql", migrationNumber+"_create_"+data.TableName+".sql"), migrationTemplate},
		{filepath.Join("internal", "capabilities", name, "migrations", "postgres", migrationNumber+"_create_"+data.TableName+".sql"), migrationPostgresTemplate},
		{filepath.Join("internal", "capabilities", name, "migrations", "postgres", ".gitkeep"), ""},
	}
	for _, target := range targets {
		if _, err := os.Stat(filepath.Join(root, target.path)); err == nil {
			return templateData{}, nil, fmt.Errorf("target already exists: %s", target.path)
		} else if !os.IsNotExist(err) {
			return templateData{}, nil, fmt.Errorf("check target %s: %w", target.path, err)
		}
	}
	return data, targets, nil
}

func nextMigrationNumber(dir string) (string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return "001", nil
		}
		return "", fmt.Errorf("read migrations: %w", err)
	}
	max := 0
	for _, entry := range entries {
		name := entry.Name()
		if len(name) < 3 {
			continue
		}
		n, err := strconv.Atoi(name[:3])
		if err == nil && n > max {
			max = n
		}
	}
	return fmt.Sprintf("%03d", max+1), nil
}

func renderAll(data templateData, targets []targetFile) (map[string][]byte, error) {
	files := make(map[string][]byte, len(targets))
	for _, target := range targets {
		t, err := template.New(filepath.Base(target.path)).Parse(target.template)
		if err != nil {
			return nil, err
		}
		var buf bytes.Buffer
		if err := t.Execute(&buf, data); err != nil {
			return nil, err
		}
		content := buf.Bytes()
		if strings.HasSuffix(target.path, ".go") {
			formatted, err := format.Source(content)
			if err != nil {
				return nil, fmt.Errorf("format %s: %w", target.path, err)
			}
			content = formatted
		}
		files[target.path] = content
	}
	return files, nil
}

var writeFile = os.WriteFile

func writeAll(root string, files map[string][]byte) error {
	createdFiles := make([]string, 0, len(files))
	createdDirs := make([]string, 0, len(files))
	seenDirs := make(map[string]bool)
	for rel, content := range files {
		path := filepath.Join(root, rel)
		dir := filepath.Dir(path)
		if err := mkdirAllTracked(root, dir, seenDirs, &createdDirs); err != nil {
			rollback(createdFiles, createdDirs)
			return err
		}
		if err := writeFile(path, content, 0o644); err != nil {
			rollback(createdFiles, createdDirs)
			return fmt.Errorf("write %s: %w", rel, err)
		}
		createdFiles = append(createdFiles, path)
	}
	return nil
}

func mkdirAllTracked(root, dir string, seen map[string]bool, created *[]string) error {
	if seen[dir] {
		return nil
	}
	seen[dir] = true
	if _, err := os.Stat(dir); err == nil {
		return nil
	}
	parent := filepath.Dir(dir)
	if parent != dir && strings.HasPrefix(parent, root) {
		if err := mkdirAllTracked(root, parent, seen, created); err != nil {
			return err
		}
	}
	if err := os.Mkdir(dir, 0o755); err != nil && !os.IsExist(err) {
		return fmt.Errorf("create directory %s: %w", dir, err)
	}
	*created = append(*created, dir)
	return nil
}

func rollback(files, dirs []string) {
	for i := len(files) - 1; i >= 0; i-- {
		_ = os.Remove(files[i])
	}
	for i := len(dirs) - 1; i >= 0; i-- {
		_ = os.Remove(dirs[i])
	}
}

func camel(name string) string {
	parts := strings.Split(name, "_")
	for i, part := range parts {
		parts[i] = strings.ToUpper(part[:1]) + part[1:]
	}
	return strings.Join(parts, "")
}

func lowerCamel(name string) string {
	value := camel(name)
	return strings.ToLower(value[:1]) + value[1:]
}
