package render

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"jimu/tools/generator/manifest"
)

func mergeConfig(sourceRoot, destinationRoot string, action manifest.MergeAction) error {
	source, err := safeJoin(sourceRoot, action.Source)
	if err != nil {
		return err
	}
	destination, err := safeJoin(destinationRoot, action.Destination)
	if err != nil {
		return err
	}
	sourceContent, err := os.ReadFile(source)
	if err != nil {
		return fmt.Errorf("read merge source %s: %w", action.Source, err)
	}
	if len(action.Sections) > 0 {
		sourceContent, err = filterTopLevelSections(sourceContent, action.Sections)
		if err != nil {
			return fmt.Errorf("filter merge source %s: %w", action.Source, err)
		}
	}
	destinationContent, err := os.ReadFile(destination)
	if os.IsNotExist(err) {
		destinationContent = nil
	} else if err != nil {
		return fmt.Errorf("read merge destination %s: %w", action.Destination, err)
	}
	merged := sourceContent
	if len(destinationContent) > 0 && action.Strategy == "sections" {
		merged = mergeTopLevelSections(destinationContent, sourceContent)
	}
	if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
		return err
	}
	return os.WriteFile(destination, merged, 0o644)
}

func renderSectionTemplate(sourceRoot, destinationRoot string, action manifest.TemplateAction) error {
	source, err := safeJoin(sourceRoot, action.Source)
	if err != nil {
		return err
	}
	destination, err := safeJoin(destinationRoot, action.Destination)
	if err != nil {
		return err
	}
	content, err := os.ReadFile(source)
	if err != nil {
		return fmt.Errorf("read section template %s: %w", action.Source, err)
	}
	sections, err := templateSections(action.Data)
	if err != nil {
		return err
	}
	filtered, err := filterTopLevelSections(content, sections)
	if err != nil {
		return fmt.Errorf("filter section template %s: %w", action.Source, err)
	}
	if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
		return err
	}
	return os.WriteFile(destination, filtered, 0o644)
}

func templateSections(data any) ([]string, error) {
	values, ok := data.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("section template data must be an object")
	}
	raw, ok := values["Sections"]
	if !ok {
		return nil, fmt.Errorf("section template data requires Sections")
	}
	switch items := raw.(type) {
	case []string:
		return items, nil
	case []any:
		sections := make([]string, 0, len(items))
		for _, item := range items {
			section, ok := item.(string)
			if !ok {
				return nil, fmt.Errorf("section template data Sections must contain strings")
			}
			sections = append(sections, section)
		}
		return sections, nil
	default:
		return nil, fmt.Errorf("section template data Sections must be an array")
	}
}

func filterTopLevelSections(content []byte, sections []string) ([]byte, error) {
	keys, blocks, err := orderedTopLevelBlocks(content)
	if err != nil {
		return nil, err
	}
	wanted := make(map[string]bool, len(sections))
	for _, section := range sections {
		wanted[section] = true
	}
	var out strings.Builder
	for i, key := range keys {
		if key == "" || wanted[key] {
			out.WriteString(blocks[i])
		}
	}
	return []byte(out.String()), nil
}

func orderedTopLevelBlocks(content []byte) ([]string, []string, error) {
	lines := strings.SplitAfter(string(content), "\n")
	keys := make([]string, 0)
	blocks := make([]string, 0)
	pending := strings.Builder{}
	current := -1
	for index, line := range lines {
		body := strings.TrimRight(line, "\r\n")
		trimmed := strings.TrimSpace(body)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			pending.WriteString(line)
			continue
		}
		if body[0] == ' ' || body[0] == '\t' {
			if current < 0 {
				return nil, nil, fmt.Errorf("section parser: line %d is indented before a top-level key", index+1)
			}
			blocks[current] += pending.String() + line
			pending.Reset()
			continue
		}
		key, _, ok := strings.Cut(body, ":")
		if !ok || strings.TrimSpace(key) == "" {
			return nil, nil, fmt.Errorf("section parser: line %d is not a top-level key", index+1)
		}
		keys = append(keys, strings.TrimSpace(key))
		blocks = append(blocks, pending.String()+line)
		pending.Reset()
		current = len(blocks) - 1
	}
	if pending.Len() > 0 {
		if current < 0 {
			keys = append(keys, "")
			blocks = append(blocks, pending.String())
		} else {
			blocks[current] += pending.String()
		}
	}
	return keys, blocks, nil
}

func mergeTopLevelSections(existing, source []byte) []byte {
	existingKeys := topLevelKeys(existing)
	var out strings.Builder
	out.Write(existing)
	for key, block := range topLevelBlocks(source) {
		if existingKeys[key] {
			continue
		}
		if out.Len() > 0 && !strings.HasSuffix(out.String(), "\n") {
			out.WriteByte('\n')
		}
		out.WriteString(block)
	}
	return []byte(out.String())
}

// MergeSections preserves existing top-level YAML sections and appends sections
// that are present in the newly rendered configuration.
func MergeSections(existing, source []byte) []byte {
	return mergeTopLevelSections(existing, source)
}

func topLevelKeys(content []byte) map[string]bool {
	keys := map[string]bool{}
	for key := range topLevelBlocks(content) {
		keys[key] = true
	}
	return keys
}

func topLevelBlocks(content []byte) map[string]string {
	lines := strings.SplitAfter(string(content), "\n")
	blocks := map[string]string{}
	current := ""
	var block strings.Builder
	flush := func() {
		if current != "" {
			blocks[current] = block.String()
		}
		block.Reset()
	}
	for _, line := range lines {
		trimmed := strings.TrimRight(line, "\r\n")
		if len(trimmed) > 0 && trimmed[0] != ' ' && trimmed[0] != '\t' && !strings.HasPrefix(trimmed, "#") {
			if index := strings.IndexByte(trimmed, ':'); index > 0 {
				flush()
				current = strings.TrimSpace(trimmed[:index])
			}
		}
		block.WriteString(line)
	}
	flush()
	return blocks
}
