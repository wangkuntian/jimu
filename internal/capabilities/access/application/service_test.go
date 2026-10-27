package application

import (
	"context"
	stderrors "errors"
	"testing"

	"jimu/internal/capabilities/access/domain"
	"jimu/internal/contract"
	dbutil "jimu/internal/kernel/db"
	"jimu/internal/kernel/tenant"
	apperrors "jimu/internal/shared/errors"
	"jimu/internal/shared/pagination"

	mysqlerr "github.com/go-sql-driver/mysql"
	"gorm.io/gorm"
)

func TestRoleServiceCreateMapsDuplicateNameToConflict(t *testing.T) {
	service := NewRoleService(&fakeRoleRepository{createErr: &mysqlerr.MySQLError{Number: 1062, Message: "Duplicate entry"}})

	_, err := service.Create(context.Background(), CreateRoleRequest{Name: "admin", Description: "ops"})
	if roleAppCode(err) != apperrors.CodeConflict {
		t.Fatalf("code = %d, want %d", roleAppCode(err), apperrors.CodeConflict)
	}
}

func TestRoleServiceListPassesPagination(t *testing.T) {
	repo := &fakeRoleRepository{roles: []domain.Role{{ID: 1, Name: "admin"}}, total: 12}
	service := NewRoleService(repo)

	roles, total, err := service.List(context.Background(), pagination.Pagination{Page: 2, PageSize: 5, Sort: "created_at", Order: "asc"}, domain.RoleListFilter{})
	if err != nil {
		t.Fatal(err)
	}
	if repo.offset != 5 || repo.limit != 5 || repo.sort != "created_at" || repo.order != "asc" {
		t.Fatalf("pagination = offset:%d limit:%d sort:%q order:%q", repo.offset, repo.limit, repo.sort, repo.order)
	}
	if total != 12 || len(roles) != 1 || roles[0].Name != "admin" {
		t.Fatalf("roles = %#v total = %d", roles, total)
	}
}

func TestRoleServiceListPassesTenantFilter(t *testing.T) {
	repo := &fakeRoleRepository{}
	_, _, err := NewRoleService(repo).List(context.Background(), pagination.Pagination{Page: 1, PageSize: 20}, domain.RoleListFilter{TenantID: 9})
	if err != nil {
		t.Fatal(err)
	}
	if repo.filter.TenantID != 9 {
		t.Fatalf("tenantID = %d, want 9", repo.filter.TenantID)
	}
}

func TestRoleServiceListPassesKeywordAndStatus(t *testing.T) {
	status := int8(0)
	repo := &fakeRoleRepository{}
	_, _, err := NewRoleService(repo).List(context.Background(), pagination.Pagination{Page: 1, PageSize: 20}, domain.RoleListFilter{
		Keyword: "ops",
		Status:  &status,
	})
	if err != nil {
		t.Fatal(err)
	}
	if repo.filter.Keyword != "ops" || repo.filter.Status == nil || *repo.filter.Status != 0 {
		t.Fatalf("filter = %#v", repo.filter)
	}
}

func TestRoleServiceUpdateMapsNotFound(t *testing.T) {
	service := NewRoleService(&fakeRoleRepository{findErr: gorm.ErrRecordNotFound})

	err := service.Update(context.Background(), 8, UpdateRoleRequest{Name: "admin"})
	if roleAppCode(err) != apperrors.CodeNotFound {
		t.Fatalf("code = %d, want %d", roleAppCode(err), apperrors.CodeNotFound)
	}
}

func TestRoleServiceUpdateMapsConcurrentUpdateToConflict(t *testing.T) {
	service := NewRoleService(&fakeRoleRepository{updateErr: dbutil.ErrConcurrentUpdate})

	err := service.Update(context.Background(), 8, UpdateRoleRequest{Name: "admin"})
	if roleAppCode(err) != apperrors.CodeConflict {
		t.Fatalf("code = %d, want %d", roleAppCode(err), apperrors.CodeConflict)
	}
}

