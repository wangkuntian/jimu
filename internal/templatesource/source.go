package templatesource

import "jimu/tools/generator/manifest"

// Source is a resolved template input root and its provenance.
type Source struct {
	Root      string
	Framework *manifest.Framework
}

// Resolver resolves an optional explicit source root. The returned cleanup
// must be called after the command finishes with the source.
type Resolver func(explicitRoot string) (Source, func() error, error)
