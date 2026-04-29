package repository

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"

	"github.com/xavierli/network-ticket/internal/model"
)

type TicketTypeRepo struct {
	db *sqlx.DB
}

func NewTicketTypeRepo(db *sqlx.DB) *TicketTypeRepo {
	return &TicketTypeRepo{db: db}
}

// Create inserts a new ticket type and returns the ID.
func (r *TicketTypeRepo) Create(ctx context.Context, tt *model.TicketType) (int64, error) {
	query := `INSERT INTO ticket_types (code, name, description, color, status)
		VALUES (?, ?, ?, ?, ?)`
	result, err := r.db.ExecContext(ctx, query,
		tt.Code, tt.Name, tt.Description, tt.Color, tt.Status,
	)
	if err != nil {
		return 0, fmt.Errorf("insert ticket_type: %w", err)
	}
	id, _ := result.LastInsertId()
	return id, nil
}

// GetByID returns a ticket type by its primary key.
func (r *TicketTypeRepo) GetByID(ctx context.Context, id int64) (*model.TicketType, error) {
	var tt model.TicketType
	query := `SELECT * FROM ticket_types WHERE id = ?`
	if err := r.db.GetContext(ctx, &tt, query, id); err != nil {
		return nil, fmt.Errorf("get ticket_type by id: %w", err)
	}
	return &tt, nil
}

// GetByCode returns a ticket type by its code.
func (r *TicketTypeRepo) GetByCode(ctx context.Context, code string) (*model.TicketType, error) {
	var tt model.TicketType
	query := `SELECT * FROM ticket_types WHERE code = ?`
	if err := r.db.GetContext(ctx, &tt, query, code); err != nil {
		return nil, fmt.Errorf("get ticket_type by code: %w", err)
	}
	return &tt, nil
}

// List returns all ticket types ordered by id.
func (r *TicketTypeRepo) List(ctx context.Context) ([]model.TicketType, error) {
	var types []model.TicketType
	query := `SELECT * FROM ticket_types ORDER BY id`
	if err := r.db.SelectContext(ctx, &types, query); err != nil {
		return nil, fmt.Errorf("list ticket_types: %w", err)
	}
	return types, nil
}

// Update updates a ticket type.
func (r *TicketTypeRepo) Update(ctx context.Context, tt *model.TicketType) error {
	query := `UPDATE ticket_types SET
		code = ?, name = ?, description = ?, color = ?, status = ?
		WHERE id = ?`
	if _, err := r.db.ExecContext(ctx, query,
		tt.Code, tt.Name, tt.Description, tt.Color, tt.Status, tt.ID,
	); err != nil {
		return fmt.Errorf("update ticket_type: %w", err)
	}
	return nil
}

// Delete removes a ticket type by ID.
func (r *TicketTypeRepo) Delete(ctx context.Context, id int64) error {
	query := `DELETE FROM ticket_types WHERE id = ?`
	if _, err := r.db.ExecContext(ctx, query, id); err != nil {
		return fmt.Errorf("delete ticket_type: %w", err)
	}
	return nil
}

// CountTicketsByType returns the number of tickets associated with a ticket type.
func (r *TicketTypeRepo) CountTicketsByType(ctx context.Context, ticketTypeID int64) (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM tickets WHERE ticket_type_id = ?`
	if err := r.db.GetContext(ctx, &count, query, ticketTypeID); err != nil {
		return 0, fmt.Errorf("count tickets by type: %w", err)
	}
	return count, nil
}
