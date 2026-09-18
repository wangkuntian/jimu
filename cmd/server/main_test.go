package main

import (
	"sort"
	"testing"

	"jimu/internal/capabilities/catalog"
)

func TestWiredCapabilitiesMatchCatalog(t *testing.T) {
	wired := append([]string(nil), wiredCapabilities...)
	known := catalog.Names()
	sort.Strings(wired)
	sort.Strings(known)
	if len(wired) != len(known) {
		t.Fatalf("wired %d capabilities (%v), catalog declares %d (%v)", len(wired), wired, len(known), known)
	}
	for i := range wired {
		if wired[i] != known[i] {
			t.Fatalf("wired[%d] = %q, catalog[%d] = %q", i, wired[i], i, known[i])
		}
	}
}

func TestWiredCapabilitiesHaveNoDuplicates(t *testing.T) {
	seen := map[string]bool{}
	for _, name := range wiredCapabilities {
		if seen[name] {
			t.Fatalf("duplicate wired capability %q", name)
		}
		seen[name] = true
	}
}
