package service

import (
	"context"

	"go.uber.org/zap"

	"github.com/xavierli/network-ticket/internal/model"
	"github.com/xavierli/network-ticket/internal/repository"
)

// ClientService provides CRUD operations for clients.
type ClientService struct {
	clientRepo *repository.ClientRepo
	logger     *zap.Logger
}

// NewClientService creates a new ClientService.
func NewClientService(clientRepo *repository.ClientRepo, logger *zap.Logger) *ClientService {
	return &ClientService{
		clientRepo: clientRepo,
		logger:     logger,
	}
}

// List returns all clients.
func (s *ClientService) List(ctx context.Context) ([]model.Client, error) {
	return s.clientRepo.List(ctx)
}

// GetByID returns a client by its primary key.
func (s *ClientService) GetByID(ctx context.Context, id int64) (*model.Client, error) {
	return s.clientRepo.GetByID(ctx, id)
}

// Create inserts a new client.
func (s *ClientService) Create(ctx context.Context, c *model.Client) (int64, error) {
	id, err := s.clientRepo.Create(ctx, c)
	if err != nil {
		return 0, err
	}
	s.logger.Info("client created", zap.Int64("id", id), zap.String("name", c.Name))
	return id, nil
}

// Update updates a client.
func (s *ClientService) Update(ctx context.Context, c *model.Client) error {
	if err := s.clientRepo.Update(ctx, c); err != nil {
		return err
	}
	s.logger.Info("client updated", zap.Int64("id", c.ID))
	return nil
}

// Delete removes a client by ID.
func (s *ClientService) Delete(ctx context.Context, id int64) error {
	if err := s.clientRepo.Delete(ctx, id); err != nil {
		return err
	}
	s.logger.Info("client deleted", zap.Int64("id", id))
	return nil
}
