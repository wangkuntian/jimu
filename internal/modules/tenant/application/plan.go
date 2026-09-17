package application

import (
	"context"
	"strings"

	"jimu/internal/modules/tenant/domain"
	"jimu/internal/shared/errors"
	"jimu/internal/shared/pagination"
)

// CreatePlanRequest 创建套餐请求
type CreatePlanRequest struct {
	Code       string `json:"code" binding:"required,min=1,max=32"`
	Name       string `json:"name" binding:"required,min=1,max=64"`
	MaxUsers   int    `json:"max_users" binding:"omitempty,min=0"`    // 0=不限
	MaxRoles   int    `json:"max_roles" binding:"omitempty,min=0"`    // 0=不限
	MaxAPIKeys int    `json:"max_api_keys" binding:"omitempty,min=0"` // 0=不限
}

// UpdatePlanRequest 更新套餐请求（编码不可修改，字段为空表示不修改）
type UpdatePlanRequest struct {
	Name       *string `json:"name" binding:"omitempty,min=1,max=64"`
	MaxUsers   *int    `json:"max_users" binding:"omitempty,min=0"`
	MaxRoles   *int    `json:"max_roles" binding:"omitempty,min=0"`
	MaxAPIKeys *int    `json:"max_api_keys" binding:"omitempty,min=0"`
}

// AssignPlanRequest 为租户分配套餐请求（plan_id=0 表示取消套餐）
type AssignPlanRequest struct {
	PlanID uint64 `json:"plan_id"`
}

// PlanResponse 套餐信息
type PlanResponse struct {
	ID         uint64 `json:"id"`
	Code       string `json:"code"`
	Name       string `json:"name"`
	MaxUsers   int    `json:"max_users"`
	MaxRoles   int    `json:"max_roles"`
	MaxAPIKeys int    `json:"max_api_keys"`
}

// UsageResponse 租户用量：套餐 + 各资源当前用量与上限
type UsageResponse struct {
	TenantID uint64               `json:"tenant_id"`
	Plan     *PlanResponse        `json:"plan"` // null = 未分配套餐（不限）
	Users    domain.ResourceUsage `json:"users"`
	Roles    domain.ResourceUsage `json:"roles"`
	APIKeys  domain.ResourceUsage `json:"api_keys"`
}

func ToPlanResponse(p domain.Plan) PlanResponse {
	return PlanResponse{
		ID:         p.ID,
		Code:       p.Code,
		Name:       p.Name,
		MaxUsers:   p.MaxUsers,
		MaxRoles:   p.MaxRoles,
		MaxAPIKeys: p.MaxAPIKeys,
	}
}

// PlanService 套餐管理（平台级：套餐定义与租户绑定）
type PlanService struct {
	plans domain.PlanRepository
	quota domain.QuotaRepository
}

func NewPlanService(plans domain.PlanRepository, quota domain.QuotaRepository) *PlanService {
	return &PlanService{plans: plans, quota: quota}
}

// Create 创建套餐。编码全局唯一且不可修改，统一转小写存储。
func (s *PlanService) Create(ctx context.Context, req CreatePlanRequest) (*PlanResponse, error) {
	code := strings.ToLower(strings.TrimSpace(req.Code))
	existing, err := s.plans.FindByCode(ctx, code)
	if err == nil && existing != nil {
		return nil, errors.New(errors.CodeTenantExists, "plan code already exists")
	}
	if err != nil && !isNotFound(err) {
		return nil, errors.Wrap(errors.CodeInternalError, "failed to find plan by code", err)
	}

	plan := &domain.Plan{
		Code:       code,
		Name:       req.Name,
		MaxUsers:   req.MaxUsers,
		MaxRoles:   req.MaxRoles,
		MaxAPIKeys: req.MaxAPIKeys,
	}
	if err := s.plans.Create(ctx, plan); err != nil {
		return nil, errors.Wrap(errors.CodeInternalError, "failed to create plan", err)
	}
	resp := ToPlanResponse(*plan)
	return &resp, nil
}

// List 分页列出套餐。
func (s *PlanService) List(ctx context.Context, p pagination.Pagination) ([]PlanResponse, int64, error) {
	plans, total, err := s.plans.List(ctx, p.GetOffset(), p.GetLimit())
	if err != nil {
		return nil, 0, errors.Wrap(errors.CodeInternalError, "failed to list plans", err)
	}
	out := make([]PlanResponse, 0, len(plans))
	for _, plan := range plans {
		out = append(out, ToPlanResponse(plan))
	}
	return out, total, nil
}

// Update 更新套餐上限；对已使用该套餐的租户立即生效。
func (s *PlanService) Update(ctx context.Context, id uint64, req UpdatePlanRequest) error {
	plan, err := s.plans.FindByID(ctx, id)
	if err != nil {
		if isNotFound(err) {
			return errors.Wrap(errors.CodeTenantNotFound, "plan not found", err)
		}
		return errors.Wrap(errors.CodeInternalError, "failed to get plan", err)
	}
	if req.Name != nil {
		plan.Name = *req.Name
	}
	if req.MaxUsers != nil {
		plan.MaxUsers = *req.MaxUsers
	}
	if req.MaxRoles != nil {
		plan.MaxRoles = *req.MaxRoles
	}
	if req.MaxAPIKeys != nil {
		plan.MaxAPIKeys = *req.MaxAPIKeys
	}
	if err := s.plans.Update(ctx, plan); err != nil {
		return errors.Wrap(errors.CodeInternalError, "failed to update plan", err)
	}
	return nil
}

