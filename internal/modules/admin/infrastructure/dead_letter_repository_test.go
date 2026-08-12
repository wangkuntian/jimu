package infrastructure

import (
	"context"
	"testing"

	"jimu/internal/platform/queue/domain"

	"github.com/stretchr/testify/assert"
)

func TestMysqlDeadLetterRepositoryCRUD(t *testing.T) {
	db := newHistoryTestDB(t)
	repo := NewMysqlDeadLetterRepository(db)
	ctx := context.Background()

	err := repo.Create(ctx, &domain.DeadLetter{JobID: 1, Type: "outbox:user.created", Payload: "{}", FailReason: "boom"})
	assert.NoError(t, err)
	err = repo.Create(ctx, &domain.DeadLetter{JobID: 2, Type: "outbox:user.updated", Payload: "{}", FailReason: "boom", Resolved: true})
	assert.NoError(t, err)

	unresolved, total, err := repo.List(ctx, 0, 10, false)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Len(t, unresolved, 1)
	assert.Equal(t, "outbox:user.created", unresolved[0].Type)

	resolved, total, err := repo.List(ctx, 0, 10, true)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Len(t, resolved, 1)

	assert.NoError(t, repo.MarkResolved(ctx, 1))
	after, total, err := repo.List(ctx, 0, 10, false)
	assert.NoError(t, err)
	assert.Equal(t, int64(0), total)
	assert.Len(t, after, 0)
}
