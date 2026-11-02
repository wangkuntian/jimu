package manifest

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"sort"
)

// Load 从磁盘读取并严格校验 manifest。文件必须包含与内容匹配的 digest。
func Load(path string) (Document, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return Document{}, fmt.Errorf("read manifest %s: %w", path, err)
	}
	decoder := json.NewDecoder(bytes.NewReader(b))
	decoder.DisallowUnknownFields()
	var doc Document
	if err := decoder.Decode(&doc); err != nil {
		return Document{}, fmt.Errorf("decode manifest: %w", err)
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return Document{}, fmt.Errorf("decode manifest: multiple JSON values")
		}
		return Document{}, fmt.Errorf("decode manifest: %w", err)
	}
	if doc.SchemaVersion == 0 {
		return Document{}, fmt.Errorf("schema_version is required")
	}
	switch doc.SchemaVersion {
	case CurrentSchemaVersion:
		if err := validatePersistedFields(b, CurrentSchemaVersion); err != nil {
			return Document{}, err
		}
		if err := ValidateComplete(doc); err != nil {
			return Document{}, err
		}
		if err := validateDigest(doc); err != nil {
			return Document{}, err
		}
		return doc, nil
	case LegacySchemaVersion:
		if err := validatePersistedFields(b, LegacySchemaVersion); err != nil {
			return Document{}, err
		}
		if err := validateCompleteLegacy(doc); err != nil {
			return Document{}, err
		}
		if err := validateDigestLegacy(doc); err != nil {
			return Document{}, err
		}
		return upgradeFromV2(doc)
	default:
		return Document{}, fmt.Errorf("unsupported schema version %d", doc.SchemaVersion)
	}
}

func validateDigest(doc Document) error {
	if doc.Digest == "" {
		return fmt.Errorf("manifest digest is required")
	}
	expected, err := Digest(doc)
	if err != nil {
		return err
	}
	if doc.Digest != expected {
		return fmt.Errorf("manifest digest mismatch: got %q, want %q", doc.Digest, expected)
	}
	return nil
}

func validateDigestLegacy(doc Document) error {
	if doc.Digest == "" {
		return fmt.Errorf("manifest digest is required")
	}
	expected, err := digestLegacyV2(doc)
	if err != nil {
		return err
	}
	if doc.Digest != expected {
		return fmt.Errorf("manifest digest mismatch: got %q, want %q", doc.Digest, expected)
	}
	return nil
}

func validatePersistedFields(content []byte, schemaVersion int) error {
	var root map[string]json.RawMessage
	if err := json.Unmarshal(content, &root); err != nil {
		return fmt.Errorf("decode manifest fields: %w", err)
	}
	if schemaVersion == LegacySchemaVersion {
		if _, ok := root["composition"]; ok {
			return fmt.Errorf("composition is not valid in schema v2")
		}
	}
	if err := requireJSONField(root, "generated_file_sha256"); err != nil {
		return err
	}
	var framework map[string]json.RawMessage
	if err := json.Unmarshal(root["framework"], &framework); err != nil {
		return fmt.Errorf("decode framework fields: %w", err)
	}
	if err := requireJSONField(framework, "dirty"); err != nil {
		return fmt.Errorf("framework.%w", err)
	}
	var selection map[string]json.RawMessage
	if err := json.Unmarshal(root["selection"], &selection); err != nil {
		return fmt.Errorf("decode selection fields: %w", err)
	}
	if err := requireJSONField(selection, "template_version"); err != nil {
		return fmt.Errorf("selection.%w", err)
	}
	if schemaVersion == CurrentSchemaVersion {
		if err := requireJSONField(root, "composition"); err != nil {
			return err
		}
		var composition map[string]json.RawMessage
		if err := json.Unmarshal(root["composition"], &composition); err != nil {
			return fmt.Errorf("decode composition fields: %w", err)
		}
		for _, field := range []string{"contract_version", "requested", "resolved", "auto_added", "drivers", "validity", "support"} {
			if err := requireJSONField(composition, field); err != nil {
				return fmt.Errorf("composition.%w", err)
			}
		}
	}
	return nil
}

func requireJSONField(fields map[string]json.RawMessage, name string) error {
	value, ok := fields[name]
	if !ok || bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
		return fmt.Errorf("%s is required", name)
	}
	return nil
}

// CanonicalJSON 返回不含尾部换行的确定性 JSON。map 键由 encoding/json 按字典序编码，
// 生成文件列表和 map 中的切片在这里统一排序。
func CanonicalJSON(doc Document) ([]byte, error) {
	if err := Validate(doc); err != nil {
		return nil, err
	}
	return marshalCanonical(doc)
}

func marshalCanonical(doc Document) ([]byte, error) {
	normalized := normalize(doc)
	b, err := json.Marshal(normalized)
	if err != nil {
		return nil, fmt.Errorf("encode canonical manifest: %w", err)
	}
	return b, nil
}

// Digest 计算不包含自身 digest 字段的 canonical JSON SHA-256。
func Digest(doc Document) (string, error) {
	doc.Digest = ""
	b, err := CanonicalJSON(doc)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(b)
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}

// digestLegacyV2 verifies a v2 manifest with the v2 JSON shape: no
// composition field and schema_version 2.
func digestLegacyV2(doc Document) (string, error) {
	if err := validateLegacy(doc); err != nil {
		return "", err
	}
	doc.Digest = ""
	b, err := marshalCanonical(doc)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(b)
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}

