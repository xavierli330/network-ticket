package repository

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"

	"github.com/xavierli/network-ticket/internal/model"
)

type AuditLogRepo struct {
	db *sqlx.DB
}

func NewAuditLogRepo(db *sqlx.DB) *AuditLogRepo {
	return &AuditLogRepo{db: db}
}

// Create inserts a new audit log entry.
func (r *AuditLogRepo) Create(ctx context.Context, al *model.AuditLog) (int64, error) {
	query := `INSERT INTO audit_logs (actor, action, resource_type, resource_id, detail, ip_address)
		VALUES (?, ?, ?, ?, ?, ?)`
	result, err := r.db.ExecContext(ctx, query,
		al.Actor, al.Action, al.ResourceType, al.ResourceID, al.Detail, al.IPAddress,
	)
	if err != nil {
		return 0, fmt.Errorf("insert audit_log: %w", err)
	}
	id, _ := result.LastInsertId()
	return id, nil
}

// List returns a paginated list of audit logs.
func (r *AuditLogRepo) List(ctx context.Context, page, pageSize int, operator string) ([]model.AuditLog, int, error) {
	// Count total.
	var total int
	var args []interface{}
	countQuery := `SELECT COUNT(*) FROM audit_logs`
	if operator != "" {
		countQuery += ` WHERE actor = ?`
		args = append(args, operator)
	}
	if err := r.db.GetContext(ctx, &total, countQuery, args...); err != nil {
		return nil, 0, fmt.Errorf("count audit_logs: %w", err)
	}

	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	var logs []model.AuditLog
	var listArgs []interface{}
	listQuery := `SELECT * FROM audit_logs`
	if operator != "" {
		listQuery += ` WHERE actor = ?`
		listArgs = append(listArgs, operator)
	}
	listQuery += ` ORDER BY created_at DESC LIMIT ? OFFSET ?`
	listArgs = append(listArgs, pageSize, offset)
	if err := r.db.SelectContext(ctx, &logs, listQuery, listArgs...); err != nil {
		return nil, 0, fmt.Errorf("list audit_logs: %w", err)
	}
	return logs, total, nil
}
