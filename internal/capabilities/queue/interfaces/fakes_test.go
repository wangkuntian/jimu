package interfaces

import (
	"context"

	"jimu/internal/capabilities/queue/domain"
)

// fakeJobRepo 可配置的任务仓储 mock
type fakeJobRepo struct {
	create   func(ctx context.Context, job *domain.Job) error
	findByID func(ctx context.Context, id uint64) (*domain.Job, error)
	update   func(ctx context.Context, job *domain.Job) error
	list     func(ctx context.Context, tenantID uint64, offset, limit int, filters map[string]interface{}) ([]domain.Job, int64, error)
}

func (f *fakeJobRepo) Create(ctx context.Context, job *domain.Job) error {
	if f.create != nil {
		return f.create(ctx, job)
	}
	job.ID = 5
	return nil
}

func (f *fakeJobRepo) FindByID(ctx context.Context, id uint64) (*domain.Job, error) {
	if f.findByID != nil {
		return f.findByID(ctx, id)
	}
	return &domain.Job{ID: id, Type: "email", Status: domain.JobStatusPending}, nil
}

func (f *fakeJobRepo) Update(ctx context.Context, job *domain.Job) error {
	if f.update != nil {
		return f.update(ctx, job)
	}
	return nil
}

func (f *fakeJobRepo) List(ctx context.Context, tenantID uint64, offset, limit int, filters map[string]interface{}) ([]domain.Job, int64, error) {
	if f.list != nil {
		return f.list(ctx, tenantID, offset, limit, filters)
	}
	return []domain.Job{{ID: 1, Type: "email"}}, 1, nil
}

// fakeDeadLetterRepo 可配置的死信仓储 mock
type fakeDeadLetterRepo struct {
	list         func(ctx context.Context, tenantID uint64, offset, limit int, resolved bool) ([]domain.DeadLetter, int64, error)
	markResolved func(ctx context.Context, tenantID uint64, id uint64) error
}

func (f *fakeDeadLetterRepo) Create(ctx context.Context, d *domain.DeadLetter) error { return nil }

func (f *fakeDeadLetterRepo) List(ctx context.Context, tenantID uint64, offset, limit int, resolved bool) ([]domain.DeadLetter, int64, error) {
	if f.list != nil {
		return f.list(ctx, tenantID, offset, limit, resolved)
	}
	return []domain.DeadLetter{{ID: 1, JobID: 1}}, 1, nil
}

func (f *fakeDeadLetterRepo) MarkResolved(ctx context.Context, tenantID uint64, id uint64) error {
	if f.markResolved != nil {
		return f.markResolved(ctx, tenantID, id)
	}
	return nil
}
