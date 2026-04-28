package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
)

type NonceRepo struct {
	db *sqlx.DB
}

func NewNonceRepo(db *sqlx.DB) *NonceRepo {
	return &NonceRepo{db: db}
}

// CheckAndSet attempts to insert a nonce. Returns true if the nonce was new (inserted),
// false if it already existed. Uses INSERT IGNORE to handle duplicates atomically.
func (r *NonceRepo) CheckAndSet(ctx context.Context, nonce string, ttl time.Duration) (bool, error) {
	query := `INSERT IGNORE INTO nonce_records (nonce) VALUES (?)`
	result, err := r.db.ExecContext(ctx, query, nonce)
	if err != nil {
		return false, fmt.Errorf("check_and_set nonce: %w", err)
	}
	affected, _ := result.RowsAffected()
	return affected > 0, nil
}

// CleanExpired removes nonce records older than the given TTL.
func (r *NonceRepo) CleanExpired(ctx context.Context, ttl time.Duration) error {
	cutoff := time.Now().Add(-ttl)
	query := `DELETE FROM nonce_records WHERE created_at < ?`
	if _, err := r.db.ExecContext(ctx, query, cutoff); err != nil {
		return fmt.Errorf("clean expired nonces: %w", err)
	}
	return nil
}
