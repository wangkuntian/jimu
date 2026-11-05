// Package upgrade classifies the three-way diff between a generated project's
// manifest baseline, its current files and the target template artifacts.
// It never writes to the project tree.
package upgrade

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"unicode/utf8"

	"jimu/tools/generator/manifest"

	"github.com/pmezard/go-difflib/difflib"
)

// Category is the upgrade decision for one generated file.
type Category string

const (
	// Untouched means local and target content still match the manifest baseline.
	Untouched Category = "untouched"
	// Modified means only the target template changed; installing it is safe.
	Modified Category = "modified"
	// Conflict means local ownership and the target template diverged; never overwrite.
	Conflict Category = "conflict"
	// New means the target template adds a file that does not exist locally.
	New Category = "new"
	// Deleted means the target template no longer generates a baseline file.
	Deleted Category = "deleted"
)

// Action is the installation decision that follows from a category.
type Action string

const (
	ActionNone    Action = "none"
	ActionInstall Action = "install"
	ActionKeep    Action = "keep"
)

// Status summarizes whether an upgrade is available and whether it is safe.
type Status string

const (
	StatusUpToDate         Status = "up-to-date"
	StatusUpgradeAvailable Status = "upgrade-available"
	StatusConflict         Status = "conflict"
)

// Entry is one path in the upgrade report. Hashes are omitted when the
// corresponding side does not exist.
type Entry struct {
	Path                     string   `json:"path"`
	Category                 Category `json:"category"`
	Action                   Action   `json:"action"`
	Reason                   string   `json:"reason"`
	BaselineExists           bool     `json:"baseline_exists"`
	LocalExists              bool     `json:"local_exists"`
	TargetExists             bool     `json:"target_exists"`
	BaselineContentAvailable bool     `json:"baseline_content_available"`
	BaselineSHA256           string   `json:"baseline_sha256,omitempty"`
	LocalSHA256              string   `json:"local_sha256,omitempty"`
	TargetSHA256             string   `json:"target_sha256,omitempty"`
	Diff                     string   `json:"diff,omitempty"`
	DiffUnavailableReason    string   `json:"diff_unavailable_reason,omitempty"`
	Binary                   bool     `json:"binary,omitempty"`
}

// Report is the deterministic, read-only result of an upgrade plan.
type Report struct {
	ProjectRoot string
	Entries     []Entry
	Counts      map[Category]int
}

// Options describes the three inputs of the diff.
type Options struct {
	ProjectRoot string
	Previous    manifest.Document
	TargetRoot  string
	TargetFiles []string
	// IncludeDiff materializes local-to-target text diffs for changed files.
	// The manifest contains hashes only, so baseline content is never available.
	IncludeDiff bool
	// Merged lists manifest merge destinations whose target content already
	// preserves local sections. User edits in those paths are not conflicts.
	Merged []string
}

var categoryOrder = []Category{Conflict, Modified, New, Deleted, Untouched}