// upgradeFromV2 converts a verified legacy manifest into the current schema in
// memory. Schema v2 has no explicit composition block, so its declared
// capability set is the resolved set and no dependency is known to be auto-added.
// v2 manifests could only be written after preflight, so validity is derived as
// valid; consumers that need current validity must re-run the resolver.
func upgradeFromV2(doc Document) (Document, error) {
	doc.SchemaVersion = CurrentSchemaVersion
	doc.Composition = &Composition{
		ContractVersion: CompositionContractVersion,
		Requested:       sortedClone(doc.Selection.Capabilities),
		Resolved:        sortedClone(doc.Selection.Capabilities),
		AutoAdded:       []string{},
		Drivers:         cloneDriverMap(doc.Selection.Drivers),
		Validity:        CompositionValidityValid,
		Support:         compositionSupportFromLegacy(doc.Selection.Support),
	}
	doc.Digest = ""
	digest, err := Digest(doc)
	if err != nil {
		return Document{}, err
	}
	doc.Digest = digest
	return doc, nil
}

func compositionSupportFromLegacy(support string) string {
	switch support {
	case "official":
		return CompositionSupportCertified
	case "experimental", "internal":
		return CompositionSupportExperimental
	default:
		return CompositionSupportExperimental
	}
}

// Write 计算 digest 后将 manifest 写入目标路径。
func Write(path string, doc Document) error {
	if err := ValidateComplete(doc); err != nil {
		return err
	}
	doc.Digest = ""
	digest, err := Digest(doc)
	if err != nil {
		return err
	}
	doc.Digest = digest
	b, err := CanonicalJSON(doc)
	if err != nil {
		return err
	}
	b = append(b, '\n')
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create manifest directory: %w", err)
	}
	if err := os.WriteFile(path, b, 0o644); err != nil {
		return fmt.Errorf("write manifest %s: %w", path, err)
	}
	return nil
}

func normalize(doc Document) Document {
	doc.Selection.Capabilities = nonNilClone(doc.Selection.Capabilities)
	doc.Selection.Drivers = cloneSortedDriverMap(doc.Selection.Drivers)
	if doc.Composition != nil {
		composition := *doc.Composition
		composition.Requested = sortedClone(composition.Requested)
		composition.Resolved = sortedClone(composition.Resolved)
		composition.AutoAdded = sortedClone(composition.AutoAdded)
		composition.Drivers = cloneSortedDriverMap(composition.Drivers)
		doc.Composition = &composition
	}
	doc.Capabilities = nonNilClone(doc.Capabilities)
	for i := range doc.Capabilities {
		doc.Capabilities[i].Requires = sortedClone(doc.Capabilities[i].Requires)
		doc.Capabilities[i].SoftRequires = sortedClone(doc.Capabilities[i].SoftRequires)
		doc.Capabilities[i].Owns = sortedClone(doc.Capabilities[i].Owns)
		doc.Capabilities[i].Configs = sortedClone(doc.Capabilities[i].Configs)
		doc.Capabilities[i].Permissions = nonNilClone(doc.Capabilities[i].Permissions)
		doc.Capabilities[i].Migrations = sortedClone(doc.Capabilities[i].Migrations)
		doc.Capabilities[i].Drivers = sortedClone(doc.Capabilities[i].Drivers)
		doc.Capabilities[i].Assets = sortedClone(doc.Capabilities[i].Assets)
		doc.Capabilities[i].Permissions = nonNilClone(doc.Capabilities[i].Permissions)
	}
	doc.Copy = nonNilClone(doc.Copy)
	for i := range doc.Copy {
		doc.Copy[i].Include = sortedClone(doc.Copy[i].Include)
		doc.Copy[i].Exclude = sortedClone(doc.Copy[i].Exclude)
	}
	doc.Templates = nonNilClone(doc.Templates)
	for i := range doc.Templates {
		if doc.Templates[i].Data == nil {
			doc.Templates[i].Data = map[string]any{}
		}
	}
	doc.Merges = nonNilClone(doc.Merges)
	for i := range doc.Merges {
		doc.Merges[i].Sections = sortedClone(doc.Merges[i].Sections)
	}
	doc.Rewrites = nonNilClone(doc.Rewrites)
	for i := range doc.Rewrites {
		doc.Rewrites[i].Files = sortedClone(doc.Rewrites[i].Files)
	}
	doc.Assets = nonNilClone(doc.Assets)
	for i := range doc.Assets {
		doc.Assets[i].Include = sortedClone(doc.Assets[i].Include)
		doc.Assets[i].Exclude = sortedClone(doc.Assets[i].Exclude)
	}
	doc.Prune = nonNilClone(doc.Prune)
	doc.GeneratedFiles = sortedClone(doc.GeneratedFiles)
	if doc.GeneratedFileSHA256 == nil {
		doc.GeneratedFileSHA256 = map[string]string{}
	}
	doc.Report.Capabilities = nonNilClone(doc.Report.Capabilities)
	doc.Report.Migrations = sortedClone(doc.Report.Migrations)
	doc.Report.Tables = sortedClone(doc.Report.Tables)
	doc.Report.HeavyDeps = sortedClone(doc.Report.HeavyDeps)
	return doc
}

func cloneDriverMap(in map[string][]string) map[string][]string {
	out := make(map[string][]string, len(in))
	for name, values := range in {
		out[name] = nonNilClone(values)
	}
	return out
}

func cloneSortedDriverMap(in map[string][]string) map[string][]string {
	out := make(map[string][]string, len(in))
	for name, values := range in {
		out[name] = slices.Sorted(slices.Values(values))
	}
	return out
}

func nonNilClone[T any](values []T) []T {
	if values == nil {
		return []T{}
	}
	return slices.Clone(values)
}

func sortedClone(values []string) []string {
	if values == nil {
		return []string{}
	}
	out := slices.Clone(values)
	sort.Strings(out)
	return out
}
