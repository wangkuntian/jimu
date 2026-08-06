package application

import (
	"context"
	stderrors "errors"
	"fmt"
	"time"

	"jimu/internal/contract"
	"jimu/internal/modules/user/domain"
	"jimu/internal/platform/cache"
	"jimu/internal/platform/event"
	"jimu/internal/shared/errors"
	"jimu/internal/shared/pagination"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type UserService struct {
	repo     domain.UserRepository
	cache    cache.Cache
	eventBus *event.EventBus
}

func NewUserService(repo domain.UserRepository, cache cache.Cache, eventBus ...*event.EventBus) *UserService {
	s := &UserService{repo: repo, cache: cache}
	if len(eventBus) > 0 {
		s.eventBus = eventBus[0]
	}
	return s
}

const userCacheTTL = 5 * time.Minute

func (s *UserService) Create(ctx context.Context, req CreateUserRequest) (*UserResponse, error) {
	existing, err := s.repo.FindByUsername(ctx, req.Username)
	if err == nil && existing != nil {
		return nil, errors.New(errors.CodeConflict, "username already exists")
	}
	if err != nil && !stderrors.Is(err, gorm.ErrRecordNotFound) {
		return nil, errors.Wrap(errors.CodeInternalError, "failed to find user", err)
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, errors.Wrap(errors.CodeInternalError, "failed to hash password", err)
	}

	user := &domain.User{
		Username: req.Username,
		Password: string(hashedPassword),
		Status:   1,
	}
	if err := s.repo.Create(ctx, user); err != nil {
		return nil, errors.Wrap(errors.CodeInternalError, "failed to create user", err)
	}
	resp := ToUserResponse(*user)

	// 发布用户创建事件（异步，不影响主流程）
	if s.eventBus != nil {
		s.eventBus.PublishAsync(contract.EventUserCreated, contract.UserCreatedEvent{
			UserID:   user.ID,
			Username: user.Username,
		})
	}

	return &resp, nil
}

func (s *UserService) Get(ctx context.Context, id uint64) (*UserResponse, error) {
	// Cache-Aside: 先查缓存
	if s.cache != nil {
		cacheKey := fmt.Sprintf("user:id:%d", id)
		var resp UserResponse
		if found, _ := s.cache.Get(ctx, cacheKey, &resp); found {
			return &resp, nil
		}
	}

	user, err := s.repo.FindByID(ctx, id)
	if err != nil {
		code := errors.CodeInternalError
		message := "failed to get user"
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			code = errors.CodeNotFound
			message = "user not found"
		}
		return nil, errors.Wrap(code, message, err)
	}
	resp := ToUserResponse(*user)

	// 写入缓存
	if s.cache != nil {
		_ = s.cache.Set(ctx, fmt.Sprintf("user:id:%d", id), resp, userCacheTTL)
	}
	return &resp, nil
}

func (s *UserService) List(ctx context.Context, p pagination.Pagination) ([]UserResponse, int64, error) {
	users, total, err := s.repo.List(ctx, p.GetOffset(), p.GetLimit(), p.Sort, p.Order)
	if err != nil {
		return nil, 0, errors.Wrap(errors.CodeInternalError, "failed to list users", err)
	}
	return ToUserResponses(users), total, nil
}

func (s *UserService) Update(ctx context.Context, id uint64, req UpdateUserRequest) error {
	user, err := s.repo.FindByID(ctx, id)
	if err != nil {
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			return errors.Wrap(errors.CodeNotFound, "user not found", err)
		}
		return errors.Wrap(errors.CodeInternalError, "failed to get user", err)
	}
	if req.Status != nil {
		user.Status = *req.Status
	}
	if err := s.repo.Update(ctx, user); err != nil {
		return errors.Wrap(errors.CodeInternalError, "failed to update user", err)
	}
	s.invalidateUserCache(ctx, id)

	if s.eventBus != nil {
		s.eventBus.PublishAsync(contract.EventUserUpdated, contract.UserUpdatedEvent{
			UserID:  id,
			Changes: []string{"status"},
		})
	}
	return nil
}

func (s *UserService) Delete(ctx context.Context, id uint64) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		return errors.Wrap(errors.CodeInternalError, "failed to delete user", err)
	}
	s.invalidateUserCache(ctx, id)

	if s.eventBus != nil {
		s.eventBus.PublishAsync(contract.EventUserDeleted, contract.UserDeletedEvent{
			UserID: id,
		})
	}
	return nil
}

// invalidateUserCache 清除用户相关缓存
func (s *UserService) invalidateUserCache(ctx context.Context, id uint64) {
	if s.cache == nil {
		return
	}
	_ = s.cache.Delete(ctx, fmt.Sprintf("user:id:%d", id))
	_ = s.cache.DeletePattern(ctx, "user:list:*")
}

// BatchDelete 批量删除用户（逐个软删除，返回成功/失败计数）
func (s *UserService) BatchDelete(ctx context.Context, ids []uint64) (BatchResult, error) {
	result := BatchResult{}
	for _, id := range ids {
		if err := s.Delete(ctx, id); err != nil {
			result.Failed++
			continue
		}
		result.Success++
	}
	if result.Success == 0 && result.Failed > 0 {
		return result, errors.Wrap(errors.CodeInternalError, "batch delete failed", nil)
	}
	return result, nil
}
