package service

import (
	"context"
	"fmt"

	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"

	"github.com/xavierli/network-ticket/internal/model"
	"github.com/xavierli/network-ticket/internal/repository"
)

type UserService struct {
	userRepo *repository.UserRepo
	logger   *zap.Logger
}

func NewUserService(userRepo *repository.UserRepo, logger *zap.Logger) *UserService {
	return &UserService{userRepo: userRepo, logger: logger}
}

func (s *UserService) List(ctx context.Context, page, pageSize int) ([]model.User, int, error) {
	users, err := s.userRepo.List(ctx, page, pageSize)
	if err != nil {
		return nil, 0, err
	}
	total, err := s.userRepo.Count(ctx)
	if err != nil {
		return nil, 0, err
	}
	if users == nil {
		users = []model.User{}
	}
	return users, total, nil
}

func (s *UserService) CreateUser(ctx context.Context, username, password, role string) (*model.User, error) {
	if _, err := s.userRepo.GetByUsername(ctx, username); err == nil {
		return nil, fmt.Errorf("username already exists")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}
	u := &model.User{
		Username: username,
		Password: string(hash),
		Role:     role,
		Status:   "active",
	}
	id, err := s.userRepo.Create(ctx, u)
	if err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}
	u.ID = id
	u.Password = ""
	s.logger.Info("user created", zap.Int64("id", id), zap.String("username", username))
	return u, nil
}

func (s *UserService) UpdateUser(ctx context.Context, id int64, username, password, role string) error {
	u, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("get user: %w", err)
	}
	u.Username = username
	u.Role = role
	if password != "" {
		hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			return fmt.Errorf("hash password: %w", err)
		}
		u.Password = string(hash)
	}
	if err := s.userRepo.Update(ctx, u); err != nil {
		return fmt.Errorf("update user: %w", err)
	}
	s.logger.Info("user updated", zap.Int64("id", id))
	return nil
}

func (s *UserService) DeleteUser(ctx context.Context, id int64) error {
	if err := s.userRepo.Delete(ctx, id); err != nil {
		return fmt.Errorf("delete user: %w", err)
	}
	s.logger.Info("user deleted", zap.Int64("id", id))
	return nil
}
