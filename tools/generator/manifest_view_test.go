package generator

import (
	"slices"

	"jimu/internal/capabilities/catalog"
	"jimu/tools/generator/workspace"
)

// Marker is a test-only view over the v0.3.3 manifest. It keeps selection
// assertions compact while the filesystem contract remains manifest-only.
type Marker struct {
	Generator    string
	Version      string
	SourceRoot   string
	SourceCommit string
	Module       string
	Shape        string
	Profile      string
	Capabilities []string
	DomainOnly   []string
	Drivers      map[string][]string
	Assets       []string
	Files        []string
}

const markerFile = ".jimu/manifest.json"

func LoadMarker(dir string) (*Marker, error) {
	doc, err := workspace.LoadProjectManifest(dir)
	if err != nil {
		return nil, err
	}
	module := ""
	for _, rewrite := range doc.Rewrites {
		if rewrite.Kind == "module" {
			module = rewrite.To
			break
		}
	}
	domainOnly := make([]string, 0)
	for _, capability := range doc.Capabilities {
		if capability.DomainOnly {
			domainOnly = append(domainOnly, capability.Name)
		}
	}
	assets := make([]string, 0, len(doc.Assets))
	for _, asset := range doc.Assets {
		assets = append(assets, asset.Destination)
	}
	slices.Sort(domainOnly)
	slices.Sort(assets)
	return &Marker{
		Generator:    "jimu new",
		Version:      doc.Framework.Version,
		SourceRoot:   FrameworkRoot(),
		SourceCommit: doc.Framework.Commit,
		Module:       module,
		Shape:        doc.Selection.Shape,
		Profile:      doc.Selection.Profile,
		Capabilities: slices.Clone(doc.Selection.Capabilities),
		DomainOnly:   domainOnly,
		Drivers:      cloneDriverMap(doc.Selection.Drivers),
		Assets:       assets,
		Files:        slices.Clone(doc.GeneratedFiles),
	}, nil
}

func cloneDriverMap(in map[string][]string) map[string][]string {
	out := make(map[string][]string, len(in))
	for name, values := range in {
		out[name] = slices.Clone(values)
	}
	return out
}

func declaredOrder(existing []string, name string, inCatalog map[string]bool) (declared, ungated []string, err error) {
	names := make(map[string]bool, len(existing)+1)
	for _, value := range existing {
		names[value] = true
	}
	names[name] = true
	emitted := map[string]bool{}
	emit := func(value string) {
		if emitted[value] {
			return
		}
		emitted[value] = true
		declared = append(declared, value)
	}
	for _, descriptor := range catalog.All() {
		if inCatalog[descriptor.Name] && names[descriptor.Name] {
			emit(descriptor.Name)
		}
	}
	for _, value := range existing {
		if !inCatalog[value] {
			emit(value)
		}
	}
	if !inCatalog[name] && !slices.Contains(existing, name) {
		emit(name)
	}
	for _, value := range declared {
		if !inCatalog[value] {
			ungated = append(ungated, value)
		}
	}
	return declared, ungated, nil
}
