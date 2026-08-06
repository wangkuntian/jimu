package queue

import (
	"context"

	"jimu/internal/modules/admin/domain"
)

// DeadLetterHandler 死信队列处理器
type DeadLetterHandler struct {
	repo domain.DeadLetterRepository
}

// NewDeadLetterHandler 创建死信处理器
func NewDeadLetterHandler(repo domain.DeadLetterRepository) *DeadLetterHandler {
	return &DeadLetterHandler{repo: repo}
}

// List 获取死信列表
func (h *DeadLetterHandler) List(ctx context.Context, offset, limit int) ([]domain.DeadLetter, int64, error) {
	return h.repo.List(ctx, offset, limit, false)
}

// Resolve 处理死信
func (h *DeadLetterHandler) Resolve(ctx context.Context, id uint64) error {
	return h.repo.MarkResolved(ctx, id)
}
