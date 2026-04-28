package nonce

import (
	"context"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
)

// DBStore implements Store backed by a MySQL nonce_records table.
type DBStore struct {
	db *sqlx.DB
}

// NewDBStore creates a new DBStore. The nonce_records table must already exist
// (created via migration 009).
func NewDBStore(db *sqlx.DB) *DBStore {
	return &DBStore{db: db}
}

// CheckAndSet inserts the nonce using INSERT IGNORE. Returns true when the row
// was inserted (new nonce), false when the nonce already existed.
func (s *DBStore) CheckAndSet(ctx context.Context, nonce string, ttl time.Duration) (bool, error) {
	result, err := s.db.ExecContext(ctx,
		"INSERT IGNORE INTO nonce_records (nonce, created_at) VALUES (?, ?)",
		nonce, time.Now(),
	)
	if err != nil {
		return false, fmt.Errorf("insert nonce: %w", err)
	}
	affected, _ := result.RowsAffected()
	return affected > 0, nil
}

// Clean removes nonce records older than 5 minutes.
func (s *DBStore) Clean(ctx context.Context) error {
	cutoff := time.Now().Add(-5 * time.Minute)
	_, err := s.db.ExecContext(ctx, "DELETE FROM nonce_records WHERE created_at < ?", cutoff)
	return err
}
