package repository

import (
	"context"
	"fmt"
	"strings"

	"github.com/jmoiron/sqlx"

	"github.com/xavierli/network-ticket/internal/model"
)

type TicketRepo struct {
	db *sqlx.DB
}

func NewTicketRepo(db *sqlx.DB) *TicketRepo {
	return &TicketRepo{db: db}
}

// Create inserts a new ticket and returns the ID.
func (r *TicketRepo) Create(ctx context.Context, t *model.Ticket) (int64, error) {
	query := `INSERT INTO tickets
		(ticket_no, alert_source_id, source_type, ticket_type_id, alert_raw, alert_parsed, title, description,
		 severity, status, client_id, external_id, callback_data, fingerprint, timeout_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	result, err := r.db.ExecContext(ctx, query,
		t.TicketNo, t.AlertSourceID, t.SourceType, t.TicketTypeID, t.AlertRaw, t.AlertParsed, t.Title, t.Description,
		t.Severity, t.Status, t.ClientID, t.ExternalID, t.CallbackData, t.Fingerprint, t.TimeoutAt,
	)
	if err != nil {
		return 0, fmt.Errorf("insert ticket: %w", err)
	}
	id, _ := result.LastInsertId()
	return id, nil
}

// UpdateTicketTypeID sets the ticket_type_id for a ticket.
func (r *TicketRepo) UpdateTicketTypeID(ctx context.Context, ticketID int64, ticketTypeID *int64) error {
	query := `UPDATE tickets SET ticket_type_id = ? WHERE id = ?`
	if _, err := r.db.ExecContext(ctx, query, ticketTypeID, ticketID); err != nil {
		return fmt.Errorf("update ticket type id: %w", err)
	}
	return nil
}

// GetByID returns a ticket by its primary key.
func (r *TicketRepo) GetByID(ctx context.Context, id int64) (*model.Ticket, error) {
	var t model.Ticket
	query := `SELECT * FROM tickets WHERE id = ?`
	if err := r.db.GetContext(ctx, &t, query, id); err != nil {
		return nil, fmt.Errorf("get ticket by id: %w", err)
	}
	return &t, nil
}

// GetByTicketNo returns a ticket by its ticket number.
func (r *TicketRepo) GetByTicketNo(ctx context.Context, ticketNo string) (*model.Ticket, error) {
	var t model.Ticket
	query := `SELECT * FROM tickets WHERE ticket_no = ?`
	if err := r.db.GetContext(ctx, &t, query, ticketNo); err != nil {
		return nil, fmt.Errorf("get ticket by ticket_no: %w", err)
	}
	return &t, nil
}

// GetByFingerprint returns the most recent pending or in_progress ticket with the given fingerprint.
func (r *TicketRepo) GetByFingerprint(ctx context.Context, fingerprint string) (*model.Ticket, error) {
	var t model.Ticket
	query := `SELECT * FROM tickets
		WHERE fingerprint = ? AND status IN ('pending', 'in_progress')
		ORDER BY created_at DESC LIMIT 1`
	if err := r.db.GetContext(ctx, &t, query, fingerprint); err != nil {
		return nil, fmt.Errorf("get ticket by fingerprint: %w", err)
	}
	return &t, nil
}

// UpdateStatus updates the ticket status.
func (r *TicketRepo) UpdateStatus(ctx context.Context, id int64, status string) error {
	query := `UPDATE tickets SET status = ? WHERE id = ?`
	if _, err := r.db.ExecContext(ctx, query, status, id); err != nil {
		return fmt.Errorf("update ticket status: %w", err)
	}
	return nil
}

// List returns a paginated list of tickets matching the filter, along with the total count.
func (r *TicketRepo) List(ctx context.Context, f model.TicketFilter) ([]model.Ticket, int, error) {
	var conditions []string
	var args []interface{}

	if f.Status != "" {
		conditions = append(conditions, "status = ?")
		args = append(args, f.Status)
	}
	if f.ClientID != 0 {
		conditions = append(conditions, "client_id = ?")
		args = append(args, f.ClientID)
	}
	if f.Severity != "" {
		conditions = append(conditions, "severity = ?")
		args = append(args, f.Severity)
	}
	if f.TicketTypeID != 0 {
		conditions = append(conditions, "ticket_type_id = ?")
		args = append(args, f.TicketTypeID)
	}
	if f.Keyword != "" {
		conditions = append(conditions, "(title LIKE ? OR description LIKE ?)")
		kw := "%" + f.Keyword + "%"
		args = append(args, kw, kw)
	}

	where := ""
	if len(conditions) > 0 {
		where = "WHERE " + strings.Join(conditions, " AND ")
	}

	// Count total.
	countQuery := "SELECT COUNT(*) FROM tickets " + where
	var total int
	if err := r.db.GetContext(ctx, &total, countQuery, args...); err != nil {
		return nil, 0, fmt.Errorf("count tickets: %w", err)
	}

	// Fetch page.
	if f.Page < 1 {
		f.Page = 1
	}
	if f.PageSize < 1 {
		f.PageSize = 20
	}
	offset := (f.Page - 1) * f.PageSize
	listQuery := "SELECT * FROM tickets " + where + " ORDER BY created_at DESC LIMIT ? OFFSET ?"
	args = append(args, f.PageSize, offset)

	var tickets []model.Ticket
	if err := r.db.SelectContext(ctx, &tickets, listQuery, args...); err != nil {
		return nil, 0, fmt.Errorf("list tickets: %w", err)
	}
	return tickets, total, nil
}
