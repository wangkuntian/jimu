package application

import (
	"context"
	stderrors "errors"
	"testing"
	"time"

	"jimu/internal/modules/audit/domain"
	apperrors "jimu/internal/shared/errors"
	"jimu/internal/shared/pagination"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestAuditServiceGetMapsNotFound(t *testing.T) {
	service := NewAuditService(&fakeAuditRepository{findErr: gorm.ErrRecordNotFound}, "")

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
	service := NewAuditService(repo, "")

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
	log      *domain.AuditLog
	logs     []domain.AuditLog
	total    int64
	findErr  error
	listErr  error
	offset   int
	limit    int
	sort     string
	order    string
	verify   []domain.AuditLog
	headHash string
}

func (r *fakeAuditRepository) Create(context.Context, *domain.AuditLog) error { return nil }
func (r *fakeAuditRepository) CreateBatch(context.Context, []domain.AuditLog) error {
	return nil
}
func (r *fakeAuditRepository) FindByID(context.Context, uint64) (*domain.AuditLog, error) {
	return r.log, r.findErr
}
func (r *fakeAuditRepository) List(_ context.Context, _ uint64, offset, limit int, sort, order string) ([]domain.AuditLog, int64, error) {
	r.offset = offset
	r.limit = limit
	r.sort = sort
	r.order = order
	return r.logs, r.total, r.listErr
}

func (r *fakeAuditRepository) ListForVerify(context.Context, uint64, uint64, uint64, int) ([]domain.AuditLog, error) {
	return r.verify, nil
}

func (r *fakeAuditRepository) ChainHead(context.Context, uint64) (string, error) {
	return r.headHash, nil
}

func auditAppCode(err error) int {
	var appErr *apperrors.AppError
	if stderrors.As(err, &appErr) {
		return appErr.Code
	}
	return 0
}

// chainedEntries 构造一条以 secret 链接好的审计链
func chainedEntries(secret string, actions ...string) []domain.AuditLog {
	entries := make([]domain.AuditLog, 0, len(actions))
	prev := ""
	for i, action := range actions {
		e := domain.AuditLog{
			ID:        uint64(i + 1),
			TenantID:  1,
			Username:  "alice",
			Action:    action,
			Path:      "/" + action,
			PrevHash:  prev,
			CreatedAt: time.Unix(1700000000+int64(i), 0),
		}
		e.EntryHash = e.ComputeEntryHash([]byte(secret))
		prev = e.EntryHash
		entries = append(entries, e)
	}
	return entries
}

func TestAuditServiceVerifyIntactChain(t *testing.T) {
	entries := chainedEntries("s3cret", "a", "b")
	svc := NewAuditService(&fakeAuditRepository{verify: entries, headHash: entries[1].EntryHash}, "s3cret")

	res, err := svc.Verify(context.Background(), 1, 0, 0, 100)
	require.NoError(t, err)
	assert.True(t, res.Intact)
	assert.True(t, res.TailIntact)
	assert.Equal(t, 2, res.Checked)
	assert.Zero(t, res.Unhashed)
}

func TestAuditServiceVerifyDetectsTamperedContent(t *testing.T) {
	entries := chainedEntries("s3cret", "a", "b")
	entries[1].Action = "tampered" // 改内容但不改哈希

	svc := NewAuditService(&fakeAuditRepository{verify: entries, headHash: entries[1].EntryHash}, "s3cret")
	res, err := svc.Verify(context.Background(), 1, 0, 0, 100)
	require.NoError(t, err)
	assert.False(t, res.Intact)
	assert.Equal(t, uint64(2), res.BrokenAt)
	assert.Contains(t, res.Reason, "entry_hash")
}

func TestAuditServiceVerifyDetectsBrokenLink(t *testing.T) {
	entries := chainedEntries("s3cret", "a", "b")
	// 断开链接但保持自身哈希自洽（模拟删除/替换中间条目）
	entries[1].PrevHash = "deadbeef"
	entries[1].EntryHash = entries[1].ComputeEntryHash([]byte("s3cret"))

	svc := NewAuditService(&fakeAuditRepository{verify: entries, headHash: entries[1].EntryHash}, "s3cret")
	res, err := svc.Verify(context.Background(), 1, 0, 0, 100)
	require.NoError(t, err)
	assert.False(t, res.Intact)
	assert.Equal(t, uint64(2), res.BrokenAt)
	assert.Contains(t, res.Reason, "prev_hash")
}

func TestAuditServiceVerifySkipsUnhashedPrefix(t *testing.T) {
	entries := chainedEntries("s3cret", "a")
	legacy := append([]domain.AuditLog{{ID: 0, TenantID: 1, Action: "legacy"}}, entries...)

	svc := NewAuditService(&fakeAuditRepository{verify: legacy, headHash: entries[0].EntryHash}, "s3cret")
	res, err := svc.Verify(context.Background(), 1, 0, 0, 100)
	require.NoError(t, err)
	assert.True(t, res.Intact, "未哈希存量条目应跳过而不判为断裂")
	assert.Equal(t, 1, res.Unhashed)
	assert.Equal(t, 1, res.Checked)
}

func TestAuditServiceVerifyDetectsTailTruncation(t *testing.T) {
	entries := chainedEntries("s3cret", "a", "b")
	// 链头指向的哈希与扫描到的最后一条不一致 → 说明末尾条目被删除
	svc := NewAuditService(&fakeAuditRepository{verify: entries[:1], headHash: entries[1].EntryHash}, "s3cret")

	res, err := svc.Verify(context.Background(), 1, 0, 0, 100)
	require.NoError(t, err)
	assert.True(t, res.Intact)
	assert.False(t, res.TailIntact, "应检测到链尾被截断")
	assert.Equal(t, entries[1].EntryHash, res.HeadHash)
	assert.Equal(t, entries[0].EntryHash, res.LastHash)
}
