package application

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	stderrors "errors"
	"strings"
	"testing"
	"time"

	"jimu/internal/capabilities/audit/domain"
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

func (r *fakeAuditRepository) CountRange(context.Context, uint64, time.Time, time.Time) (int64, error) {
	return r.total, r.listErr
}

func (r *fakeAuditRepository) ListRange(_ context.Context, _ uint64, _, _ time.Time, offset, limit int) ([]domain.AuditLog, error) {
	// 模拟分页：按 offset/limit 切分 logs
	if offset >= len(r.logs) {
		return nil, r.listErr
	}
	end := offset + limit
	if end > len(r.logs) {
		end = len(r.logs)
	}
	return r.logs[offset:end], r.listErr
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

func TestAuditExportCSV(t *testing.T) {
	repo := &fakeAuditRepository{
		logs: []domain.AuditLog{
			{ID: 1, TenantID: 1, Username: "alice", Action: "login", Path: "/api/v1/auth/login", Status: 200,
				CreatedAt: time.Unix(1700000000, 0), PrevHash: "", EntryHash: "h1"},
			{ID: 2, TenantID: 1, Username: "bob", Action: "create_user", Path: "/api/v1/admin/users", Status: 201,
				CreatedAt: time.Unix(1700000010, 0), PrevHash: "h1", EntryHash: "h2"},
		},
		total: 2,
	}
	svc := NewAuditService(repo, "secret")

	var buf bytes.Buffer
	summary, err := svc.Export(context.Background(), ExportOptions{
		Format: ExportFormatCSV,
		Start:  time.Unix(1699990000, 0),
		End:    time.Unix(1700001000, 0),
	}, &buf)
	if err != nil {
		t.Fatalf("Export() error: %v", err)
	}
	if summary.Rows != 2 || summary.Format != ExportFormatCSV {
		t.Fatalf("summary = %+v", summary)
	}

	rows, err := csv.NewReader(&buf).ReadAll()
	if err != nil {
		t.Fatalf("csv parse: %v", err)
	}
	if len(rows) != 3 {
		t.Fatalf("csv rows = %d, want header + 2", len(rows))
	}
	if rows[0][0] != "id" || rows[0][5] != "action" {
		t.Fatalf("unexpected header: %v", rows[0])
	}
	if rows[1][4] != "alice" || rows[2][5] != "create_user" {
		t.Fatalf("unexpected rows: %v", rows)
	}
	if !strings.HasPrefix(rows[1][1], "2023-11-14T") {
		t.Fatalf("created_at 应为 UTC RFC3339: %v", rows[1][1])
	}
}

func TestAuditExportJSON(t *testing.T) {
	repo := &fakeAuditRepository{
		logs:  []domain.AuditLog{{ID: 1, TenantID: 1, Username: "alice", Action: "login", CreatedAt: time.Unix(1700000000, 0)}},
		total: 1,
	}
	svc := NewAuditService(repo, "secret")

	var buf bytes.Buffer
	summary, err := svc.Export(context.Background(), ExportOptions{Format: ExportFormatJSON}, &buf)
	if err != nil {
		t.Fatalf("Export() error: %v", err)
	}
	if summary.Rows != 1 {
		t.Fatalf("rows = %d", summary.Rows)
	}
	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != 1 {
		t.Fatalf("ndjson lines = %d", len(lines))
	}
	var entry map[string]any
	if err := json.Unmarshal([]byte(lines[0]), &entry); err != nil {
		t.Fatalf("ndjson parse: %v", err)
	}
	if entry["username"] != "alice" {
		t.Fatalf("entry = %v", entry)
	}
}

func TestAuditExportRejectsInvalidOptions(t *testing.T) {
	svc := NewAuditService(&fakeAuditRepository{total: 1}, "secret")
	ctx := context.Background()

	tests := []struct {
		name string
		opts ExportOptions
	}{
		{"未知格式", ExportOptions{Format: "xml"}},
		{"开始晚于结束", ExportOptions{Start: time.Unix(2000000000, 0), End: time.Unix(1000000000, 0)}},
		{"跨度过大", ExportOptions{Start: time.Now().AddDate(0, 0, -120), End: time.Now()}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := svc.Export(ctx, tt.opts, &bytes.Buffer{}); auditAppCode(err) != apperrors.CodeInvalidParam {
				t.Fatalf("code = %d, want %d", auditAppCode(err), apperrors.CodeInvalidParam)
			}
		})
	}

	// 条数超上限
	big := NewAuditService(&fakeAuditRepository{total: exportMaxRows + 1}, "secret")
	if _, err := big.Export(ctx, ExportOptions{}, &bytes.Buffer{}); auditAppCode(err) != apperrors.CodeInvalidParam {
		t.Fatalf("超出条目上限应返回参数错误, got %d", auditAppCode(err))
	}

	// writer 缺失
	if _, err := svc.Export(ctx, ExportOptions{}, nil); auditAppCode(err) != apperrors.CodeInternalError {
		t.Fatalf("缺少 writer 应返回内部错误, got %d", auditAppCode(err))
	}
}

