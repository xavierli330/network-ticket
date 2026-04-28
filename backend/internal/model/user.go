package model

import "time"

// User role constants.
const (
	UserRoleAdmin    = "admin"
	UserRoleOperator = "operator"
)

// User represents a system user.
type User struct {
	ID        int64     `db:"id"         json:"id"`
	Username  string    `db:"username"   json:"username"`
	Password  string    `db:"password"   json:"-"`
	Role      string    `db:"role"       json:"role"`
	Status    string    `db:"status"     json:"status"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
}
