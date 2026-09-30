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
	if err := Validate(doc); err != nil {
		return Document{}, err
	}
	if doc.Digest == "" {
		return Document{}, fmt.Errorf("manifest digest is required")
	}
	expected, err := Digest(doc)
	if err != nil {
		return Document{}, err
	}
	if doc.Digest != expected {
		return Document{}, fmt.Errorf("manifest digest mismatch: got %q, want %q", doc.Digest, expected)
	}
	return doc, nil
}

// CanonicalJSON 返回不含尾部换行的确定性 JSON。map 键由 encoding/json 按字典序编码，
// 生成文件列表和 map 中的切片在这里统一排序。
func CanonicalJSON(doc Document) ([]byte, error) {
	if err := Validate(doc); err != nil {
		return nil, err
	}
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

// Write 计算 digest 后将 manifest 写入目标路径。
func Write(path string, doc Document) error {
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
	if doc.Selection.Drivers == nil {
		doc.Selection.Drivers = map[string][]string{}
	} else {
		drivers := make(map[string][]string, len(doc.Selection.Drivers))
		for name, selected := range doc.Selection.Drivers {
			drivers[name] = slices.Sorted(slices.Values(selected))
		}
		doc.Selection.Drivers = drivers
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
	doc.Report.Capabilities = nonNilClone(doc.Report.Capabilities)
	doc.Report.Migrations = sortedClone(doc.Report.Migrations)
	doc.Report.Tables = sortedClone(doc.Report.Tables)
	doc.Report.HeavyDeps = sortedClone(doc.Report.HeavyDeps)
	return doc
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
