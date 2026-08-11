// internal/modules/oauth/domain/repository.go
package domain

import "context"

// BindingRepository 绑定记录仓储接口
type BindingRepository interface {
	// FindByProviderSubject 按 provider+subject 查绑定
	FindByProviderSubject(ctx context.Context, provider, subject string) (*OAuthBinding, error)
	// Create 创建绑定
	Create(ctx context.Context, binding *OAuthBinding) error
}
