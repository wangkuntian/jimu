//go:build jimu_nodb

package profiles

import "jimu/internal/assembly"

// StructuralSeed is a no-op when the build has no database kernel.
func StructuralSeed(*assembly.Context) error { return nil }