// Classify compares the manifest baseline hashes, current project files and
// target template files. Output is grouped by category and sorted by path so
// repeated runs are byte-for-byte stable.
func Classify(opts Options) (Report, error) {
	if opts.ProjectRoot == "" || opts.TargetRoot == "" {
		return Report{}, fmt.Errorf("project root and target root are required")
	}
	projectRoot, err := filepath.Abs(opts.ProjectRoot)
	if err != nil {
		return Report{}, fmt.Errorf("resolve project root: %w", err)
	}
	targetRoot, err := filepath.Abs(opts.TargetRoot)
	if err != nil {
		return Report{}, fmt.Errorf("resolve target root: %w", err)
	}
	baseline, err := baselineHashes(opts.Previous)
	if err != nil {
		return Report{}, err
	}
	target, err := hashTargetFiles(targetRoot, opts.TargetFiles)
	if err != nil {
		return Report{}, err
	}
	paths := map[string]bool{}
	for rel := range baseline {
		paths[rel] = true
	}
	for rel := range target {
		paths[rel] = true
	}
	merged := make(map[string]bool, len(opts.Merged))
	for _, rel := range opts.Merged {
		merged[rel] = true
	}
	entries := make([]Entry, 0, len(paths))
	counts := zeroCounts()
	for _, rel := range sortedKeys(paths) {
		entry, err := classifyPath(projectRoot, rel, baseline, target, merged[rel])
		if err != nil {
			return Report{}, err
		}
		if opts.IncludeDiff {
			entry, err = addContentDiff(projectRoot, targetRoot, entry)
			if err != nil {
				return Report{}, err
			}
		}
		entries = append(entries, entry)
		counts[entry.Category]++
	}
	sort.SliceStable(entries, func(i, j int) bool {
		if entries[i].Category != entries[j].Category {
			return categoryRank(entries[i].Category) < categoryRank(entries[j].Category)
		}
		return entries[i].Path < entries[j].Path
	})
	return Report{ProjectRoot: projectRoot, Entries: entries, Counts: counts}, nil
}

// Status computes the overall upgrade state from the category counts.
func (r Report) Status() Status {
	if r.Counts[Conflict] > 0 {
		return StatusConflict
	}
	if r.Counts[Modified] > 0 || r.Counts[New] > 0 || r.Counts[Deleted] > 0 {
		return StatusUpgradeAvailable
	}
	return StatusUpToDate
}

// HasConflicts reports whether any entry needs manual resolution.
func (r Report) HasConflicts() bool {
	return r.Counts[Conflict] > 0
}

// Count returns the number of entries in a category.
func (r Report) Count(category Category) int {
	return r.Counts[category]
}

func baselineHashes(doc manifest.Document) (map[string]string, error) {
	hashes := make(map[string]string, len(doc.GeneratedFiles))
	for _, rel := range doc.GeneratedFiles {
		hash, ok := doc.GeneratedFileSHA256[rel]
		if !ok {
			return nil, fmt.Errorf("baseline manifest has no hash for %q", rel)
		}
		hashes[rel] = hash
	}
	return hashes, nil
}

func hashTargetFiles(root string, files []string) (map[string]string, error) {
	hashes := make(map[string]string, len(files))
	for _, rel := range files {
		if rel == "" {
			return nil, fmt.Errorf("target file path must not be empty")
		}
		path := filepath.Join(root, filepath.FromSlash(rel))
		info, err := os.Lstat(path)
		if err != nil {
			return nil, fmt.Errorf("inspect target file %s: %w", rel, err)
		}
		if !info.Mode().IsRegular() {
			return nil, fmt.Errorf("target file %s is not a regular file", rel)
		}
		hash, err := fileHash(path)
		if err != nil {
			return nil, fmt.Errorf("hash target file %s: %w", rel, err)
		}
		hashes[rel] = hash
	}
	return hashes, nil
}

type localFile struct {
	exists  bool
	regular bool
	hash    string
}

func inspectLocalFile(projectRoot, rel string) (localFile, error) {
	path := filepath.Join(projectRoot, filepath.FromSlash(rel))
	info, err := os.Lstat(path)
	if os.IsNotExist(err) {
		return localFile{}, nil
	}
	if err != nil {
		return localFile{}, fmt.Errorf("inspect local file %s: %w", rel, err)
	}
	if !info.Mode().IsRegular() {
		return localFile{exists: true}, nil
	}
	hash, err := fileHash(path)
	if err != nil {
		return localFile{}, fmt.Errorf("hash local file %s: %w", rel, err)
	}
	return localFile{exists: true, regular: true, hash: hash}, nil
}

