package manifest

import (
	"encoding/hex"
	"fmt"
	"path"
	"path/filepath"
	"sort"
	"strings"
)

// Validate 检查 manifest 的 schema、路径和动作目标不变量。
func Validate(doc Document) error {
	if doc.SchemaVersion == 0 {
		return fmt.Errorf("schema_version is required")
	}
	if doc.SchemaVersion != CurrentSchemaVersion {
		return fmt.Errorf("unsupported schema version %d", doc.SchemaVersion)
	}
	if strings.TrimSpace(doc.Framework.Module) == "" {
		return fmt.Errorf("framework.module is required")
	}
	if strings.TrimSpace(doc.Selection.Shape) == "" {
		return fmt.Errorf("selection.shape is required")
	}
	switch doc.Selection.Support {
	case "", "official", "experimental", "internal":
	default:
		return fmt.Errorf("selection.support has unknown value %q", doc.Selection.Support)
	}
	if err := validateNames("selection.capabilities", doc.Selection.Capabilities); err != nil {
		return err
	}
	if err := validateCapabilities(doc.Capabilities); err != nil {
		return err
	}

	usedDestinations := map[string]string{}
	for i, action := range doc.Copy {
		if err := validateActionPath(fmt.Sprintf("copy[%d].source", i), action.Source); err != nil {
			return err
		}
		if err := validateActionPath(fmt.Sprintf("copy[%d].destination", i), action.Destination); err != nil {
			return err
		}
		for j, include := range action.Include {
			if err := validateActionPath(fmt.Sprintf("copy[%d].include[%d]", i, j), include); err != nil {
				return err
			}
		}
		for j, exclude := range action.Exclude {
			if exclude == "*_test.go" {
				continue
			}
			if err := validateActionPath(fmt.Sprintf("copy[%d].exclude[%d]", i, j), exclude); err != nil {
				return err
			}
		}
		if err := reserveDestination(usedDestinations, action.Destination, "copy"); err != nil {
			return err
		}
	}
	for i, action := range doc.Templates {
		if err := validateActionPath(fmt.Sprintf("templates[%d].source", i), action.Source); err != nil {
			return err
		}
		if err := validateActionPath(fmt.Sprintf("templates[%d].destination", i), action.Destination); err != nil {
			return err
		}
		if err := reserveDestination(usedDestinations, action.Destination, "templates"); err != nil {
			return err
		}
	}
	for i, action := range doc.Merges {
		if err := validateActionPath(fmt.Sprintf("merges[%d].source", i), action.Source); err != nil {
			return err
		}
		if err := validateActionPath(fmt.Sprintf("merges[%d].destination", i), action.Destination); err != nil {
			return err
		}
	}
	for i, action := range doc.Rewrites {
		if strings.TrimSpace(action.Kind) == "" {
			return fmt.Errorf("rewrites[%d].kind is required", i)
		}
		if strings.TrimSpace(action.From) == "" {
			return fmt.Errorf("rewrites[%d].from is required", i)
		}
		if strings.TrimSpace(action.To) == "" {
			return fmt.Errorf("rewrites[%d].to is required", i)
		}
		for j, file := range action.Files {
			if err := validateActionPath(fmt.Sprintf("rewrites[%d].files[%d]", i, j), file); err != nil {
				return err
			}
		}
	}
	for i, action := range doc.Assets {
		if err := validateActionPath(fmt.Sprintf("assets[%d].source", i), action.Source); err != nil {
			return err
		}
		if err := validateActionPath(fmt.Sprintf("assets[%d].destination", i), action.Destination); err != nil {
			return err
		}
		if err := reserveDestination(usedDestinations, action.Destination, "assets"); err != nil {
			return err
		}
		for j, pattern := range append(append([]string{}, action.Include...), action.Exclude...) {
			if err := validateActionPath(fmt.Sprintf("assets[%d].patterns[%d]", i, j), pattern); err != nil {
				return err
			}
		}
	}
	for i, rule := range doc.Prune {
		if strings.TrimSpace(rule.Kind) == "" {
			return fmt.Errorf("prune[%d].kind is required", i)
		}
		if err := validateActionPath(fmt.Sprintf("prune[%d].path", i), rule.Path); err != nil {
			return err
		}
	}
	if err := validateUniquePaths("generated_files", doc.GeneratedFiles); err != nil {
		return err
	}
	for i, file := range doc.GeneratedFiles {
		if err := validateActionPath(fmt.Sprintf("generated_files[%d]", i), file); err != nil {
			return err
		}
	}
	generated := make(map[string]bool, len(doc.GeneratedFiles))
	for _, file := range doc.GeneratedFiles {
		generated[file] = true
	}
	hashPaths := make([]string, 0, len(doc.GeneratedFileSHA256))
	for file := range doc.GeneratedFileSHA256 {
		hashPaths = append(hashPaths, file)
	}
	sort.Strings(hashPaths)
	for _, file := range hashPaths {
		if err := validateActionPath("generated_file_sha256", file); err != nil {
			return err
		}
		if !generated[file] {
			return fmt.Errorf("generated_file_sha256 path %q is not listed in generated_files", file)
		}
		if err := validateSHA256("generated_file_sha256["+file+"]", doc.GeneratedFileSHA256[file]); err != nil {
			return err
		}
	}
	if doc.Digest != "" {
		if err := validateSHA256("digest", doc.Digest); err != nil {
			return err
		}
	}
	return nil
}