// Delete 删除套餐；仍被租户使用时拒绝，避免租户静默失去配额约束。
func (s *PlanService) Delete(ctx context.Context, id uint64) error {
	if _, err := s.plans.FindByID(ctx, id); err != nil {
		if isNotFound(err) {
			return errors.Wrap(errors.CodeTenantNotFound, "plan not found", err)
		}
		return errors.Wrap(errors.CodeInternalError, "failed to get plan", err)
	}
	used, err := s.plans.CountTenants(ctx, id)
	if err != nil {
		return errors.Wrap(errors.CodeInternalError, "failed to count tenants of plan", err)
	}
	if used > 0 {
		return errors.New(errors.CodeConflict, "plan is still assigned to tenants")
	}
	if err := s.plans.Delete(ctx, id); err != nil {
		return errors.Wrap(errors.CodeInternalError, "failed to delete plan", err)
	}
	return nil
}

// Assign 为租户分配套餐（planID=0 取消套餐，取消后不再受配额限制）。
func (s *PlanService) Assign(ctx context.Context, tenantID, planID uint64) error {
	if planID != 0 {
		if _, err := s.plans.FindByID(ctx, planID); err != nil {
			if isNotFound(err) {
				return errors.Wrap(errors.CodeTenantNotFound, "plan not found", err)
			}
			return errors.Wrap(errors.CodeInternalError, "failed to get plan", err)
		}
	}
	if err := s.quota.AssignPlan(ctx, tenantID, planID); err != nil {
		return errors.Wrap(errors.CodeInternalError, "failed to assign plan", err)
	}
	return nil
}

// Usage 返回租户当前用量与上限。
func (s *PlanService) Usage(ctx context.Context, tenantID uint64) (*UsageResponse, error) {
	plan, err := s.quota.FindPlanByTenant(ctx, tenantID)
	if err != nil {
		return nil, errors.Wrap(errors.CodeInternalError, "failed to load tenant plan", err)
	}
	resp := &UsageResponse{TenantID: tenantID}
	if plan != nil {
		planResp := ToPlanResponse(*plan)
		resp.Plan = &planResp
	}
	resources := []struct {
		resource domain.QuotaResource
		target   *domain.ResourceUsage
		limit    int
	}{
		{domain.QuotaResourceUsers, &resp.Users, limitOf(plan, func(p *domain.Plan) int { return p.MaxUsers })},
		{domain.QuotaResourceRoles, &resp.Roles, limitOf(plan, func(p *domain.Plan) int { return p.MaxRoles })},
		{domain.QuotaResourceAPIKeys, &resp.APIKeys, limitOf(plan, func(p *domain.Plan) int { return p.MaxAPIKeys })},
	}
	for _, item := range resources {
		used, err := s.quota.Count(ctx, tenantID, item.resource)
		if err != nil {
			return nil, errors.Wrap(errors.CodeInternalError, "failed to count tenant usage", err)
		}
		item.target.Used = used
		item.target.Limit = item.limit
	}
	return resp, nil
}

// limitOf 取套餐中某项上限；未分配套餐时为 0（不限）
func limitOf(plan *domain.Plan, pick func(*domain.Plan) int) int {
	if plan == nil {
		return 0
	}
	return pick(plan)
}

// QuotaService 配额校验：创建资源前检查套餐上限，未分配套餐或上限为 0 时放行。
type QuotaService struct {
	quota domain.QuotaRepository
}

func NewQuotaService(quota domain.QuotaRepository) *QuotaService {
	return &QuotaService{quota: quota}
}

// CheckUserQuota 校验用户数配额（管理端创建用户、开通式注册等调用）
func (s *QuotaService) CheckUserQuota(ctx context.Context, tenantID uint64) error {
	return s.check(ctx, tenantID, domain.QuotaResourceUsers, func(p *domain.Plan) int { return p.MaxUsers })
}

// CheckRoleQuota 校验角色数配额
func (s *QuotaService) CheckRoleQuota(ctx context.Context, tenantID uint64) error {
	return s.check(ctx, tenantID, domain.QuotaResourceRoles, func(p *domain.Plan) int { return p.MaxRoles })
}

// CheckAPIKeyQuota 校验 API Key 数量配额
func (s *QuotaService) CheckAPIKeyQuota(ctx context.Context, tenantID uint64) error {
	return s.check(ctx, tenantID, domain.QuotaResourceAPIKeys, func(p *domain.Plan) int { return p.MaxAPIKeys })
}

func (s *QuotaService) check(ctx context.Context, tenantID uint64, resource domain.QuotaResource, pick func(*domain.Plan) int) error {
	if tenantID == 0 {
		return nil // 平台级视角：无租户归属，不受配额约束
	}
	plan, err := s.quota.FindPlanByTenant(ctx, tenantID)
	if err != nil {
		return errors.Wrap(errors.CodeInternalError, "failed to load tenant plan", err)
	}
	if plan == nil {
		return nil
	}
	limit := pick(plan)
	if limit <= 0 {
		return nil
	}
	used, err := s.quota.Count(ctx, tenantID, resource)
	if err != nil {
		return errors.Wrap(errors.CodeInternalError, "failed to count tenant usage", err)
	}
	if used >= int64(limit) {
		return errors.New(errors.CodeQuotaExceeded, string(resource)+" quota exceeded")
	}
	return nil
}
