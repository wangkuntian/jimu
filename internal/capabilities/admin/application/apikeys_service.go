package application

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"time"

	"jimu/internal/capabilities/admin/domain"
	"jimu/internal/platform/tenant"
	apperrors "jimu/internal/shared/errors"
)

const apiKeyPrefix = "jimu_"

// AdminAPIKeyService API Key 管理服务
type AdminAPIKeyService struct {
	repo  domain.APIKeyRepository
	quota TenantQuota // nil = 未启用租户配额
}

// WithQuota 注入租户配额校验（未注入时不做配额检查）
func (s *AdminAPIKeyService) WithQuota(quota TenantQuota) *AdminAPIKeyService {
	s.quota = quota
	return s
}

// NewAdminAPIKeyService 创建 API Key 管理服务
func NewAdminAPIKeyService(repo domain.APIKeyRepository) *AdminAPIKeyService {
	return &AdminAPIKeyService{repo: repo}
}

// tenantVisible 判断资源归属租户对上下文租户是否可见（见 platform/tenant.Visible）。
func tenantVisible(resourceTenant, ctxTenant uint64) bool {
	return tenant.Visible(resourceTenant, ctxTenant)
}

// ListKeys 获取 API Key 列表（按上下文租户过滤；0=平台级视角不过滤）
func (s *AdminAPIKeyService) ListKeys(ctx context.Context, offset, limit int) ([]domain.APIKey, int64, error) {
	return s.repo.List(ctx, tenant.FromContext(ctx), offset, limit)
}

// CreateKeyInput 创建 API Key 输入
type CreateKeyInput struct {
	Name      string
	Scopes    []string
	ExpiresIn int // days, 0 = no expiry
	CreatedBy uint64
}

// CreateKey 创建新 API Key（返回明文，仅此一次）
func (s *AdminAPIKeyService) CreateKey(ctx context.Context, input CreateKeyInput) (string, *domain.APIKey, error) {
	if input.Name == "" {
		return "", nil, apperrors.New(apperrors.CodeInvalidParam, "name is required")
	}
	if s.quota != nil {
		tenantID := tenant.FromContext(ctx)
		if tenantID == 0 {
			tenantID = tenant.DefaultTenantID
		}
		if err := s.quota.CheckAPIKeyQuota(ctx, tenantID); err != nil {
			return "", nil, err
		}
	}

	// Generate random key
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", nil, apperrors.Wrap(apperrors.CodeInternalError, "failed to generate key", err)
	}
	plaintext := apiKeyPrefix + hex.EncodeToString(raw)

	scopesJSON, err := json.Marshal(input.Scopes)
	if err != nil {
		return "", nil, apperrors.Wrap(apperrors.CodeInternalError, "failed to marshal scopes", err)
	}

	// 新 Key 归属创建者所在租户；上下文无租户（平台级/旧 token）时归默认租户
	tenantID := tenant.FromContext(ctx)
	if tenantID == 0 {
		tenantID = tenant.DefaultTenantID
	}

	key := &domain.APIKey{
		TenantID:  tenantID,
		Name:      input.Name,
		KeyPrefix: plaintext[:min(8+len(apiKeyPrefix), len(plaintext))],
		KeyHash:   domain.HashKey(plaintext),
		Scopes:    string(scopesJSON),
		Enabled:   true,
		CreatedBy: input.CreatedBy,
	}
	if input.ExpiresIn > 0 {
		key.ExpiresAt = time.Now().Add(time.Duration(input.ExpiresIn) * 24 * time.Hour)
	}

	if err := s.repo.Create(ctx, key); err != nil {
		return "", nil, err
	}
	return plaintext, key, nil
}

// GetKey 获取 API Key 详情（跨租户不可见）
func (s *AdminAPIKeyService) GetKey(ctx context.Context, id uint64) (*domain.APIKey, error) {
	key, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if !tenantVisible(key.TenantID, tenant.FromContext(ctx)) {
		return nil, apperrors.New(apperrors.CodeNotFound, "api key not found")
	}
	return key, nil
}

// RevokeKey 撤销 API Key（跨租户不可见）
func (s *AdminAPIKeyService) RevokeKey(ctx context.Context, id uint64) error {
	key, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return err
	}
	if !tenantVisible(key.TenantID, tenant.FromContext(ctx)) {
		return apperrors.New(apperrors.CodeNotFound, "api key not found")
	}
	return s.repo.Delete(ctx, id)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
