package application

import (
	"context"
	"encoding/json"
	stderrors "errors"
	"testing"
	"time"

	"jimu/internal/contract"
	"jimu/internal/kernel/access"
	"jimu/internal/kernel/auth"
	apperrors "jimu/internal/shared/errors"

	"github.com/stretchr/testify/require"
)

// fakeAuthorizationStore 内存版 access.AuthorizationStore：分别可注入角色/策略错误。
type fakeAuthorizationStore struct {
	roles    []access.RoleIdentity
	policies []access.Policy
	rolesErr error
	polErr   error
}

func (s *fakeAuthorizationStore) RolesForUser(context.Context, uint64) ([]access.RoleIdentity, error) {
	if s.rolesErr != nil {
		return nil, s.rolesErr
	}
	return s.roles, nil
}

func (s *fakeAuthorizationStore) Policies(context.Context) ([]access.Policy, error) {
	if s.polErr != nil {
		return nil, s.polErr
	}
	return s.policies, nil
}

func newMeService(repo contract.AccountRepository, authz access.AuthorizationStore) *AuthService {
	return NewAuthService(repo, auth.New("01234567890123456789012345678901", "jimu", 30, 7), newFakeSessionStore(), nil, 30, authz)
}

func TestMeFailsWhenUserMissing(t *testing.T) {
	service := newMeService(&fakeUserRepo{users: map[string]*contract.Account{}}, &fakeAuthorizationStore{})

	_, err := service.Me(context.Background(), 42, 7)
	require.Equal(t, apperrors.CodeUserNotFound, appCode(err), "err = %v", err)
}

// TestMeMapsNonNotFoundErrorsToInternal 只有「确实不存在」才是 404；
// DB 故障等其它错误必须保留为内部错误，否则会话引导会把瞬时故障误报成账号被删。
func TestMeMapsNonNotFoundErrorsToInternal(t *testing.T) {
	service := newMeService(&fakeUserRepo{findByIDErr: stderrors.New("db down")}, &fakeAuthorizationStore{})

	_, err := service.Me(context.Background(), 42, 7)
	require.Equal(t, apperrors.CodeInternalError, appCode(err), "err = %v", err)
}

func TestMeFailsWhenRolesLookupFails(t *testing.T) {
	repo := &fakeUserRepo{users: map[string]*contract.Account{
		"alice": {ID: 42, Username: "alice", Status: 1},
	}}
	service := newMeService(repo, &fakeAuthorizationStore{rolesErr: stderrors.New("roles down")})

	_, err := service.Me(context.Background(), 42, 7)
	require.Equal(t, apperrors.CodeInternalError, appCode(err), "err = %v", err)
}

func TestMeFailsWhenPoliciesLookupFails(t *testing.T) {
	repo := &fakeUserRepo{users: map[string]*contract.Account{
		"alice": {ID: 42, Username: "alice", Status: 1},
	}}
	service := newMeService(repo, &fakeAuthorizationStore{polErr: stderrors.New("policies down")})

	_, err := service.Me(context.Background(), 42, 7)
	require.Equal(t, apperrors.CodeInternalError, appCode(err), "err = %v", err)
}

// TestMeDegradesWithoutAuthorizationStore 未装配 store 时仍须成功（降级分支），
// 且 roles / permissions 序列化为 [] 而不是 null —— 前端按数组渲染。
func TestMeDegradesWithoutAuthorizationStore(t *testing.T) {
	repo := &fakeUserRepo{users: map[string]*contract.Account{
		"alice": {ID: 42, Username: "alice", Status: 1},
	}}
	service := newMeService(repo, nil)

	view, err := service.Me(context.Background(), 42, 7)
	require.NoError(t, err)
	require.NotNil(t, view.Roles)
	require.NotNil(t, view.Permissions)
	require.Empty(t, view.Roles)
	require.Empty(t, view.Permissions)

	raw, err := json.Marshal(view)
	require.NoError(t, err)
	require.Contains(t, string(raw), `"roles":[]`)
	require.Contains(t, string(raw), `"permissions":[]`)
}

// TestMeSerializesNilRolesAsEmptyArray 生产实现里 RolesForUser 对零角色用户返回 nil
// （GORM Pluck 零行），服务层必须归一化为空切片，响应的 roles 才是 [] 而不是 null。
// 这是公开注册用户的真实路径：store 存在、角色为空、策略表非空。
func TestMeSerializesNilRolesAsEmptyArray(t *testing.T) {
	repo := &fakeUserRepo{users: map[string]*contract.Account{
		"roleless": {ID: 42, Username: "roleless", Status: 1},
	}}
	authz := &fakeAuthorizationStore{
		roles: nil,
		policies: []access.Policy{
			{Role: access.RoleSubject(1), Resource: "/api/v1/tenants", Action: "GET"},
		},
	}
	service := newMeService(repo, authz)

	view, err := service.Me(context.Background(), 42, 7)
	require.NoError(t, err)
	require.NotNil(t, view.Roles)
	require.Empty(t, view.Roles)
	require.Empty(t, view.Permissions)

	raw, err := json.Marshal(view)
	require.NoError(t, err)
	require.Contains(t, string(raw), `"roles":[]`)
	require.Contains(t, string(raw), `"permissions":[]`)
}

// TestMeReturnsClaimTenantAndEffectivePermissions 租户取参数（已验签 claim）而非账号行：
// 账号行 TenantID 故意与参数不同，返回值必须是参数值。
func TestMeReturnsClaimTenantAndEffectivePermissions(t *testing.T) {
	created := time.Date(2026, 10, 13, 9, 30, 0, 0, time.UTC)
	repo := &fakeUserRepo{users: map[string]*contract.Account{
		"alice": {ID: 42, Username: "alice", Status: 1, TenantID: 99, CreatedAt: created},
	}}
	authz := &fakeAuthorizationStore{
		roles: []access.RoleIdentity{{ID: 1, Name: "超级管理员", Scope: contract.RoleScopePlatform}},
		policies: []access.Policy{
			{Role: access.RoleSubject(1), Resource: "/api/v1/tenants", Action: "GET"},
			{Role: access.RoleSubject(1), Resource: "/api/v1/users", Action: "GET"},
			{Role: access.RoleSubject(2), Resource: "/api/v1/audits", Action: "GET"},
		},
	}
	service := newMeService(repo, authz)

	view, err := service.Me(context.Background(), 42, 7)
	require.NoError(t, err)
	require.Equal(t, uint64(42), view.User.ID)
	require.Equal(t, "alice", view.User.Username)
	require.EqualValues(t, 1, view.User.Status)
	require.Equal(t, created, view.User.CreatedAt)
	require.EqualValues(t, 7, view.User.TenantID, "租户取 claim 参数而非账号行 TenantID=99")
	require.Equal(t, []string{"超级管理员"}, view.Roles)
	require.Equal(t, []PermissionPoint{
		{Resource: "/api/v1/tenants", Action: "GET"},
		{Resource: "/api/v1/users", Action: "GET"},
	}, view.Permissions)
}
