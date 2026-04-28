package repository

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"

	"github.com/xavierli/network-ticket/internal/model"
)

type TicketLogRepo struct {
	db *sqlx.DB
}

func NewTicketLogRepo(db *sqlx.DB) *TicketLogRepo {
	return &TicketLogRepo{db: db}
}

// Create inserts a new ticket log entry.
func (r *TicketLogRepo) Create(ctx context.Context, tl *model.TicketLog) (int64, error) {
	query := `INSERT INTO ticket_logs (ticket_id, action, from_state, to_state, operator, detail)
		VALUES (?, ?, ?, ?, ?, ?)`
	result, err := r.db.ExecContext(ctx, query,
		tl.TicketID, tl.Action, tl.FromState, tl.ToState, tl.Operator, tl.Detail,
	)
	if err != nil {
		return 0, fmt.Errorf("insert ticket_log: %w", err)
	}
	id, _ := result.LastInsertId()
	return id, nil
}

// ListByTicketID returns all log entries for a given ticket.
func (r *TicketLogRepo) ListByTicketID(ctx context.Context, ticketID int64) ([]model.TicketLog, error) {
	var logs []model.TicketLog
	query := `SELECT * FROM ticket_logs WHERE ticket_id = ? ORDER BY id`
	if err := r.db.SelectContext(ctx, &logs, query, ticketID); err != nil {
		return nil, fmt.Errorf("list ticket_logs by ticket_id: %w", err)
	}
	return logs, nil
}
