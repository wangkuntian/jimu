package manifest

// Schema versions understood by the generator. New manifests always use the
// current version; the legacy version is read-only and exists for upgrades.
const (
	LegacySchemaVersion  = 2
	CurrentSchemaVersion = 3
)

// TemplateContractVersion is the official template contract implemented by this generator.
const TemplateContractVersion = "v0.1"

// CompositionContractVersion is the capability composition contract persisted
// in schema v3 manifests.
const CompositionContractVersion = "v0.1"

// Composition validity and support values mirror internal/contract's
// composition dimensions without making the manifest package depend on it.
const (
	CompositionValidityValid   = "valid"
	CompositionValidityInvalid = "invalid"

	CompositionSupportCertified    = "certified"
	CompositionSupportValidated    = "validated"
	CompositionSupportExperimental = "experimental"
)

// Document 是生成器使用的完整 JSON manifest。
type Document struct {
	SchemaVersion       int               `json:"schema_version"`
	Framework           Framework         `json:"framework"`
	Selection           Selection         `json:"selection"`
	Composition         *Composition      `json:"composition,omitempty"`
	Capabilities        []Capability      `json:"capabilities"`
	Copy                []CopyAction      `json:"copy"`
	Templates           []TemplateAction  `json:"templates"`
	Merges              []MergeAction     `json:"merges"`
	Rewrites            []RewriteAction   `json:"rewrites"`
	Assets              []AssetAction     `json:"assets"`
	Prune               []PruneRule       `json:"prune"`
	Report              ReportSpec        `json:"report"`
	GeneratedFiles      []string          `json:"generated_files"`
	GeneratedFileSHA256 map[string]string `json:"generated_file_sha256"`
	Digest              string            `json:"digest"`
}

type Framework struct {
	Module  string `json:"module"`
	Version string `json:"version"`
	Commit  string `json:"commit"`
	Dirty   bool   `json:"dirty"`
}

type Selection struct {
	Shape           string              `json:"shape"`
	Template        string              `json:"template"`
	TemplateVersion string              `json:"template_version"`
	Profile         string              `json:"profile"`
	Support         string              `json:"support"`
	Capabilities    []string            `json:"capabilities"`
	Drivers         map[string][]string `json:"drivers"`
}

// Composition is the persisted result of capability resolution. It is the
// fact source for upgrades and diagnostics; schema v2 manifests are upgraded
// in memory when loaded and carry a synthesized composition.
type Composition struct {
	ContractVersion string              `json:"contract_version"`
	Requested       []string            `json:"requested"`
	Resolved        []string            `json:"resolved"`
	AutoAdded       []string            `json:"auto_added"`
	Drivers         map[string][]string `json:"drivers"`
	Validity        string              `json:"validity"`
	Support         string              `json:"support"`
}

type Capability struct {
	Name                    string       `json:"name"`
	Requires                []string     `json:"requires"`
	SoftRequires            []string     `json:"soft_requires"`
	MigrationOnly           bool         `json:"migration_only"`
	DomainOnly              bool         `json:"domain_only"`
	Owns                    []string     `json:"owns"`
	Configs                 []string     `json:"configs"`
	Permissions             []Permission `json:"permissions"`
	Mount                   string       `json:"mount"`
	Migrations              []string     `json:"migrations"`
	Drivers                 []string     `json:"drivers"`
	Assets                  []string     `json:"assets"`
	ProtectedHTTPMiddleware string       `json:"protected_http_middleware"`
	MigrationSchemaDeps     []string     `json:"migration_schema_deps"`
	RuntimeServices         []string     `json:"runtime_services"`
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
