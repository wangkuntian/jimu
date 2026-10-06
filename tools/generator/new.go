package generator

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"maps"
	"os"
	"path"
	"path/filepath"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"

	"jimu/tools/generator/frameworkmanifest"
	"jimu/tools/generator/manifest"
	"jimu/tools/generator/plan"
	"jimu/tools/generator/workspace"
	"jimu/tools/internal/profileassets"
)

// NewOptions 是 `jimu new` 的全部输入。
type NewOptions struct {
	Dir     string
	Profile string
	With    string
	Shape   string
	Module  string
	NoTidy  bool
	DryRun  bool
	Force   bool
	Report  bool

	// NoSelfCheck 只供本仓构建类测试使用；CLI 始终执行自检。
	NoSelfCheck bool
}

// Result 是一次生成或增量更新的摘要。
type Result struct {
	Dir          string
	Module       string
	Shape        string
	Capabilities []string
	CopySet      []string
	Drivers      []string
	Assets       []string
	Files        []string
	FileCount    int
	Lines        int
	Changed      []string
}

// NewProject 是根包 facade；框架事实、动作计划和文件事务分别由职责子包拥有。
func NewProject(opts NewOptions) (*Result, error) {
	root, err := frameworkRoot()
	if err != nil {
		return nil, err
	}
	module, err := resolveModule(opts.Dir, opts.Module)
	if err != nil {
		return nil, err
	}
	if err := validateModule(module); err != nil {
		return nil, err
	}
	doc, err := frameworkmanifest.Export(frameworkmanifest.Request{
		Root: root, Profile: opts.Profile, With: opts.With, Shape: opts.Shape, Module: module,
	})
	if err != nil {
		return nil, err
	}
	if err := manifest.Validate(doc); err != nil {
		return nil, err
	}
	if _, err := plan.Build(doc); err != nil {
		return nil, err
	}
	created, err := workspace.Create(doc, workspace.CreateOptions{
		Target: opts.Dir, SourceRoot: root, Module: module, Force: opts.Force,
		NoTidy: opts.NoTidy, DryRun: opts.DryRun, Report: opts.Report, NoSelfCheck: opts.NoSelfCheck,
	})
	if err != nil {
		return nil, err
	}
	return &Result{
		Dir: created.Target, Module: created.Module, Shape: doc.Selection.Shape,
		Capabilities: slices.Clone(doc.Selection.Capabilities),
		CopySet:      manifestCapabilityNames(doc), Drivers: flattenDrivers(doc.Selection.Drivers),
		Assets: manifestAssetNames(doc), Files: slices.Clone(created.Files), FileCount: created.FileCount,
	}, nil
}

func manifestCapabilityNames(doc manifest.Document) []string {
	values := make([]string, 0, len(doc.Capabilities))
	for _, capability := range doc.Capabilities {
		values = append(values, capability.Name)
	}
	slices.Sort(values)
	return slices.Compact(values)
}

func manifestAssetNames(doc manifest.Document) []string {
	values := make([]string, 0, len(doc.Assets))
	for _, asset := range doc.Assets {
		values = append(values, asset.Destination)
	}
	slices.Sort(values)
	return slices.Compact(values)
}

const compositionManifestDir = "internal/capabilities/catalog"

type testDeps struct {
	pkg            string
	missingImports []string
	missingAssets  []string
	compositionDep bool
	decls          map[string]bool
	refs           map[string]bool
}

func (d testDeps) satisfied(catalogComplete bool) bool {
	return len(d.missingImports) == 0 && len(d.missingAssets) == 0 && (catalogComplete || !d.compositionDep)
}

