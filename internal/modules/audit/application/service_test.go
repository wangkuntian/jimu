package application

import (
	"context"
	stderrors "errors"
	"testing"

	"jimu/internal/modules/audit/domain"
	apperrors "jimu/internal/shared/errors"
	"jimu/internal/shared/pagination"

	"gorm.io/gorm"
)

func TestAuditServiceGetMapsNotFound(t *testing.T) {
	service := NewAuditService(&fakeAuditRepository{findErr: gorm.ErrRecordNotFound})

	_, err := service.Get(context.Background(), 9)
	if auditAppCode(err) != apperrors.CodeNotFound {
		t.Fatalf("code = %d, want %d", auditAppCode(err), apperrors.CodeNotFound)
	}
}

func TestAuditServiceListReturnsDTOAndPassesPagination(t *testing.T) {
	repo := &fakeAuditRepository{
		logs:  []domain.AuditLog{{ID: 1, Username: "alice", Action: "create"}},
		total: 6,
	}
	service := NewAuditService(repo)

	logs, total, err := service.List(context.Background(), pagination.Pagination{Page: 2, PageSize: 5, Sort: "created_at", Order: "asc"})
	if err != nil {
		t.Fatal(err)
	}
	if repo.offset != 5 || repo.limit != 5 || repo.sort != "created_at" || repo.order != "asc" {
		t.Fatalf("pagination = offset:%d limit:%d sort:%q order:%q", repo.offset, repo.limit, repo.sort, repo.order)
	}
	if total != 6 || len(logs) != 1 || logs[0].Username != "alice" {
		t.Fatalf("logs = %#v total = %d", logs, total)
	}
}

type fakeAuditRepository struct {
	log     *domain.AuditLog
	logs    []domain.AuditLog
	total   int64
	findErr error
	listErr error
	offset  int
	limit   int
	sort    string
	order   string
}

func (r *fakeAuditRepository) Create(context.Context, *domain.AuditLog) error { return nil }
func (r *fakeAuditRepository) CreateBatch(context.Context, []domain.AuditLog) error {
	return nil
}
func (r *fakeAuditRepository) FindByID(context.Context, uint64) (*domain.AuditLog, error) {
	return r.log, r.findErr
}
func (r *fakeAuditRepository) List(_ context.Context, offset, limit int, sort, order string) ([]domain.AuditLog, int64, error) {
	r.offset = offset
	r.limit = limit
	r.sort = sort
	r.order = order
	return r.logs, r.total, r.listErr
}

func auditAppCode(err error) int {
	var appErr *apperrors.AppError
	if stderrors.As(err, &appErr) {
		return appErr.Code
	}
	return 0
}
