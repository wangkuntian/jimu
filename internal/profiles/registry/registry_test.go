package registry

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNamesIsTheSupportedProfilesInReportOrder(t *testing.T) {
	assert.Equal(t, []string{"full", "minimal"}, Names())
}

func TestAllCoversEveryName(t *testing.T) {
	all := All()
	require.Len(t, all, len(Names()))
	for _, n := range Names() {
		a, ok := all[n]
		require.True(t, ok, "All() 缺少形态 %s", n)
		assert.Equal(t, n, a.Name)
	}
}

func TestLookupRejectsUnknownProfile(t *testing.T) {
	_, err := Lookup("ghost")
	require.ErrorContains(t, err, `unknown profile "ghost"`)
	require.ErrorContains(t, err, "full, minimal")
	if _, err := Lookup("minimal"); err != nil {
		t.Fatalf("Lookup(minimal) error: %v", err)
	}
}
