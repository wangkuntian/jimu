package application

import (
	"context"
	"encoding/json"
	stderrors "errors"
	"fmt"

	"jimu/internal/capabilities/audit/domain"
	"jimu/internal/kernel/tenant"
	"jimu/internal/shared/errors"
	"jimu/internal/shared/pagination"

	"gorm.io/gorm"
)

type AuditService struct {
	repo   domain.AuditRepository
	secret []byte // 审计链 HMAC 密钥（空则按 SHA-256 校验）
}

func NewAuditService(repo domain.AuditRepository, hashSecret string) *AuditService {
	return &AuditService{repo: repo, secret: []byte(hashSecret)}
}

// Record 记录审计日志
func (s *AuditService) Record(ctx context.Context, log domain.AuditLog) error {
	return s.repo.Create(ctx, serializeChanges(&log))
}

// serializeChanges 将 Changes 序列化到 ChangesRaw 以便存储
func serializeChanges(log *domain.AuditLog) *domain.AuditLog {
	if len(log.Changes) > 0 {
		if raw, err := json.Marshal(log.Changes); err == nil {
			log.ChangesRaw = string(raw)
		}
	}
	return log
}

// RecordChange 记录带字段变更的审计日志（便捷方法）
func (s *AuditService) RecordChange(ctx context.Context, userID uint64, username, action, resource string, changes []domain.Change) error {
	return s.repo.Create(ctx, &domain.AuditLog{
		UserID:   userID,
		Username: username,
		Action:   action,
		Resource: resource,
		Detail:   fmt.Sprintf("%d field(s) changed", len(changes)),
		Changes:  changes,
	})
}

func (s *AuditService) Get(ctx context.Context, id uint64) (*AuditLogResponse, error) {
	log, err := s.repo.FindByID(ctx, id)
	if err != nil {
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.Wrap(errors.CodeNotFound, "audit log not found", err)
		}
		return nil, errors.Wrap(errors.CodeInternalError, "failed to get audit log", err)
	}
	resp := ToAuditLogResponse(*log)
	return &resp, nil
}

// List 查询审计日志。上下文带租户时仅返回该租户的日志（0=平台级视角，不过滤）。
func (s *AuditService) List(ctx context.Context, p pagination.Pagination) ([]AuditLogResponse, int64, error) {
	logs, total, err := s.repo.List(ctx, tenant.FromContext(ctx), p.GetOffset(), p.GetLimit(), p.Sort, p.Order)
	if err != nil {
		return nil, 0, errors.Wrap(errors.CodeInternalError, "failed to list audit logs", err)
	}
	return ToAuditLogResponses(logs), total, nil
}

// VerifyResult 审计链校验结果
type VerifyResult struct {
	Checked    int    `json:"checked"`             // 参与校验的已哈希条目数
	Unhashed   int    `json:"unhashed"`            // 迁移前未哈希的存量条目数
	Intact     bool   `json:"intact"`              // 链是否完整
	BrokenAt   uint64 `json:"broken_at,omitempty"` // 首个异常条目 ID
	Reason     string `json:"reason,omitempty"`    // 异常原因
	TailIntact bool   `json:"tail_intact"`         // 链尾是否与链头一致（检测末尾条目被删）
	HeadHash   string `json:"head_hash,omitempty"` // 链头记录的哈希
	LastHash   string `json:"last_hash,omitempty"` // 扫描到的最后一条哈希
}

// Verify 校验审计链完整性：按 id 升序重算每条哈希并检查与上一条的衔接。
// tenantID=0 表示平台级视角（跨租户，按租户分别跟踪链）。
// fromID/toID 可缩小校验范围；全量扫描且未截断时才做链尾截断检测。
func (s *AuditService) Verify(ctx context.Context, tenantID, fromID, toID uint64, limit int) (*VerifyResult, error) {
	logs, err := s.repo.ListForVerify(ctx, tenantID, fromID, toID, limit)
	if err != nil {
		return nil, errors.Wrap(errors.CodeInternalError, "failed to load audit logs", err)
	}

	res := &VerifyResult{Intact: true, TailIntact: true}
	expected := map[uint64]string{}
	lastHash := map[uint64]string{}

	for i := range logs {
		l := &logs[i]
		if l.EntryHash == "" {
			// 迁移前的历史条目未参与哈希，跳过校验（链从其后第一条开始）
			res.Unhashed++
			continue
		}
		if l.PrevHash != expected[l.TenantID] {
			res.Intact = false
			res.BrokenAt = l.ID
			res.Reason = "prev_hash mismatch: entry is not linked to the previous record"
			break
		}
		if !l.MatchesEntryHash(s.secret) {
			res.Intact = false
			res.BrokenAt = l.ID
			res.Reason = "entry_hash mismatch: record content was modified"
			break
		}
		expected[l.TenantID] = l.EntryHash
		lastHash[l.TenantID] = l.EntryHash
		res.Checked++
	}

	// 链尾检测：仅在单租户全量扫描且未被 limit 截断时进行
	if tenantID != 0 && fromID == 0 && toID == 0 && len(logs) < limit {
		head, err := s.repo.ChainHead(ctx, tenantID)
		if err != nil {
			return nil, errors.Wrap(errors.CodeInternalError, "failed to load audit chain head", err)
		}
		res.HeadHash = head
		res.LastHash = lastHash[tenantID]
		res.TailIntact = head == res.LastHash
	}
	return res, nil
}
