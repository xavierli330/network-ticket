package repository

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"

	"github.com/xavierli/network-ticket/internal/model"
)

type AlertSourceRepo struct {
	db *sqlx.DB
}

func NewAlertSourceRepo(db *sqlx.DB) *AlertSourceRepo {
	return &AlertSourceRepo{db: db}
}

// GetByID returns an alert source by its primary key.
func (r *AlertSourceRepo) GetByID(ctx context.Context, id int64) (*model.AlertSource, error) {
	var as model.AlertSource
	query := `SELECT * FROM alert_sources WHERE id = ?`
	if err := r.db.GetContext(ctx, &as, query, id); err != nil {
		return nil, fmt.Errorf("get alert_source by id: %w", err)
	}
	return &as, nil
}

// List returns all active alert sources.
func (r *AlertSourceRepo) List(ctx context.Context) ([]model.AlertSource, error) {
	var sources []model.AlertSource
	query := `SELECT * FROM alert_sources ORDER BY id`
	if err := r.db.SelectContext(ctx, &sources, query); err != nil {
		return nil, fmt.Errorf("list alert_sources: %w", err)
	}
	return sources, nil
}

// Create inserts a new alert source.
func (r *AlertSourceRepo) Create(ctx context.Context, as *model.AlertSource) (int64, error) {
	query := `INSERT INTO alert_sources
		(name, type, config, parser_config, webhook_secret, poll_endpoint, poll_interval, dedup_fields, dedup_window_sec, status)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	result, err := r.db.ExecContext(ctx, query,
		as.Name, as.Type, as.Config, as.ParserConfig, as.WebhookSecret,
		as.PollEndpoint, as.PollInterval, as.DedupFields, as.DedupWindow, as.Status,
	)
	if err != nil {
		return 0, fmt.Errorf("insert alert_source: %w", err)
	}
	id, _ := result.LastInsertId()
	return id, nil
}

// Update updates an alert source.
func (r *AlertSourceRepo) Update(ctx context.Context, as *model.AlertSource) error {
	query := `UPDATE alert_sources SET
		name = ?, type = ?, config = ?, parser_config = ?, webhook_secret = ?,
		poll_endpoint = ?, poll_interval = ?, dedup_fields = ?, dedup_window_sec = ?, status = ?
		WHERE id = ?`
	if _, err := r.db.ExecContext(ctx, query,
		as.Name, as.Type, as.Config, as.ParserConfig, as.WebhookSecret,
		as.PollEndpoint, as.PollInterval, as.DedupFields, as.DedupWindow, as.Status, as.ID,
	); err != nil {
		return fmt.Errorf("update alert_source: %w", err)
	}
	return nil
}

// Delete removes an alert source by ID.
func (r *AlertSourceRepo) Delete(ctx context.Context, id int64) error {
	query := `DELETE FROM alert_sources WHERE id = ?`
	if _, err := r.db.ExecContext(ctx, query, id); err != nil {
		return fmt.Errorf("delete alert_source: %w", err)
	}
	return nil
}
