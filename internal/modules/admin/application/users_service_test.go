package application

import (
	"context"
	"errors"
	"testing"
	"time"

	userdomain "jimu/internal/modules/user/domain"
	"jimu/internal/shared/pagination"

	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

// testRole 用于 roles 表的测试模型
type testRole struct {
	ID   uint64 `gorm:"primaryKey"`
	Name string
}

func (testRole) TableName() string { return "roles" }

func TestAdminUserServiceListUsers(t *testing.T) {
	ctx := context.Background()
	now := time.Now()

	svc := NewAdminUserService(&fakeUserRepository{list: func(ctx context.Context, offset, limit int, sort, order string) ([]userdomain.User, int64, error) {
		return []userdomain.User{{ID: 1, Username: "alice", Status: 1, CreatedAt: now}}, 1, nil
	}})
	users, total, err := svc.ListUsers(ctx, ListUserFilter{}, pagination.Pagination{})
	assert.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Len(t, users, 1)
	assert.Equal(t, "alice", users[0].Username)
	assert.Equal(t, now.Format("2006-01-02 15:04:05"), users[0].CreatedAt)

	// 用户名过滤
	users, _, err = svc.ListUsers(ctx, ListUserFilter{Username: "ali"}, pagination.Pagination{})
	assert.NoError(t, err)
	assert.Len(t, users, 1)

	users, _, err = svc.ListUsers(ctx, ListUserFilter{Username: "nope"}, pagination.Pagination{})
	assert.NoError(t, err)
	assert.Len(t, users, 0)

	// 状态过滤
	status := int8(1)
	users, _, err = svc.ListUsers(ctx, ListUserFilter{Status: &status}, pagination.Pagination{})
	assert.NoError(t, err)
	assert.Len(t, users, 1)
	other := int8(0)
	users, _, err = svc.ListUsers(ctx, ListUserFilter{Status: &other}, pagination.Pagination{})
	assert.NoError(t, err)
	assert.Len(t, users, 0)

	// 仓储错误
	svcErr := NewAdminUserService(&fakeUserRepository{list: func(ctx context.Context, offset, limit int, sort, order string) ([]userdomain.User, int64, error) {
		return nil, 0, errors.New("db down")
	}})
	_, _, err = svcErr.ListUsers(ctx, ListUserFilter{}, pagination.Pagination{})
	assert.Error(t, err)
}

func TestAdminUserServiceGetUser(t *testing.T) {
	ctx := context.Background()
	svc := NewAdminUserService(&fakeUserRepository{})
	user, err := svc.GetUser(ctx, 1)
	assert.NoError(t, err)
	assert.Equal(t, uint64(1), user.ID)

	// 未找到
	svc = NewAdminUserService(&fakeUserRepository{findByID: func(ctx context.Context, id uint64) (*userdomain.User, error) {
		return nil, gorm.ErrRecordNotFound
	}})
	_, err = svc.GetUser(ctx, 1)
	assert.Error(t, err)

	// 其他错误
	svc = NewAdminUserService(&fakeUserRepository{findByID: func(ctx context.Context, id uint64) (*userdomain.User, error) {
		return nil, errors.New("boom")
	}})
	_, err = svc.GetUser(ctx, 1)
	assert.Error(t, err)
}

func TestAdminUserServiceCreateUser(t *testing.T) {
	ctx := context.Background()
	svc := NewAdminUserService(&fakeUserRepository{})

	// 未指定状态时默认启用
	user, err := svc.CreateUser(ctx, AdminCreateUserRequest{Username: "bob", Password: "password123", Status: 0})
	assert.NoError(t, err)
	assert.Equal(t, int8(1), user.Status)
	assert.Equal(t, uint64(7), user.ID)

	// 指定状态
	user, err = svc.CreateUser(ctx, AdminCreateUserRequest{Username: "bob", Password: "password123", Status: 2})
	assert.NoError(t, err)
	assert.Equal(t, int8(2), user.Status)

	// 仓储错误
	svc = NewAdminUserService(&fakeUserRepository{create: func(ctx context.Context, user *userdomain.User) error {
		return errors.New("conflict")
	}})
	_, err = svc.CreateUser(ctx, AdminCreateUserRequest{Username: "bob", Password: "password123"})
	assert.Error(t, err)
}

func TestAdminUserServiceUpdateUser(t *testing.T) {
	ctx := context.Background()
	svc := NewAdminUserService(&fakeUserRepository{})

	// 更新状态
	err := svc.UpdateUser(ctx, 1, AdminUpdateUserRequest{Status: int8Ptr(0)})
	assert.NoError(t, err)

	// 不更新状态（nil）
	err = svc.UpdateUser(ctx, 1, AdminUpdateUserRequest{})
	assert.NoError(t, err)

	// 未找到
	svc = NewAdminUserService(&fakeUserRepository{findByID: func(ctx context.Context, id uint64) (*userdomain.User, error) {
		return nil, gorm.ErrRecordNotFound
	}})
	err = svc.UpdateUser(ctx, 1, AdminUpdateUserRequest{Status: int8Ptr(0)})
	assert.Error(t, err)

	// 更新仓储错误
	svc = NewAdminUserService(&fakeUserRepository{update: func(ctx context.Context, user *userdomain.User) error {
		return errors.New("boom")
	}})
	err = svc.UpdateUser(ctx, 1, AdminUpdateUserRequest{Status: int8Ptr(0)})
	assert.Error(t, err)
}

func TestAdminUserServiceDisableUser(t *testing.T) {
	ctx := context.Background()
	svc := NewAdminUserService(&fakeUserRepository{})
	assert.NoError(t, svc.DisableUser(ctx, 1))
}

func TestAdminUserServiceAssignRoles(t *testing.T) {
	ctx := context.Background()
	db := newSqliteDB(t, &userRole{}, &testRole{})
	assert.NoError(t, db.Create(&testRole{ID: 1, Name: "admin"}).Error)
	assert.NoError(t, db.Create(&testRole{ID: 2, Name: "viewer"}).Error)

	svc := NewAdminUserService(&fakeUserRepository{}, db)

	// 成功分配角色
	err := svc.AssignRoles(ctx, 1, []string{"admin", "viewer"})
	assert.NoError(t, err)
	var count int64
	assert.NoError(t, db.Model(&userRole{}).Where("user_id = ?", 1).Count(&count).Error)
	assert.Equal(t, int64(2), count)

	// 空角色列表：清空后仍成功
	err = svc.AssignRoles(ctx, 1, nil)
	assert.NoError(t, err)

	// 角色不存在
	err = svc.AssignRoles(ctx, 1, []string{"nope"})
	assert.Error(t, err)

	// 用户不存在
	svc = NewAdminUserService(&fakeUserRepository{findByID: func(ctx context.Context, id uint64) (*userdomain.User, error) {
		return nil, gorm.ErrRecordNotFound
	}}, db)
	err = svc.AssignRoles(ctx, 1, []string{"admin"})
	assert.Error(t, err)

	// 用户查询其他错误
	svc = NewAdminUserService(&fakeUserRepository{findByID: func(ctx context.Context, id uint64) (*userdomain.User, error) {
		return nil, errors.New("boom")
	}}, db)
	err = svc.AssignRoles(ctx, 1, []string{"admin"})
	assert.Error(t, err)

	// db 未配置
	svc = NewAdminUserService(&fakeUserRepository{})
	err = svc.AssignRoles(ctx, 1, []string{"admin"})
	assert.Error(t, err)
}

func TestAdminUserServiceAssignRolesRoleQueryError(t *testing.T) {
	// 构造角色表查询失败场景：迁移后立即断连的 sqlite 很难模拟，使用不含 roles 表的 db
	ctx := context.Background()
	db := newSqliteDB(t, &userRole{})
	svc := NewAdminUserService(&fakeUserRepository{}, db)
	err := svc.AssignRoles(ctx, 1, []string{"admin"})
	// roles 表不存在时查询报错，返回内部错误
	assert.Error(t, err)
}
