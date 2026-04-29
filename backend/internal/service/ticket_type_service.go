package service

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	"github.com/xavierli/network-ticket/internal/model"
	"github.com/xavierli/network-ticket/internal/repository"
)

type TicketTypeService struct {
	ticketTypeRepo *repository.TicketTypeRepo
	logger         *zap.Logger
}

func NewTicketTypeService(ticketTypeRepo *repository.TicketTypeRepo, logger *zap.Logger) *TicketTypeService {
	return &TicketTypeService{ticketTypeRepo: ticketTypeRepo, logger: logger}
}

// List returns all ticket types.
func (s *TicketTypeService) List(ctx context.Context) ([]model.TicketType, error) {
	types, err := s.ticketTypeRepo.List(ctx)
	if err != nil {
		return nil, err
	}
	if types == nil {
		types = []model.TicketType{}
	}
	return types, nil
}

// Create creates a new ticket type.
func (s *TicketTypeService) Create(ctx context.Context, code, name string, description *string, color, status string) (*model.TicketType, error) {
	// Check for duplicate code.
	if _, err := s.ticketTypeRepo.GetByCode(ctx, code); err == nil {
		return nil, fmt.Errorf("code already exists")
	}

	if status == "" {
		status = "active"
	}
	if color == "" {
		color = "#6B7280"
	}

	tt := &model.TicketType{
		Code:        code,
		Name:        name,
		Description: description,
		Color:       color,
		Status:      status,
	}
	id, err := s.ticketTypeRepo.Create(ctx, tt)
	if err != nil {
		return nil, fmt.Errorf("create ticket type: %w", err)
	}
	tt.ID = id
	s.logger.Info("ticket type created", zap.Int64("id", id), zap.String("code", code))
	return tt, nil
}

// Update updates an existing ticket type.
func (s *TicketTypeService) Update(ctx context.Context, id int64, code, name string, description *string, color, status string) error {
	// Verify the type exists.
	if _, err := s.ticketTypeRepo.GetByID(ctx, id); err != nil {
		return fmt.Errorf("ticket type not found")
	}

	// Check code uniqueness if changing code.
	existing, err := s.ticketTypeRepo.GetByCode(ctx, code)
	if err == nil && existing.ID != id {
		return fmt.Errorf("code already exists")
	}

	tt := &model.TicketType{
		ID:          id,
		Code:        code,
		Name:        name,
		Description: description,
		Color:       color,
		Status:      status,
	}
	if err := s.ticketTypeRepo.Update(ctx, tt); err != nil {
		return fmt.Errorf("update ticket type: %w", err)
	}
	s.logger.Info("ticket type updated", zap.Int64("id", id))
	return nil
}

// Delete deletes a ticket type if it has no associated tickets.
func (s *TicketTypeService) Delete(ctx context.Context, id int64) error {
	count, err := s.ticketTypeRepo.CountTicketsByType(ctx, id)
	if err != nil {
		return fmt.Errorf("check ticket type usage: %w", err)
	}
	if count > 0 {
		return fmt.Errorf("ticket type is in use")
	}
	if err := s.ticketTypeRepo.Delete(ctx, id); err != nil {
		return fmt.Errorf("delete ticket type: %w", err)
	}
	s.logger.Info("ticket type deleted", zap.Int64("id", id))
	return nil
}