func TestRoleServiceDeleteWrapsRepositoryError(t *testing.T) {
	cause := stderrors.New("sql: connection refused")
	service := NewRoleService(&fakeRoleRepository{deleteErr: cause})

	err := service.Delete(context.Background(), 8)
	if roleAppCode(err) != apperrors.CodeInternalError || !stderrors.Is(err, cause) {
		t.Fatalf("err = %v", err)
	}
}

func TestRoleServiceCreatePlatformUsesRequestedTenant(t *testing.T) {
	repo := &fakeRoleRepository{}
	quota := &fakeTenantQuota{exists: true}
	service := NewRoleService(repo).WithQuota(quota)

	resp, err := service.Create(tenant.WithPlatformScope(context.Background()), CreateRoleRequest{
		TenantID: "9",
		Name:     "operator",
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.TenantID != "9" || len(repo.created) != 1 || len(quota.existsChecked) != 1 || quota.existsChecked[0] != 9 {
		t.Fatalf("resp = %#v created = %v existsChecked = %v", resp, repo.created, quota.existsChecked)
	}
}

func TestRoleServiceCreatePlatformRequiresExistingTenant(t *testing.T) {
	service := NewRoleService(&fakeRoleRepository{}).WithQuota(&fakeTenantQuota{exists: true})
	_, err := service.Create(tenant.WithPlatformScope(context.Background()), CreateRoleRequest{Name: "operator"})
	if roleAppCode(err) != apperrors.CodeInvalidParam {
		t.Fatalf("missing tenant_id code = %d", roleAppCode(err))
	}

	service = NewRoleService(&fakeRoleRepository{}).WithQuota(&fakeTenantQuota{exists: false})
	_, err = service.Create(tenant.WithPlatformScope(context.Background()), CreateRoleRequest{TenantID: "9", Name: "operator"})
	if roleAppCode(err) != apperrors.CodeNotFound {
		t.Fatalf("missing tenant code = %d", roleAppCode(err))
	}
}

func TestRoleServiceCreateRejectsCrossTenant(t *testing.T) {
	service := NewRoleService(&fakeRoleRepository{})
	_, err := service.Create(tenant.WithTenant(context.Background(), 7), CreateRoleRequest{TenantID: "9", Name: "operator"})
	if roleAppCode(err) != apperrors.CodeForbidden {
		t.Fatalf("code = %d, want %d", roleAppCode(err), apperrors.CodeForbidden)
	}
}

func TestRoleServiceUpdateStatus(t *testing.T) {
	enabled := int8(1)
	repo := &fakeRoleRepository{role: &domain.Role{ID: 3, Name: "ops", Status: 0}}
	service := NewRoleService(repo)
	if err := service.Update(context.Background(), 3, UpdateRoleRequest{Name: "ops", Status: &enabled}); err != nil {
		t.Fatal(err)
	}
	if repo.updatedRole == nil || repo.updatedRole.Status != 1 {
		t.Fatalf("updated role = %#v", repo.updatedRole)
	}

	disabled := int8(0)
	if err := service.Update(context.Background(), 3, UpdateRoleRequest{Name: "ops", Status: &disabled}); err != nil {
		t.Fatal(err)
	}
	if repo.updatedRole == nil || repo.updatedRole.Status != 0 {
		t.Fatalf("disabled role = %#v", repo.updatedRole)
	}
}

func TestRoleServiceAssignPermissionsDeduplicatesAndMapsMissing(t *testing.T) {
	repo := &fakeRoleRepository{role: &domain.Role{ID: 3}}
	service := NewRoleService(repo)
	if err := service.AssignPermissions(context.Background(), 3, []uint64{2, 1, 2}); err != nil {
		t.Fatal(err)
	}
	if len(repo.assigned) != 2 || repo.assigned[0] != 2 || repo.assigned[1] != 1 {
		t.Fatalf("assigned = %v", repo.assigned)
	}

	repo = &fakeRoleRepository{role: &domain.Role{ID: 3}, assignErr: domain.ErrPermissionNotFound}
	err := NewRoleService(repo).AssignPermissions(context.Background(), 3, []uint64{99})
	if roleAppCode(err) != apperrors.CodeInvalidParam {
		t.Fatalf("code = %d, want %d", roleAppCode(err), apperrors.CodeInvalidParam)
	}
}

func TestRoleServicePlatformScopeManagesOtherTenant(t *testing.T) {
	repo := &fakeRoleRepository{role: &domain.Role{ID: 3, TenantID: 2, Name: "ops", Status: 1}}
	service := NewRoleService(repo)
	ctx := tenant.WithPlatformScope(tenant.WithTenant(context.Background(), 1))

	resp, err := service.Get(ctx, 3)
	if err != nil || resp.TenantID != "2" {
		t.Fatalf("platform get resp = %#v err = %v", resp, err)
	}
	if err := service.Update(ctx, 3, UpdateRoleRequest{Name: "ops2"}); err != nil {
		t.Fatal(err)
	}
	if err := service.AssignPermissions(ctx, 3, []uint64{1}); err != nil {
		t.Fatal(err)
	}
	if _, err := service.GetPermissions(ctx, 3); err != nil {
		t.Fatal(err)
	}
	if err := service.Delete(ctx, 3); err != nil {
		t.Fatal(err)
	}

	_, err = service.Get(tenant.WithTenant(context.Background(), 1), 3)
	if roleAppCode(err) != apperrors.CodeNotFound {
		t.Fatalf("cross-tenant get code = %d, want %d", roleAppCode(err), apperrors.CodeNotFound)
	}
}

func TestRoleServiceOptions(t *testing.T) {
	repo := &fakeRoleRepository{options: []domain.Role{
		{
			ID:       2,
			TenantID: 9,
			Name:     "租户管理员",
			Code:     contract.RoleCodeTenantAdmin,
			Scope:    contract.RoleScopeTenant,
			IsSystem: true,
		},
	}}
	service := NewRoleService(repo)

	got, err := service.Options(tenant.WithTenant(context.Background(), 9), 0)
	if err != nil {
		t.Fatal(err)
	}
	if repo.optionsTenantID != 9 || len(got) != 1 || got[0].Code != contract.RoleCodeTenantAdmin || !got[0].IsSystem {
		t.Fatalf("options tenant = %d got = %#v", repo.optionsTenantID, got)
	}
	if repo.optionsIncludePlatform {
		t.Fatal("tenant-scoped options must not include platform roles")
	}

	_, err = service.Options(tenant.WithTenant(context.Background(), 9), 10)
	if roleAppCode(err) != apperrors.CodeForbidden {
		t.Fatalf("cross tenant code = %d, want %d", roleAppCode(err), apperrors.CodeForbidden)
	}

	_, err = service.Options(tenant.WithPlatformScope(tenant.WithTenant(context.Background(), 1)), 10)
	if err != nil {
		t.Fatal(err)
	}
	if repo.optionsTenantID != 10 {
		t.Fatalf("platform options tenant = %d, want 10", repo.optionsTenantID)
	}
	if repo.optionsIncludePlatform {
		t.Fatal("non-default tenant options must not include platform roles")
	}

	_, err = service.Options(tenant.WithPlatformScope(tenant.WithTenant(context.Background(), tenant.DefaultTenantID)), 0)
	if err != nil {
		t.Fatal(err)
	}
	if repo.optionsTenantID != tenant.DefaultTenantID || !repo.optionsIncludePlatform {
		t.Fatalf("default tenant options = tenant:%d includePlatform:%v", repo.optionsTenantID, repo.optionsIncludePlatform)
	}

	repo.optionsIncludePlatform = false
	_, err = service.Options(tenant.WithTenant(context.Background(), tenant.DefaultTenantID), 0)
	if err != nil {
		t.Fatal(err)
	}
	if repo.optionsIncludePlatform {
		t.Fatal("non-platform identities must not receive platform role options")
	}

	missingService := NewRoleService(repo).WithQuota(&fakeTenantQuota{exists: false})
	_, err = missingService.Options(tenant.WithPlatformScope(tenant.WithTenant(context.Background(), 1)), 12)
	if roleAppCode(err) != apperrors.CodeNotFound {
		t.Fatalf("missing tenant code = %d, want %d", roleAppCode(err), apperrors.CodeNotFound)
	}
}

func TestSystemRoleIsImmutable(t *testing.T) {
	repo := &fakeRoleRepository{role: &domain.Role{
		ID:       3,
		TenantID: 1,
		Name:     "租户管理员",
		Code:     contract.RoleCodeTenantAdmin,
		Scope:    contract.RoleScopeTenant,
		IsSystem: true,
		Status:   1,
	}}
	service := NewRoleService(repo)

	if err := service.Update(context.Background(), 3, UpdateRoleRequest{Name: "changed"}); roleAppCode(err) != apperrors.CodeForbidden {
		t.Fatalf("update code = %d, want %d", roleAppCode(err), apperrors.CodeForbidden)
	}
	if err := service.Delete(context.Background(), 3); roleAppCode(err) != apperrors.CodeForbidden {
		t.Fatalf("delete code = %d, want %d", roleAppCode(err), apperrors.CodeForbidden)
	}
	if err := service.AssignPermissions(context.Background(), 3, []uint64{1}); roleAppCode(err) != apperrors.CodeForbidden {
		t.Fatalf("assign code = %d, want %d", roleAppCode(err), apperrors.CodeForbidden)
	}
	if repo.updatedRole != nil || repo.assigned != nil {
		t.Fatalf("system role was mutated: updated=%#v assigned=%v", repo.updatedRole, repo.assigned)
	}
	if repo.deleted {
		t.Fatal("system role was deleted")
	}
}

func TestTenantRoleCannotReceivePlatformOnlyPermission(t *testing.T) {
	repo := &fakeRoleRepository{
		role: &domain.Role{ID: 3, TenantID: 9, Scope: contract.RoleScopeTenant},
		permissionsByIDs: map[uint64]domain.Permission{
			1: {ID: 1, Resource: "/api/v1/tenants", Action: "GET"},
		},
	}
	err := NewRoleService(repo).AssignPermissions(tenant.WithTenant(context.Background(), 9), 3, []uint64{1})
	if roleAppCode(err) != apperrors.CodeForbidden {
		t.Fatalf("code = %d, want %d", roleAppCode(err), apperrors.CodeForbidden)
	}
	if repo.assigned != nil {
		t.Fatalf("invalid permission was assigned: %v", repo.assigned)
	}

	repo = &fakeRoleRepository{
		role: &domain.Role{ID: 4, TenantID: 1, Scope: contract.RoleScopePlatform},
		permissionsByIDs: map[uint64]domain.Permission{
			1: {ID: 1, Resource: "/api/v1/tenants", Action: "GET"},
		},
	}
	ctx := tenant.WithPlatformScope(tenant.WithTenant(context.Background(), 1))
	if err := NewRoleService(repo).AssignPermissions(ctx, 4, []uint64{1}); err != nil {
		t.Fatal(err)
	}
	if len(repo.assigned) != 1 || repo.assigned[0] != 1 {
		t.Fatalf("assigned = %v", repo.assigned)
	}
}

type fakeRoleRepository struct {
	role                   *domain.Role
	roles                  []domain.Role
	total                  int64
	findErr                error
	createErr              error
	updateErr              error
	deleteErr              error
	assignErr              error
	offset                 int
	limit                  int
	sort                   string
	order                  string
	filter                 domain.RoleListFilter
	created                []string
	updatedRole            *domain.Role
	deleted                bool
	assigned               []uint64
	permissions            []domain.Permission
	getPermissionsErr      error
	options                []domain.Role
	optionsTenantID        uint64
	optionsIncludePlatform bool
	permissionsByIDs       map[uint64]domain.Permission
}

func (r *fakeRoleRepository) FindByID(context.Context, uint64) (*domain.Role, error) {
	if r.role != nil {
		return r.role, r.findErr
	}
	return &domain.Role{}, r.findErr
}

func (r *fakeRoleRepository) List(_ context.Context, filter domain.RoleListFilter, offset, limit int, sort, order string) ([]domain.Role, int64, error) {
	r.filter = filter
	r.offset = offset
	r.limit = limit
	r.sort = sort
	r.order = order
	return r.roles, r.total, nil
}

func (r *fakeRoleRepository) Create(_ context.Context, role *domain.Role) error {
	if r.createErr == nil {
		r.created = append(r.created, role.Name)
	}
	return r.createErr
}
func (r *fakeRoleRepository) Update(_ context.Context, role *domain.Role) error {
	if r.updateErr == nil {
		updated := *role
		r.updatedRole = &updated
	}
	return r.updateErr
}
func (r *fakeRoleRepository) Delete(_ context.Context, _ uint64) error {
	if r.deleteErr == nil {
		r.deleted = true
	}
	return r.deleteErr
}
func (r *fakeRoleRepository) AssignPermissions(_ context.Context, _ uint64, permissionIDs []uint64) error {
	if r.assignErr == nil {
		r.assigned = append([]uint64(nil), permissionIDs...)
	}
	return r.assignErr
}
func (r *fakeRoleRepository) GetPermissions(context.Context, uint64) ([]domain.Permission, error) {
	return r.permissions, r.getPermissionsErr
}

func (r *fakeRoleRepository) ListOptions(_ context.Context, tenantID uint64, includePlatform bool) ([]domain.Role, error) {
	r.optionsTenantID = tenantID
	r.optionsIncludePlatform = includePlatform
	return r.options, nil
}

func (r *fakeRoleRepository) FindPermissionsByIDs(_ context.Context, ids []uint64) ([]domain.Permission, error) {
	out := make([]domain.Permission, 0, len(ids))
	for _, id := range ids {
		if permission, ok := r.permissionsByIDs[id]; ok {
			out = append(out, permission)
		} else if r.permissionsByIDs == nil {
			out = append(out, domain.Permission{ID: id, Resource: "/api/v1/test", Action: "GET"})
		}
	}
	return out, nil
}

func roleAppCode(err error) int {
	var appErr *apperrors.AppError
	if stderrors.As(err, &appErr) {
		return appErr.Code
	}
	return 0
}

// fakeTenantQuota 只返回预设的配额校验结果
type fakeTenantQuota struct {
	err           error
	checked       []uint64
	exists        bool
	existsErr     error
	existsChecked []uint64
}

func (f *fakeTenantQuota) CheckRoleQuota(_ context.Context, tenantID uint64) error {
	f.checked = append(f.checked, tenantID)
	return f.err
}

func (f *fakeTenantQuota) TenantExists(_ context.Context, tenantID uint64) (bool, error) {
	f.existsChecked = append(f.existsChecked, tenantID)
	return f.exists, f.existsErr
}

func TestRoleServiceCreateRejectsWhenQuotaExceeded(t *testing.T) {
	quota := &fakeTenantQuota{err: apperrors.New(apperrors.CodeQuotaExceeded, "roles quota exceeded")}
	repo := &fakeRoleRepository{}
	service := NewRoleService(repo).WithQuota(quota)

	_, err := service.Create(tenant.WithTenant(context.Background(), 7), CreateRoleRequest{Name: "auditor"})
	if roleAppCode(err) != apperrors.CodeQuotaExceeded {
		t.Fatalf("code = %d, want %d", roleAppCode(err), apperrors.CodeQuotaExceeded)
	}
	if len(repo.created) != 0 {
		t.Fatalf("配额超限时不应创建角色: %+v", repo.created)
	}
	if len(quota.checked) != 1 || quota.checked[0] != 7 {
		t.Fatalf("应按上下文租户校验配额, checked = %v", quota.checked)
	}
}