// pruneUnsatisfiableTests keeps generated tests whose imports, assets and catalog
// assumptions are satisfied by the selected project.
func pruneUnsatisfiableTests(root, module string, assets []string, catalogComplete bool) ([]string, error) {
	pkgs := map[string]bool{}
	var testRels []string
	if err := filepath.WalkDir(root, func(file string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if entry.Name() == ".git" {
				return fs.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(entry.Name(), ".go") {
			return nil
		}
		rel, err := filepath.Rel(root, file)
		if err != nil {
			return err
		}
		if strings.HasSuffix(entry.Name(), "_test.go") {
			testRels = append(testRels, filepath.ToSlash(rel))
		} else {
			pkgs[filepath.ToSlash(filepath.Dir(rel))] = true
		}
		return nil
	}); err != nil {
		return nil, err
	}
	slices.Sort(testRels)
	prefix := module + "/"
	absent := missingAssets(assets)
	deps := make(map[string]testDeps, len(testRels))
	type groupKey struct{ dir, pkg string }
	groups := map[groupKey][]string{}
	for _, rel := range testRels {
		dep, err := analyzeTestFile(filepath.Join(root, filepath.FromSlash(rel)), prefix, pkgs, absent)
		if err != nil {
			return nil, err
		}
		deps[rel] = dep
		key := groupKey{dir: filepath.ToSlash(filepath.Dir(rel)), pkg: dep.pkg}
		groups[key] = append(groups[key], rel)
	}
	drop := map[string]bool{}
	for _, key := range slices.SortedFunc(maps.Keys(groups), func(a, b groupKey) int {
		if a.dir != b.dir {
			return strings.Compare(a.dir, b.dir)
		}
		return strings.Compare(a.pkg, b.pkg)
	}) {
		files := groups[key]
		var unsatisfied []string
		droppedDecls := map[string]bool{}
		for _, rel := range files {
			if deps[rel].satisfied(catalogComplete) {
				continue
			}
			unsatisfied = append(unsatisfied, rel)
			for name := range deps[rel].decls {
				droppedDecls[name] = true
			}
		}
		if len(unsatisfied) == 0 {
			continue
		}
		cascade := false
		for _, rel := range files {
			if !deps[rel].satisfied(catalogComplete) {
				continue
			}
			for name := range deps[rel].refs {
				if droppedDecls[name] {
					cascade = true
					break
				}
			}
			if cascade {
				break
			}
		}
		if cascade {
			for _, rel := range files {
				drop[rel] = true
			}
		} else {
			for _, rel := range unsatisfied {
				drop[rel] = true
			}
		}
	}
	discarded := make([]string, 0, len(drop))
	for _, rel := range slices.Sorted(maps.Keys(drop)) {
		if err := os.Remove(filepath.Join(root, filepath.FromSlash(rel))); err != nil {
			return nil, fmt.Errorf("remove unsatisfiable test %s: %w", rel, err)
		}
		discarded = append(discarded, rel)
	}
	if err := assertNoProductionGap(root, prefix, pkgs); err != nil {
		return nil, err
	}
	return discarded, nil
}

func assertNoProductionGap(root, prefix string, pkgs map[string]bool) error {
	return filepath.WalkDir(root, func(file string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if entry.Name() == ".git" {
				return fs.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
			return nil
		}
		missing, err := missingModuleImports(file, prefix, pkgs)
		if err != nil {
			return err
		}
		if len(missing) > 0 {
			return fmt.Errorf("复制集缺口：生产文件 %s import %s，但生成树里没有该包", filepathSlash(file), strings.Join(missing, ", "))
		}
		return nil
	})
}

func analyzeTestFile(file, prefix string, pkgs map[string]bool, absent []string) (testDeps, error) {
	parsed, err := parser.ParseFile(token.NewFileSet(), file, nil, parser.SkipObjectResolution)
	if err != nil {
		return testDeps{}, fmt.Errorf("parse %s: %w", filepathSlash(file), err)
	}
	d := testDeps{pkg: parsed.Name.Name, decls: map[string]bool{}, refs: map[string]bool{}}
	for _, spec := range parsed.Imports {
		name := strings.Trim(spec.Path.Value, `"`)
		if name == prefix+compositionManifestDir {
			d.compositionDep = true
			continue
		}
		if rel, ok := strings.CutPrefix(name, prefix); ok && !pkgs[rel] {
			d.missingImports = append(d.missingImports, name)
		}
	}
	d.missingAssets = assetRefsOf(parsed, absent)
	for _, decl := range parsed.Decls {
		switch value := decl.(type) {
		case *ast.FuncDecl:
			if value.Recv == nil {
				d.decls[value.Name.Name] = true
			}
		case *ast.GenDecl:
			for _, spec := range value.Specs {
				switch item := spec.(type) {
				case *ast.ValueSpec:
					for _, name := range item.Names {
						recordDecl(&d, name.Name)
					}
				case *ast.TypeSpec:
					recordDecl(&d, item.Name.Name)
				}
			}
		}
	}
	ast.Inspect(parsed, func(node ast.Node) bool {
		if ident, ok := node.(*ast.Ident); ok && ident.Name != "_" {
			d.refs[ident.Name] = true
		}
		return true
	})
	return d, nil
}

func recordDecl(d *testDeps, name string) {
	if name != "_" {
		d.decls[name] = true
	}
}

func missingAssets(copied []string) []string {
	have := make(map[string]bool, len(copied))
	for _, value := range copied {
		have[profileassets.Canonical(value)] = true
	}
	var missing []string
	for _, paths := range profileassets.Declared() {
		for _, value := range paths {
			canonical := profileassets.Canonical(value)
			if canonical != "" && !have[canonical] {
				missing = append(missing, canonical)
			}
		}
	}
	slices.Sort(missing)
	return slices.Compact(missing)
}

func missingAssetRefs(file string, absent []string) ([]string, error) {
	if len(absent) == 0 {
		return nil, nil
	}
	parsed, err := parser.ParseFile(token.NewFileSet(), file, nil, parser.SkipObjectResolution)
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", filepathSlash(file), err)
	}
	return assetRefsOf(parsed, absent), nil
}

