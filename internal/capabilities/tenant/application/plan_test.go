package application

import (
	"context"
	"testing"

	"jimu/internal/capabilities/tenant/domain"
	"jimu/internal/shared/errors"
	"jimu/internal/shared/pagination"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// fakePlanRepo 内存套餐仓储
type fakePlanRepo struct {
	plans    []*domain.Plan
	nextID   uint64
	tenants  map[uint64]uint64 // planID -> 使用该套餐的租户数
	createFn func(*domain.Plan) error
}

func newFakePlanRepo(plans ...*domain.Plan) *fakePlanRepo {
	return &fakePlanRepo{plans: plans, nextID: uint64(len(plans) + 1), tenants: map[uint64]uint64{}}
}

func (r *fakePlanRepo) FindByID(_ context.Context, id uint64) (*domain.Plan, error) {
	for _, plan := range r.plans {
		if plan.ID == id {
			return plan, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

func (r *fakePlanRepo) FindByCode(_ context.Context, code string) (*domain.Plan, error) {
	for _, plan := range r.plans {
		if plan.Code == code {
			return plan, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

func (r *fakePlanRepo) List(_ context.Context, _, _ int) ([]domain.Plan, int64, error) {
	out := make([]domain.Plan, 0, len(r.plans))
	for _, plan := range r.plans {
		out = append(out, *plan)
	}
	return out, int64(len(out)), nil
}

func (r *fakePlanRepo) Create(_ context.Context, plan *domain.Plan) error {
	if r.createFn != nil {
		return r.createFn(plan)
	}
	plan.ID = r.nextID
	r.nextID++
	r.plans = append(r.plans, plan)
	return nil
}

func (r *fakePlanRepo) Update(_ context.Context, plan *domain.Plan) error {
	for i, existing := range r.plans {
		if existing.ID == plan.ID {
			r.plans[i] = plan
			return nil
		}
	}
	return gorm.ErrRecordNotFound
}

func (r *fakePlanRepo) Delete(_ context.Context, id uint64) error {
	kept := r.plans[:0]
	for _, plan := range r.plans {
		if plan.ID != id {
			kept = append(kept, plan)
		}
	}
	r.plans = kept
	return nil
}

func (r *fakePlanRepo) CountTenants(_ context.Context, planID uint64) (int64, error) {
	return int64(r.tenants[planID]), nil
}

// fakeQuotaRepo 内存配额仓储
type fakeQuotaRepo struct {
	planByTenant map[uint64]*domain.Plan
	counts       map[uint64]map[domain.QuotaResource]int64
	assigned     map[uint64]uint64
}

func newFakeQuotaRepo() *fakeQuotaRepo {
	return &fakeQuotaRepo{
		planByTenant: map[uint64]*domain.Plan{},
		counts:       map[uint64]map[domain.QuotaResource]int64{},
		assigned:     map[uint64]uint64{},
	}
}

func (r *fakeQuotaRepo) FindPlanByTenant(_ context.Context, tenantID uint64) (*domain.Plan, error) {
	return r.planByTenant[tenantID], nil
}

func (r *fakeQuotaRepo) Count(_ context.Context, tenantID uint64, resource domain.QuotaResource) (int64, error) {
	return r.counts[tenantID][resource], nil
}

func (r *fakeQuotaRepo) AssignPlan(_ context.Context, tenantID, planID uint64) error {
	r.assigned[tenantID] = planID
	return nil
}

func freePlan() *domain.Plan {
	return &domain.Plan{ID: 1, Code: "free", Name: "免费版", MaxUsers: 2, MaxRoles: 1, MaxAPIKeys: 1}
}

func TestQuotaServiceEnforcesLimits(t *testing.T) {
	ctx := context.Background()
	quotaRepo := newFakeQuotaRepo()
	quotaRepo.planByTenant[7] = freePlan()
	quotaRepo.counts[7] = map[domain.QuotaResource]int64{
		domain.QuotaResourceUsers:   2, // 已达上限
		domain.QuotaResourceRoles:   0,
		domain.QuotaResourceAPIKeys: 1, // 已达上限
	}
	svc := NewQuotaService(quotaRepo)

	assert.Equal(t, errors.CodeQuotaExceeded, tenantAppCode(svc.CheckUserQuota(ctx, 7)))
	assert.Equal(t, errors.CodeQuotaExceeded, tenantAppCode(svc.CheckAPIKeyQuota(ctx, 7)))
	assert.NoError(t, svc.CheckRoleQuota(ctx, 7), "未达上限应放行")

	// 未分配套餐的租户不受约束
	quotaRepo.counts[8] = map[domain.QuotaResource]int64{domain.QuotaResourceUsers: 999}
	assert.NoError(t, svc.CheckUserQuota(ctx, 8))

	// 平台级视角（tenantID=0）不校验
	assert.NoError(t, svc.CheckUserQuota(ctx, 0))
}

func TestQuotaServiceTreatsZeroLimitAsUnlimited(t *testing.T) {
	ctx := context.Background()
	quotaRepo := newFakeQuotaRepo()
	quotaRepo.planByTenant[7] = &domain.Plan{ID: 2, Code: "pro", MaxUsers: 0, MaxRoles: 0, MaxAPIKeys: 0}
	quotaRepo.counts[7] = map[domain.QuotaResource]int64{domain.QuotaResourceUsers: 1000}
	svc := NewQuotaService(quotaRepo)

	assert.NoError(t, svc.CheckUserQuota(ctx, 7))
	assert.NoError(t, svc.CheckRoleQuota(ctx, 7))
	assert.NoError(t, svc.CheckAPIKeyQuota(ctx, 7))
}

func TestPlanServiceCreateNormalizesCodeAndRejectsDuplicate(t *testing.T) {
	ctx := context.Background()
	repo := newFakePlanRepo(freePlan())
	svc := NewPlanService(repo, newFakeQuotaRepo())

	plan, err := svc.Create(ctx, CreatePlanRequest{Code: "  PRO  ", Name: "专业版", MaxUsers: 100})
	require.NoError(t, err)
	assert.Equal(t, "pro", plan.Code, "编码应统一小写并去空格")

	_, err = svc.Create(ctx, CreatePlanRequest{Code: "pro", Name: "重复"})
	assert.Equal(t, errors.CodeTenantExists, tenantAppCode(err))
}

func TestPlanServiceDeleteRejectsAssignedPlan(t *testing.T) {
	ctx := context.Background()
	repo := newFakePlanRepo(freePlan())
	repo.tenants[1] = 3
	svc := NewPlanService(repo, newFakeQuotaRepo())

	err := svc.Delete(ctx, 1)
	assert.Equal(t, errors.CodeConflict, tenantAppCode(err), "仍被租户使用的套餐不可删除")

	delete(repo.tenants, 1)
	assert.NoError(t, svc.Delete(ctx, 1))

	err = svc.Delete(ctx, 999)
	assert.Equal(t, errors.CodeTenantNotFound, tenantAppCode(err))
}

func TestPlanServiceAssignValidatesPlan(t *testing.T) {
	ctx := context.Background()
	quotaRepo := newFakeQuotaRepo()
	svc := NewPlanService(newFakePlanRepo(freePlan()), quotaRepo)

	assert.NoError(t, svc.Assign(ctx, 7, 1))
	assert.Equal(t, uint64(1), quotaRepo.assigned[7])

	assert.NoError(t, svc.Assign(ctx, 7, 0), "plan_id=0 表示取消套餐")
	assert.Equal(t, uint64(0), quotaRepo.assigned[7])

	err := svc.Assign(ctx, 7, 999)
	assert.Equal(t, errors.CodeTenantNotFound, tenantAppCode(err))
}

func TestPlanServiceUpdateAndUsage(t *testing.T) {
	ctx := context.Background()
	repo := newFakePlanRepo(freePlan())
	quotaRepo := newFakeQuotaRepo()
	quotaRepo.planByTenant[7] = freePlan()
	quotaRepo.counts[7] = map[domain.QuotaResource]int64{
		domain.QuotaResourceUsers:   1,
		domain.QuotaResourceRoles:   1,
		domain.QuotaResourceAPIKeys: 0,
	}
	svc := NewPlanService(repo, quotaRepo)

	name, maxUsers := "升级版", 50
	require.NoError(t, svc.Update(ctx, 1, UpdatePlanRequest{Name: &name, MaxUsers: &maxUsers}))
	assert.Equal(t, "升级版", repo.plans[0].Name)
	assert.Equal(t, 50, repo.plans[0].MaxUsers)
	assert.Equal(t, 1, repo.plans[0].MaxRoles, "未传字段保持不变")

	usage, err := svc.Usage(ctx, 7)
	require.NoError(t, err)
	require.NotNil(t, usage.Plan)
	assert.Equal(t, "free", usage.Plan.Code)
	assert.Equal(t, domain.ResourceUsage{Used: 1, Limit: 2}, usage.Users)
	assert.Equal(t, domain.ResourceUsage{Used: 1, Limit: 1}, usage.Roles)
	assert.Equal(t, domain.ResourceUsage{Used: 0, Limit: 1}, usage.APIKeys)

	// 未分配套餐：plan 为 nil，上限全为 0（不限）
	usage, err = svc.Usage(ctx, 99)
	require.NoError(t, err)
	assert.Nil(t, usage.Plan)
	assert.Equal(t, 0, usage.Users.Limit)
}

func TestPlanServiceList(t *testing.T) {
	svc := NewPlanService(newFakePlanRepo(freePlan()), newFakeQuotaRepo())
	plans, total, err := svc.List(context.Background(), pagination.Pagination{Page: 1, PageSize: 20})
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	require.Len(t, plans, 1)
	assert.Equal(t, "free", plans[0].Code)
}
