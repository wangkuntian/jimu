package generator

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestScaffoldShardIndicesCoverAllCasesOnce(t *testing.T) {
	for _, tc := range []struct {
		name   string
		cases  int
		shards int
	}{
		{name: "capability builds", cases: 25, shards: 4},
		{name: "generated test trees", cases: 10, shards: 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			seen := make(map[int]bool)
			for shard := 1; shard <= tc.shards; shard++ {
				indices, err := scaffoldShardIndices(fmt.Sprintf("%d/%d", shard, tc.shards), tc.cases)
				require.NoError(t, err)
				require.NotEmpty(t, indices)
				for _, index := range indices {
					assert.False(t, seen[index], "case %d assigned more than once", index)
					seen[index] = true
				}
			}
			assert.Len(t, seen, tc.cases)
		})
	}
}

func TestScaffoldShardIndicesWithoutSpecRunsAll(t *testing.T) {
	indices, err := scaffoldShardIndices("", 3)
	require.NoError(t, err)
	assert.Equal(t, []int{0, 1, 2}, indices)
}

func TestScaffoldShardIndicesRejectsInvalidSpec(t *testing.T) {
	for _, spec := range []string{"0/4", "5/4", "1/0", "1/2/3", "x/4", "1/x", "1/4"} {
		t.Run(spec, func(t *testing.T) {
			count := 4
			if spec == "1/4" {
				count = 3
			}
			_, err := scaffoldShardIndices(spec, count)
			require.Error(t, err)
		})
	}
}
