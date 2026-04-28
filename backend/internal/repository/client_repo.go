package repository

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"

	"github.com/xavierli/network-ticket/internal/model"
)

type ClientRepo struct {
	db *sqlx.DB
}

func NewClientRepo(db *sqlx.DB) *ClientRepo {
	return &ClientRepo{db: db}
}

// GetByID returns a client by its primary key.
func (r *ClientRepo) GetByID(ctx context.Context, id int64) (*model.Client, error) {
	var c model.Client
	query := `SELECT * FROM clients WHERE id = ?`
	if err := r.db.GetContext(ctx, &c, query, id); err != nil {
		return nil, fmt.Errorf("get client by id: %w", err)
	}
	return &c, nil
}

// GetByAPIKey returns a client by its API key.
func (r *ClientRepo) GetByAPIKey(ctx context.Context, apiKey string) (*model.Client, error) {
	var c model.Client
	query := `SELECT * FROM clients WHERE api_key = ?`
	if err := r.db.GetContext(ctx, &c, query, apiKey); err != nil {
		return nil, fmt.Errorf("get client by api_key: %w", err)
	}
	return &c, nil
}

// List returns all active clients.
func (r *ClientRepo) List(ctx context.Context) ([]model.Client, error) {
	var clients []model.Client
	query := `SELECT * FROM clients ORDER BY id`
	if err := r.db.SelectContext(ctx, &clients, query); err != nil {
		return nil, fmt.Errorf("list clients: %w", err)
	}
	return clients, nil
}

// Create inserts a new client.
func (r *ClientRepo) Create(ctx context.Context, c *model.Client) (int64, error) {
	query := `INSERT INTO clients (name, api_endpoint, api_key, hmac_secret, callback_url, config, status)
		VALUES (?, ?, ?, ?, ?, ?, ?)`
	result, err := r.db.ExecContext(ctx, query,
		c.Name, c.APIEndpoint, c.APIKey, c.HMACSecret, c.CallbackURL, c.Config, c.Status,
	)
	if err != nil {
		return 0, fmt.Errorf("insert client: %w", err)
	}
	id, _ := result.LastInsertId()
	return id, nil
}

// Update updates a client.
func (r *ClientRepo) Update(ctx context.Context, c *model.Client) error {
	query := `UPDATE clients SET
		name = ?, api_endpoint = ?, api_key = ?, hmac_secret = ?,
		callback_url = ?, config = ?, status = ?
		WHERE id = ?`
	if _, err := r.db.ExecContext(ctx, query,
		c.Name, c.APIEndpoint, c.APIKey, c.HMACSecret,
		c.CallbackURL, c.Config, c.Status, c.ID,
	); err != nil {
		return fmt.Errorf("update client: %w", err)
	}
	return nil
}

// Delete removes a client by ID.
func (r *ClientRepo) Delete(ctx context.Context, id int64) error {
	query := `DELETE FROM clients WHERE id = ?`
	if _, err := r.db.ExecContext(ctx, query, id); err != nil {
		return fmt.Errorf("delete client: %w", err)
	}
	return nil
}
