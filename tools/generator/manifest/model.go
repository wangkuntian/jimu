package manifest

// CurrentSchemaVersion 是当前生成器支持的 manifest schema 版本。
const CurrentSchemaVersion = 1

// Document 是生成器使用的完整 JSON manifest。
type Document struct {
	SchemaVersion  int              `json:"schema_version"`
	Framework      Framework        `json:"framework"`
	Selection      Selection        `json:"selection"`
	Capabilities   []Capability     `json:"capabilities"`
	Copy           []CopyAction     `json:"copy"`
	Templates      []TemplateAction `json:"templates"`
	Merges         []MergeAction    `json:"merges"`
	Rewrites       []RewriteAction  `json:"rewrites"`
	Assets         []AssetAction    `json:"assets"`
	Prune          []PruneRule      `json:"prune"`
	Report         ReportSpec       `json:"report"`
	GeneratedFiles []string         `json:"generated_files"`
	Digest         string           `json:"digest"`
}

type Framework struct {
	Module  string `json:"module"`
	Version string `json:"version"`
	Commit  string `json:"commit"`
}

type Selection struct {
	Shape        string              `json:"shape"`
	Profile      string              `json:"profile"`
	Capabilities []string            `json:"capabilities"`
	Drivers      map[string][]string `json:"drivers"`
}

type Capability struct {
	Name          string       `json:"name"`
	Requires      []string     `json:"requires"`
	SoftRequires  []string     `json:"soft_requires"`
	MigrationOnly bool         `json:"migration_only"`
	DomainOnly    bool         `json:"domain_only"`
	Owns          []string     `json:"owns"`
	Configs       []string     `json:"configs"`
	Permissions   []Permission `json:"permissions"`
	Mount         string       `json:"mount"`
	Migrations    []string     `json:"migrations"`
	Drivers       []string     `json:"drivers"`
	Assets        []string     `json:"assets"`
}

type CopyAction struct {
	Source      string   `json:"source"`
	Destination string   `json:"destination"`
	Optional    bool     `json:"optional"`
	Include     []string `json:"include"`
	Exclude     []string `json:"exclude"`
}

type Permission struct {
	Name     string `json:"name"`
	Resource string `json:"resource"`
	Action   string `json:"action"`
}

type TemplateAction struct {
	Source      string `json:"source"`
	Destination string `json:"destination"`
	Kind        string `json:"kind"`
	// Data is a JSON object passed to the template. Templates use both scalar
	// values and structured collections, so the manifest keeps the payload
	// deliberately template-oriented instead of restricting it to strings.
	Data any `json:"data"`
}

type MergeAction struct {
	Source      string   `json:"source"`
	Destination string   `json:"destination"`
	Strategy    string   `json:"strategy"`
	Sections    []string `json:"sections"`
}

type RewriteAction struct {
	Kind  string   `json:"kind"`
	From  string   `json:"from"`
	To    string   `json:"to"`
	Files []string `json:"files"`
}

type AssetAction struct {
	Source      string   `json:"source"`
	Destination string   `json:"destination"`
	Include     []string `json:"include"`
	Exclude     []string `json:"exclude"`
}

type PruneRule struct {
	Kind   string `json:"kind"`
	Path   string `json:"path"`
	Reason string `json:"reason"`
}

type ReportSpec struct {
	Name         string   `json:"name"`
	Capabilities []string `json:"capabilities"`
	Routes       int      `json:"routes"`
	Migrations   []string `json:"migrations"`
	Tables       []string `json:"tables"`
	HeavyDeps    []string `json:"heavy_deps"`
}
