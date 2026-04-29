package repository

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"

	"github.com/xavierli/network-ticket/internal/model"
)

type UserRepo struct {
	db *sqlx.DB
}

func NewUserRepo(db *sqlx.DB) *UserRepo {
	return &UserRepo{db: db}
}

// GetByUsername returns a user by username (includes password for auth check).
func (r *UserRepo) GetByUsername(ctx context.Context, username string) (*model.User, error) {
	var u model.User
	query := `SELECT * FROM users WHERE username = ?`
	if err := r.db.GetContext(ctx, &u, query, username); err != nil {
		return nil, fmt.Errorf("get user by username: %w", err)
	}
	return &u, nil
}

// GetByID returns a user by primary key.
func (r *UserRepo) GetByID(ctx context.Context, id int64) (*model.User, error) {
	var u model.User
	query := `SELECT * FROM users WHERE id = ?`
	if err := r.db.GetContext(ctx, &u, query, id); err != nil {
		return nil, fmt.Errorf("get user by id: %w", err)
	}
	return &u, nil
}

// List returns paginated users (password excluded).
func (r *UserRepo) List(ctx context.Context, page, pageSize int) ([]model.User, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize
	var users []model.User
	query := `SELECT id, username, role, status, created_at, updated_at FROM users ORDER BY id LIMIT ? OFFSET ?`
	if err := r.db.SelectContext(ctx, &users, query, pageSize, offset); err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}
	return users, nil
}

// Count returns the total number of users.
func (r *UserRepo) Count(ctx context.Context) (int, error) {
	var total int
	query := `SELECT COUNT(*) FROM users`
	if err := r.db.GetContext(ctx, &total, query); err != nil {
		return 0, fmt.Errorf("count users: %w", err)
	}
	return total, nil
}

// Create inserts a new user.
func (r *UserRepo) Create(ctx context.Context, u *model.User) (int64, error) {
	query := `INSERT INTO users (username, password, role, status) VALUES (?, ?, ?, ?)`
	result, err := r.db.ExecContext(ctx, query, u.Username, u.Password, u.Role, u.Status)
	if err != nil {
		return 0, fmt.Errorf("insert user: %w", err)
	}
	id, _ := result.LastInsertId()
	return id, nil
}

// Update updates a user.
func (r *UserRepo) Update(ctx context.Context, u *model.User) error {
	query := `UPDATE users SET username = ?, password = ?, role = ?, status = ? WHERE id = ?`
	if _, err := r.db.ExecContext(ctx, query, u.Username, u.Password, u.Role, u.Status, u.ID); err != nil {
		return fmt.Errorf("update user: %w", err)
	}
	return nil
}

// Delete removes a user by ID.
func (r *UserRepo) Delete(ctx context.Context, id int64) error {
	query := `DELETE FROM users WHERE id = ?`
	if _, err := r.db.ExecContext(ctx, query, id); err != nil {
		return fmt.Errorf("delete user: %w", err)
	}
	return nil
}