func classifyPath(projectRoot, rel string, baseline, target map[string]string, merged bool) (Entry, error) {
	baseHash, hasBaseline := baseline[rel]
	targetHash, hasTarget := target[rel]
	local, err := inspectLocalFile(projectRoot, rel)
	if err != nil {
		return Entry{}, err
	}
	entry := Entry{
		Path:           rel,
		BaselineExists: hasBaseline,
		LocalExists:    local.exists,
		TargetExists:   hasTarget,
		BaselineSHA256: baseHash,
		TargetSHA256:   targetHash,
	}
	if local.regular {
		entry.LocalSHA256 = local.hash
	}
	if merged {
		return classifyMerged(entry, local, hasBaseline, hasTarget, baseHash, targetHash)
	}
	switch {
	case !hasBaseline:
		if !local.exists {
			entry.Category = New
			entry.Action = ActionInstall
			entry.Reason = "target template adds a file that is not part of the manifest baseline"
			return entry, nil
		}
		entry.Category = Conflict
		entry.Action = ActionKeep
		entry.Reason = "target template adds a file, but the path already exists locally and was not in the manifest baseline"
		return entry, nil
	case !local.exists:
		if !hasTarget {
			entry.Category = Deleted
			entry.Action = ActionNone
			entry.Reason = "target template no longer generates this file and it is already absent locally"
			return entry, nil
		}
		entry.Category = Conflict
		entry.Action = ActionKeep
		if targetHash == baseHash {
			entry.Reason = "generated file is missing locally; the target template still provides the baseline version"
		} else {
			entry.Reason = "generated file is missing locally and the target template changed"
		}
		return entry, nil
	case !local.regular:
		entry.Category = Conflict
		entry.Action = ActionKeep
		entry.Reason = "local path is not a regular file"
		return entry, nil
	case !hasTarget:
		if local.hash == baseHash {
			entry.Category = Deleted
			entry.Action = ActionKeep
			entry.Reason = "target template no longer generates this file; the existing copy is left in place"
			return entry, nil
		}
		entry.Category = Conflict
		entry.Action = ActionKeep
		entry.Reason = "target template no longer generates this file, but the local copy was modified"
		return entry, nil
	case local.hash == baseHash && targetHash == baseHash:
		entry.Category = Untouched
		entry.Action = ActionNone
		entry.Reason = "local file and target template match the manifest baseline"
		return entry, nil
	case local.hash == baseHash:
		entry.Category = Modified
		entry.Action = ActionInstall
		entry.Reason = "target template changed; the local file matches the manifest baseline"
		return entry, nil
	case local.hash == targetHash:
		entry.Category = Untouched
		entry.Action = ActionNone
		entry.Reason = "local file already matches the target template"
		return entry, nil
	case targetHash == baseHash:
		entry.Category = Conflict
		entry.Action = ActionKeep
		entry.Reason = "local file differs from the manifest baseline; the target template is unchanged"
		return entry, nil
	default:
		entry.Category = Conflict
		entry.Action = ActionKeep
		entry.Reason = "local file and target template both changed"
		return entry, nil
	}
}

func addContentDiff(projectRoot, targetRoot string, entry Entry) (Entry, error) {
	if entry.Category == Untouched {
		return entry, nil
	}
	var local []byte
	if entry.LocalExists {
		localPath := filepath.Join(projectRoot, filepath.FromSlash(entry.Path))
		info, err := os.Lstat(localPath)
		if err != nil {
			return Entry{}, fmt.Errorf("inspect local file %s for diff: %w", entry.Path, err)
		}
		if !info.Mode().IsRegular() {
			entry.DiffUnavailableReason = "local path is not a regular file"
			return entry, nil
		}
		local, err = os.ReadFile(localPath)
		if err != nil {
			return Entry{}, fmt.Errorf("read local file %s for diff: %w", entry.Path, err)
		}
	}
	var target []byte
	if entry.TargetExists {
		targetPath := filepath.Join(targetRoot, filepath.FromSlash(entry.Path))
		content, err := os.ReadFile(targetPath)
		if err != nil {
			return Entry{}, fmt.Errorf("read target file %s for diff: %w", entry.Path, err)
		}
		target = content
	}
	if isBinaryContent(local) || isBinaryContent(target) {
		entry.Binary = true
		entry.DiffUnavailableReason = "binary or invalid UTF-8 content"
		return entry, nil
	}
	if bytes.Equal(local, target) {
		return entry, nil
	}
	diff, err := difflib.GetUnifiedDiffString(difflib.UnifiedDiff{
		A:        difflib.SplitLines(string(local)),
		B:        difflib.SplitLines(string(target)),
		FromFile: "local/" + entry.Path,
		ToFile:   "target/" + entry.Path,
		Context:  3,
	})
	if err != nil {
		return Entry{}, fmt.Errorf("render diff for %s: %w", entry.Path, err)
	}
	entry.Diff = diff
	return entry, nil
}

