package application

import (
	"context"
	"encoding/json"
	stderrors "errors"
	"fmt"
	"log"
	"time"

	"jimu/internal/contract"
	"jimu/internal/modules/user/domain"
	"jimu/internal/platform/cache"
	"jimu/internal/platform/encryption"
	"jimu/internal/platform/outbox"
	"jimu/internal/shared/errors"
	"jimu/internal/shared/pagination"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type UserService struct {
	repo   domain.UserRepository
	cache  cache.Cache
	outbox *outbox.Outbox
	cipher *encryption.Cipher
}

func NewUserService(repo domain.UserRepository, cache cache.Cache, deps ...interface{}) *UserService {
	s := &UserService{repo: repo, cache: cache}
	for _, dep := range deps {
		switch d := dep.(type) {
		case *outbox.Outbox:
			s.outbox = d
		case *encryption.Cipher:
			s.cipher = d
		}
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

	// 邮箱查重（盲索引精确查询）；空邮箱跳过，DB unique 索引兜底
	if req.Email != "" && s.cipher != nil {
		existing, err := s.repo.FindByEmailHash(ctx, s.cipher.BlindIndex(req.Email))
		if err == nil && existing != nil {
			return nil, errors.New(errors.CodeConflict, "email already exists")
		}
		if err != nil && !stderrors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.Wrap(errors.CodeInternalError, "failed to find user by email", err)
		}
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, errors.Wrap(errors.CodeInternalError, "failed to hash password", err)
	}

	user := &domain.User{
		Username: req.Username,
		Password: string(hashedPassword),
		Email:    req.Email,
		Phone:    req.Phone,
		Status:   1,
	}
	if err := s.repo.Create(ctx, user); err != nil {
		return nil, errors.Wrap(errors.CodeInternalError, "failed to create user", err)
	}
	resp := ToUserResponse(*user)

	// 写入 Outbox（统一事件投递路径，确保可靠投递）。
	// 业务事务已提交，outbox 写失败不回滚业务，但必须记录，避免静默丢事件。
	if s.outbox != nil {
		payload, err := json.Marshal(contract.UserCreatedEvent{
			UserID:   user.ID,
			Username: user.Username,
			Email:    user.Email,
		})
		if err != nil {
			log.Printf("user: marshal created event: %v", err)
		} else if err := s.outbox.Add(ctx, nil, outbox.Event{
			AggregateID: fmt.Sprintf("user:%d", user.ID),
			EventType:   contract.EventUserCreated,
			Payload:     payload,
		}); err != nil {
			log.Printf("user: write outbox event %s: %v", contract.EventUserCreated, err)
		}
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

	if s.outbox != nil {
		payload, err := json.Marshal(contract.UserUpdatedEvent{
			UserID:  id,
			Changes: []string{"status"},
		})
		if err != nil {
			log.Printf("user: marshal updated event: %v", err)
		} else if err := s.outbox.Add(ctx, nil, outbox.Event{
			AggregateID: fmt.Sprintf("user:%d", id),
			EventType:   contract.EventUserUpdated,
			Payload:     payload,
		}); err != nil {
			log.Printf("user: write outbox event %s: %v", contract.EventUserUpdated, err)
		}
	}
	return nil
}

func (s *UserService) Delete(ctx context.Context, id uint64) error {
	// 先加载再删除，未找到映射为 CodeNotFound
	user, err := s.repo.FindByID(ctx, id)
	if err != nil {
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			return errors.Wrap(errors.CodeNotFound, "user not found", err)
		}
		return errors.Wrap(errors.CodeInternalError, "failed to get user", err)
	}
	if err := s.repo.Delete(ctx, user.ID); err != nil {
		return errors.Wrap(errors.CodeInternalError, "failed to delete user", err)
	}
	s.invalidateUserCache(ctx, id)

	if s.outbox != nil {
		payload, err := json.Marshal(contract.UserDeletedEvent{
			UserID: id,
		})
		if err != nil {
			log.Printf("user: marshal deleted event: %v", err)
		} else if err := s.outbox.Add(ctx, nil, outbox.Event{
			AggregateID: fmt.Sprintf("user:%d", id),
			EventType:   contract.EventUserDeleted,
			Payload:     payload,
		}); err != nil {
			log.Printf("user: write outbox event %s: %v", contract.EventUserDeleted, err)
		}
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
