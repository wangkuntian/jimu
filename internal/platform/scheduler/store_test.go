package scheduler

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMemoryStore(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()

	job := JobDef{ID: "job1", Name: "Test", Cron: "@every 1m", Enabled: true}
	assert.NoError(t, store.Save(ctx, job))

	list, err := store.List(ctx)
	assert.NoError(t, err)
	assert.Len(t, list, 1)
	assert.Equal(t, "job1", list[0].ID)

	assert.NoError(t, store.Delete(ctx, "job1"))
	list, err = store.List(ctx)
	assert.NoError(t, err)
	assert.Len(t, list, 0)
}
