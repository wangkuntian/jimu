package application

import (
	"context"

	authdomain "jimu/internal/modules/auth/domain"
	userdomain "jimu/internal/modules/user/domain"
	"jimu/internal/platform/auth"
	"jimu/internal/shared/errors"

	"golang.org/x/crypto/bcrypt"
)

type AuthService struct {
	userRepo  userdomain.UserRepository
	jwtUtil   *auth.JWT
	accessMin int
}

func NewAuthService(userRepo userdomain.UserRepository, jwtUtil *auth.JWT, accessMin int) *AuthService {
	return &AuthService{
		userRepo:  userRepo,
		jwtUtil:   jwtUtil,
		accessMin: accessMin,
	}
}

func (s *AuthService) Login(ctx context.Context, username, password string) (*authdomain.TokenPair, error) {
	user, err := s.userRepo.FindByUsername(ctx, username)
	if err != nil {
		return nil, errors.New(errors.CodeUserNotFound, "user not found")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return nil, errors.New(errors.CodeInvalidPassword, "invalid password")
	}

	accessToken, refreshToken, err := s.jwtUtil.Generate(user.ID)
	if err != nil {
		return nil, errors.Wrap(errors.CodeInternalError, "failed to generate token", err)
	}

	return &authdomain.TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    s.accessMin * 60,
	}, nil
}

func (s *AuthService) Register(ctx context.Context, username, password string) (*userdomain.User, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, errors.Wrap(errors.CodeInternalError, "failed to hash password", err)
	}

	user := &userdomain.User{
		Username: username,
		Password: string(hashedPassword),
		Status:   1,
	}
	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, errors.Wrap(errors.CodeInternalError, "failed to create user", err)
	}
	return user, nil
}

func (s *AuthService) RefreshToken(ctx context.Context, refreshToken string) (*authdomain.TokenPair, error) {
	claims, err := s.jwtUtil.Parse(refreshToken)
	if err != nil {
		return nil, errors.New(errors.CodeUnauthorized, "invalid refresh token")
	}

	accessToken, newRefreshToken, err := s.jwtUtil.Generate(claims.UserID)
	if err != nil {
		return nil, errors.Wrap(errors.CodeInternalError, "failed to generate token", err)
	}

	return &authdomain.TokenPair{
		AccessToken:  accessToken,
		RefreshToken: newRefreshToken,
		ExpiresIn:    s.accessMin * 60,
	}, nil
}
