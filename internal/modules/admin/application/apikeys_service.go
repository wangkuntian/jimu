package application

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"time"

	"jimu/internal/modules/admin/domain"
	apperrors "jimu/internal/shared/errors"
)

const apiKeyPrefix = "jimu_"

// AdminAPIKeyService API Key 管理服务
type AdminAPIKeyService struct {
	repo domain.APIKeyRepository
}

// NewAdminAPIKeyService 创建 API Key 管理服务
func NewAdminAPIKeyService(repo domain.APIKeyRepository) *AdminAPIKeyService {
	return &AdminAPIKeyService{repo: repo}
}

// ListKeys 获取 API Key 列表
func (s *AdminAPIKeyService) ListKeys(ctx context.Context, offset, limit int) ([]domain.APIKey, int64, error) {
	return s.repo.List(ctx, offset, limit)
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

	key := &domain.APIKey{
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

// GetKey 获取 API Key 详情
func (s *AdminAPIKeyService) GetKey(ctx context.Context, id uint64) (*domain.APIKey, error) {
	return s.repo.FindByID(ctx, id)
}

// RevokeKey 撤销 API Key
func (s *AdminAPIKeyService) RevokeKey(ctx context.Context, id uint64) error {
	return s.repo.Delete(ctx, id)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