func isBinaryContent(content []byte) bool {
	return bytes.IndexByte(content, 0) >= 0 || !utf8.Valid(content)
}

// classifyMerged handles manifest merge destinations. Their target content was
// rendered on top of the local file, so differences only mean new template
// sections are available and the merged content is safe to install.
func classifyMerged(entry Entry, local localFile, hasBaseline, hasTarget bool, baseHash, targetHash string) (Entry, error) {
	switch {
	case !hasBaseline:
		if !local.exists {
			entry.Category = New
			entry.Action = ActionInstall
			entry.Reason = "target template adds a merged config that is not part of the manifest baseline"
			return entry, nil
		}
		entry.Category = Conflict
		entry.Action = ActionKeep
		entry.Reason = "target template adds a merged config, but the path already exists locally and was not in the manifest baseline"
		return entry, nil
	case !local.exists:
		entry.Category = Conflict
		entry.Action = ActionKeep
		entry.Reason = "generated merged config is missing locally and cannot be merged without overwriting user intent"
		return entry, nil
	case !local.regular:
		entry.Category = Conflict
		entry.Action = ActionKeep
		entry.Reason = "local merged config path is not a regular file"
		return entry, nil
	case !hasTarget:
		entry.Category = Deleted
		entry.Action = ActionKeep
		entry.Reason = "target template no longer manages this merged config; the existing copy is left in place"
		return entry, nil
	case local.hash == targetHash:
		entry.Category = Untouched
		entry.Action = ActionNone
		entry.Reason = "local merged config already contains every target section"
		return entry, nil
	case local.hash == baseHash:
		entry.Category = Modified
		entry.Action = ActionInstall
		entry.Reason = "target merged config changed; the local file matches the manifest baseline"
		return entry, nil
	default:
		entry.Category = Modified
		entry.Action = ActionInstall
		entry.Reason = "target merged config adds sections while preserving the local file"
		return entry, nil
	}
}

// VerifyLocal re-hashes every path the report would install. It fails if the
// project changed after planning, so an upgrade never overwrites a concurrent
// edit that the plan did not classify.
func (r Report) VerifyLocal(projectRoot string) error {
	for _, entry := range r.Entries {
		if entry.Action != ActionInstall {
			continue
		}
		local, err := inspectLocalFile(projectRoot, entry.Path)
		if err != nil {
			return err
		}
		if local.exists != (entry.LocalSHA256 != "") || local.hash != entry.LocalSHA256 {
			return fmt.Errorf("project changed after upgrade planning: %s", entry.Path)
		}
	}
	return nil
}

func fileHash(path string) (string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(content)
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}

func zeroCounts() map[Category]int {
	return map[Category]int{
		Untouched: 0,
		Modified:  0,
		Conflict:  0,
		New:       0,
		Deleted:   0,
	}
}

func categoryRank(category Category) int {
	index := slices.Index(categoryOrder, category)
	if index < 0 {
		return len(categoryOrder)
	}
	return index
}

func sortedKeys(values map[string]bool) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

// CategoryOrder returns the stable display order used by the report.
func CategoryOrder() []Category {
	return slices.Clone(categoryOrder)
}
