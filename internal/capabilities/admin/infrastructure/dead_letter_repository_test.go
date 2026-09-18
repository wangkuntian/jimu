package infrastructure

import (
	"context"
	"testing"

	"jimu/internal/capabilities/queue/domain"

	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

func TestMysqlDeadLetterRepositoryCRUD(t *testing.T) {
	db := newHistoryTestDB(t)
	repo := NewMysqlDeadLetterRepository(db)
	ctx := context.Background()

	err := repo.Create(ctx, &domain.DeadLetter{JobID: 1, Type: "outbox:user.created", Payload: "{}", FailReason: "boom"})
	assert.NoError(t, err)
	err = repo.Create(ctx, &domain.DeadLetter{JobID: 2, Type: "outbox:user.updated", Payload: "{}", FailReason: "boom", Resolved: true})
	assert.NoError(t, err)

	unresolved, total, err := repo.List(ctx, 0, 0, 10, false)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Len(t, unresolved, 1)
	assert.Equal(t, "outbox:user.created", unresolved[0].Type)

	resolved, total, err := repo.List(ctx, 0, 0, 10, true)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Len(t, resolved, 1)

	assert.NoError(t, repo.MarkResolved(ctx, 0, 1))
	after, total, err := repo.List(ctx, 0, 0, 10, false)
	assert.NoError(t, err)
	assert.Equal(t, int64(0), total)
	assert.Len(t, after, 0)
}

func TestMysqlDeadLetterRepositoryTenantIsolation(t *testing.T) {
	db := newHistoryTestDB(t)
	repo := NewMysqlDeadLetterRepository(db)
	ctx := context.Background()

	assert.NoError(t, repo.Create(ctx, &domain.DeadLetter{TenantID: 1, JobID: 1, Type: "a", FailReason: "boom"}))
	assert.NoError(t, repo.Create(ctx, &domain.DeadLetter{TenantID: 2, JobID: 2, Type: "b", FailReason: "boom"}))

	// List 按租户过滤
	letters, total, err := repo.List(ctx, 1, 0, 10, false)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Len(t, letters, 1)
	assert.Equal(t, "a", letters[0].Type)

	// 跨租户处理：不更新并返回 not found
	assert.ErrorIs(t, repo.MarkResolved(ctx, 1, 2), gorm.ErrRecordNotFound)
	_, total, err = repo.List(ctx, 2, 0, 10, false)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), total)

	// 同租户处理成功
	assert.NoError(t, repo.MarkResolved(ctx, 2, 2))
	_, total, err = repo.List(ctx, 2, 0, 10, false)
	assert.NoError(t, err)
	assert.Equal(t, int64(0), total)
}