// failingRangeRepo 让范围查询报错，覆盖导出的错误分支
type failingRangeRepo struct {
	fakeAuditRepository
	countErr error
	listErr2 error
}

func (r *failingRangeRepo) CountRange(context.Context, uint64, time.Time, time.Time) (int64, error) {
	return 0, r.countErr
}

func (r *failingRangeRepo) ListRange(context.Context, uint64, time.Time, time.Time, int, int) ([]domain.AuditLog, error) {
	return nil, r.listErr2
}

func TestAuditExportRepoErrors(t *testing.T) {
	ctx := context.Background()

	// 统计失败 → 内部错误
	svc := NewAuditService(&failingRangeRepo{countErr: stderrors.New("count down")}, "secret")
	if _, err := svc.Export(ctx, ExportOptions{}, &bytes.Buffer{}); auditAppCode(err) != apperrors.CodeInternalError {
		t.Fatalf("code = %d, want %d", auditAppCode(err), apperrors.CodeInternalError)
	}

	// 分批读取失败 → 内部错误
	svc = NewAuditService(&failingRangeRepo{listErr2: stderrors.New("read down")}, "secret")
	if _, err := svc.Export(ctx, ExportOptions{}, &bytes.Buffer{}); auditAppCode(err) != apperrors.CodeInternalError {
		t.Fatalf("code = %d, want %d", auditAppCode(err), apperrors.CodeInternalError)
	}
}

// failingWriter 让写出失败，覆盖导出时的写错误分支
type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) { return 0, stderrors.New("write down") }

func TestAuditExportWriterErrors(t *testing.T) {
	repo := &fakeAuditRepository{
		logs:  []domain.AuditLog{{ID: 1, TenantID: 1, Username: "alice", CreatedAt: time.Unix(1700000000, 0)}},
		total: 1,
	}
	svc := NewAuditService(repo, "secret")

	for _, format := range []string{ExportFormatCSV, ExportFormatJSON} {
		_, err := svc.Export(context.Background(), ExportOptions{Format: format}, failingWriter{})
		if auditAppCode(err) != apperrors.CodeInternalError {
			t.Fatalf("format=%s code = %d, want %d", format, auditAppCode(err), apperrors.CodeInternalError)
		}
	}
}

func TestAuditExportDefaultsToLastSevenDays(t *testing.T) {
	repo := &fakeAuditRepository{total: 0}
	svc := NewAuditService(repo, "secret")

	summary, err := svc.Export(context.Background(), ExportOptions{}, &bytes.Buffer{})
	if err != nil {
		t.Fatalf("Export() error: %v", err)
	}
	span := summary.End.Sub(summary.Start)
	if span < 6*24*time.Hour || span > 8*24*time.Hour {
		t.Fatalf("默认时间跨度应约为 7 天，实际 %v", span)
	}
	if summary.Format != ExportFormatCSV {
		t.Fatalf("默认格式应为 csv，实际 %s", summary.Format)
	}
}
