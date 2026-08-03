package application

import (
	"context"
	stderrors "errors"

	"jimu/internal/modules/user/domain"
	"jimu/internal/shared/errors"
	"jimu/internal/shared/pagination"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type UserService struct {
	repo domain.UserRepository
}

func NewUserService(repo domain.UserRepository) *UserService {
	return &UserService{repo: repo}
}

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
	return &resp, nil
}

func (s *UserService) Get(ctx context.Context, id uint64) (*UserResponse, error) {
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
	return nil
}

func (s *UserService) Delete(ctx context.Context, id uint64) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		return errors.Wrap(errors.CodeInternalError, "failed to delete user", err)
	}
	return nil
}
