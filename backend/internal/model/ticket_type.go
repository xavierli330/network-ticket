package model

import "time"

// TicketType represents a category of ticket.
type TicketType struct {
	ID          int64     `db:"id"          json:"id"`
	Code        string    `db:"code"        json:"code"`
	Name        string    `db:"name"        json:"name"`
	Description *string   `db:"description" json:"description"`
	Color       string    `db:"color"       json:"color"`
	Status      string    `db:"status"      json:"status"`
	CreatedAt   time.Time `db:"created_at"  json:"created_at"`
	UpdatedAt   time.Time `db:"updated_at"  json:"updated_at"`
}