// ValidateComplete verifies the provenance fields required in a persisted manifest.
func ValidateComplete(doc Document) error {
	if err := Validate(doc); err != nil {
		return err
	}
	if strings.TrimSpace(doc.Framework.Version) == "" {
		return fmt.Errorf("framework.version is required")
	}
	if strings.TrimSpace(doc.Framework.Commit) == "" {
		return fmt.Errorf("framework.commit is required")
	}
	if doc.Selection.Template != "" && strings.TrimSpace(doc.Selection.TemplateVersion) == "" {
		return fmt.Errorf("selection.template_version is required for template %q", doc.Selection.Template)
	}
	for _, file := range doc.GeneratedFiles {
		if _, ok := doc.GeneratedFileSHA256[file]; !ok {
			return fmt.Errorf("missing generated file hash for %q", file)
		}
	}
	return nil
}

func validateCapabilities(capabilities []Capability) error {
	seen := map[string]bool{}
	for i, capability := range capabilities {
		if strings.TrimSpace(capability.Name) == "" {
			return fmt.Errorf("capabilities[%d].name is required", i)
		}
		if seen[capability.Name] {
			return fmt.Errorf("duplicate capability %q", capability.Name)
		}
		seen[capability.Name] = true
		for field, names := range map[string][]string{
			"requires":              capability.Requires,
			"soft_requires":         capability.SoftRequires,
			"drivers":               capability.Drivers,
			"migration_schema_deps": capability.MigrationSchemaDeps,
			"runtime_services":      capability.RuntimeServices,
		} {
			if err := validateNames(fmt.Sprintf("capabilities[%d].%s", i, field), names); err != nil {
				return err
			}
		}
		switch capability.ProtectedHTTPMiddleware {
		case "", "primary", "fallback":
		default:
			return fmt.Errorf("capabilities[%d].protected middleware has unknown mode %q", i, capability.ProtectedHTTPMiddleware)
		}
		for _, dependency := range capability.MigrationSchemaDeps {
			if dependency == capability.Name {
				return fmt.Errorf("capabilities[%d].migration_schema_deps references itself", i)
			}
		}
		for field, paths := range map[string][]string{
			"configs": capability.Configs, "migrations": capability.Migrations, "assets": capability.Assets,
		} {
			for j, value := range paths {
				if err := validateActionPath(fmt.Sprintf("capabilities[%d].%s[%d]", i, field, j), value); err != nil {
					return err
				}
			}
		}
		for j, permission := range capability.Permissions {
			if strings.TrimSpace(permission.Name) == "" && strings.TrimSpace(permission.Resource) == "" {
				return fmt.Errorf("capabilities[%d].permissions[%d] must have name or resource", i, j)
			}
		}
	}
	return nil
}

func validateNames(field string, values []string) error {
	seen := map[string]bool{}
	for i, value := range values {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("%s[%d] must not be empty", field, i)
		}
		if seen[value] {
			return fmt.Errorf("duplicate value %q in %s", value, field)
		}
		seen[value] = true
	}
	return nil
}

func validateUniquePaths(field string, values []string) error {
	seen := map[string]bool{}
	for _, value := range values {
		if seen[value] {
			return fmt.Errorf("duplicate path %q in %s", value, field)
		}
		seen[value] = true
	}
	return nil
}

func validateActionPath(field, value string) error {
	if value == "" {
		return fmt.Errorf("%s is required", field)
	}
	if strings.IndexByte(value, 0) >= 0 {
		return fmt.Errorf("%s contains NUL", field)
	}
	if filepath.IsAbs(value) || path.IsAbs(value) || strings.HasPrefix(value, "\\") || isWindowsAbs(value) {
		return fmt.Errorf("%s must be relative, got absolute path %q", field, value)
	}
	if strings.Contains(value, "\\") {
		return fmt.Errorf("%s must use slash-separated relative paths", field)
	}
	for _, segment := range strings.Split(value, "/") {
		if segment == ".." {
			return fmt.Errorf("%s contains forbidden .. path segment", field)
		}
	}
	if path.Clean(value) == "." {
		return fmt.Errorf("%s must not be the current directory", field)
	}
	return nil
}

func validateSHA256(field, value string) error {
	if !strings.HasPrefix(value, "sha256:") {
		return fmt.Errorf("invalid %s %q: want sha256:<64 hex characters>", field, value)
	}
	hexPart := strings.TrimPrefix(value, "sha256:")
	if len(hexPart) != 64 {
		return fmt.Errorf("invalid %s %q: want sha256:<64 hex characters>", field, value)
	}
	if _, err := hex.DecodeString(hexPart); err != nil {
		return fmt.Errorf("invalid %s %q: %w", field, value, err)
	}
	return nil
}

func isWindowsAbs(value string) bool {
	return len(value) >= 3 && ((value[0] >= 'a' && value[0] <= 'z') || (value[0] >= 'A' && value[0] <= 'Z')) && value[1] == ':' && (value[2] == '/' || value[2] == '\\')
}

func reserveDestination(used map[string]string, destination, kind string) error {
	if previous, ok := used[destination]; ok {
		return fmt.Errorf("duplicate destination %q in %s and %s", destination, previous, kind)
	}
	used[destination] = kind
	return nil
}