func assetRefsOf(parsed *ast.File, absent []string) []string {
	if len(absent) == 0 {
		return nil
	}
	found := map[string]bool{}
	record := func(literal string) {
		clean := cleanAssetLiteral(literal)
		for _, prefix := range absent {
			if clean == prefix || strings.HasPrefix(clean, prefix+"/") || strings.HasSuffix(clean, "/"+prefix) || strings.Contains(clean, "/"+prefix+"/") {
				found[prefix] = true
			}
		}
	}
	ast.Inspect(parsed, func(node ast.Node) bool {
		switch value := node.(type) {
		case *ast.BasicLit:
			if value.Kind == token.STRING {
				if text, err := strconv.Unquote(value.Value); err == nil {
					record(text)
				}
			}
		case *ast.CallExpr:
			parts := make([]string, 0, len(value.Args))
			for _, arg := range value.Args {
				literal, ok := arg.(*ast.BasicLit)
				if !ok || literal.Kind != token.STRING {
					parts = nil
					break
				}
				text, err := strconv.Unquote(literal.Value)
				if err != nil {
					parts = nil
					break
				}
				parts = append(parts, text)
			}
			if len(parts) > 0 {
				record(strings.Join(parts, "/"))
			}
		}
		return true
	})
	return slices.Sorted(maps.Keys(found))
}

func cleanAssetLiteral(literal string) string {
	value := strings.TrimSpace(filepath.ToSlash(literal))
	for strings.HasPrefix(value, "./") || strings.HasPrefix(value, "../") {
		if strings.HasPrefix(value, "./") {
			value = value[2:]
		} else {
			value = value[3:]
		}
	}
	return path.Clean(value)
}

func missingModuleImports(file, prefix string, pkgs map[string]bool) ([]string, error) {
	parsed, err := parser.ParseFile(token.NewFileSet(), file, nil, parser.ImportsOnly)
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", filepathSlash(file), err)
	}
	var missing []string
	for _, spec := range parsed.Imports {
		name := strings.Trim(spec.Path.Value, `"`)
		if rel, ok := strings.CutPrefix(name, prefix); ok && !pkgs[rel] {
			missing = append(missing, name)
		}
	}
	return missing, nil
}

func resolveModule(dir, module string) (string, error) {
	if module != "" {
		return module, nil
	}
	base := filepath.Base(absPath(dir))
	if base == "" || base == "." || base == string(filepath.Separator) {
		return "", fmt.Errorf("cannot derive module path from %q; pass --module", dir)
	}
	return base, nil
}

func validateModule(module string) error {
	if module == "" {
		return fmt.Errorf("--module 不得为空")
	}
	if strings.Contains(module, "jimu/") || strings.HasSuffix(module, "/jimu") {
		return fmt.Errorf("invalid --module %q: 不得包含 jimu module path", module)
	}
	return nil
}

func swapIntoPlace(target, staging string) error {
	old := ""
	if _, err := os.Lstat(target); err == nil {
		suffix, err := randomSuffix()
		if err != nil {
			return err
		}
		old = target + ".old-" + suffix
		if err := os.Rename(target, old); err != nil {
			return fmt.Errorf("move existing target aside: %w", err)
		}
		defer func() { _ = os.RemoveAll(old) }()
	}
	if err := os.Rename(staging, target); err != nil {
		if old != "" {
			if restoreErr := os.Rename(old, target); restoreErr != nil {
				return fmt.Errorf("install new product: %w (also failed to restore previous target: %v)", err, restoreErr)
			}
		}
		return fmt.Errorf("install new product: %w", err)
	}
	return nil
}

func copyOneFile(root, dst, rel string) error {
	src := filepath.Join(root, filepath.FromSlash(rel))
	info, err := os.Stat(src)
	if err != nil {
		return fmt.Errorf("kernel file %s: %w", rel, err)
	}
	if info.IsDir() {
		return fmt.Errorf("kernel file %s is a directory", rel)
	}
	content, err := os.ReadFile(src)
	if err != nil {
		return fmt.Errorf("read kernel file %s: %w", rel, err)
	}
	target := filepath.Join(dst, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return fmt.Errorf("create directory for %s: %w", rel, err)
	}
	return os.WriteFile(target, content, info.Mode().Perm())
}

func flattenDrivers(drivers map[string][]string) []string {
	var out []string
	for name, list := range drivers {
		for _, driver := range list {
			out = append(out, name+":"+driver)
		}
	}
	sort.Strings(out)
	return out
}

func randomSuffix() (string, error) {
	var buf [4]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano()), nil
	}
	return hex.EncodeToString(buf[:]), nil
}

func absPath(value string) string {
	abs, err := filepath.Abs(value)
	if err != nil {
		return value
	}
	return abs
}

func filepathSlash(value string) string { return filepath.ToSlash(value) }
