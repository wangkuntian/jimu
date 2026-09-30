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
