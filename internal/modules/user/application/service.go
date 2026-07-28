package application

import (
	"context"

	"jimu/internal/modules/user/domain"
	"jimu/internal/shared/errors"

	"golang.org/x/crypto/bcrypt"
)

type UserService struct {
	repo domain.UserRepository
}

func NewUserService(repo domain.UserRepository) *UserService {
	return &UserService{repo: repo}
}

func (s *UserService) Create(ctx context.Context, req CreateUserRequest) (*domain.User, error) {
	existing, _ := s.repo.FindByUsername(ctx, req.Username)
	if existing != nil {
		return nil, errors.New(errors.CodeUserExists, "username already exists")
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
	return user, nil
}

func (s *UserService) Get(ctx context.Context, id uint64) (*domain.User, error) {
	user, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, errors.New(errors.CodeUserNotFound, "user not found")
	}
	return user, nil
}

func (s *UserService) List(ctx context.Context, page, pageSize int) ([]domain.User, int64, error) {
	offset := (page - 1) * pageSize
	return s.repo.List(ctx, offset, pageSize)
}

func (s *UserService) Update(ctx context.Context, id uint64, req UpdateUserRequest) error {
	user, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return errors.New(errors.CodeUserNotFound, "user not found")
	}
	if req.Status != nil {
		user.Status = *req.Status
	}
	return s.repo.Update(ctx, user)
}

func (s *UserService) Delete(ctx context.Context, id uint64) error {
	return s.repo.Delete(ctx, id)
}
