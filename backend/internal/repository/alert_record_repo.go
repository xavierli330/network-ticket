package repository

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"

	"github.com/xavierli/network-ticket/internal/model"
)

type AlertRecordRepo struct {
	db *sqlx.DB
}

func NewAlertRecordRepo(db *sqlx.DB) *AlertRecordRepo {
	return &AlertRecordRepo{db: db}
}

// Create inserts a new alert record.
func (r *AlertRecordRepo) Create(ctx context.Context, ar *model.AlertRecord) (int64, error) {
	query := `INSERT INTO alert_records (ticket_id, alert_raw, alert_parsed) VALUES (?, ?, ?)`
	result, err := r.db.ExecContext(ctx, query, ar.TicketID, ar.AlertRaw, ar.AlertParsed)
	if err != nil {
		return 0, fmt.Errorf("insert alert_record: %w", err)
	}
	id, _ := result.LastInsertId()
	return id, nil
}

// ListByTicketID returns all alert records for a given ticket.
func (r *AlertRecordRepo) ListByTicketID(ctx context.Context, ticketID int64) ([]model.AlertRecord, error) {
	var records []model.AlertRecord
	query := `SELECT * FROM alert_records WHERE ticket_id = ? ORDER BY id`
	if err := r.db.SelectContext(ctx, &records, query, ticketID); err != nil {
		return nil, fmt.Errorf("list alert_records by ticket_id: %w", err)
	}
	return records, nil
}
