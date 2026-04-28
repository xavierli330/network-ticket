package model

import "time"

// Client represents an external client system.
type Client struct {
	ID          int64     `db:"id"           json:"id"`
	Name        string    `db:"name"         json:"name"`
	APIEndpoint string    `db:"api_endpoint" json:"api_endpoint"`
	APIKey      string    `db:"api_key"      json:"-"`
	HMACSecret  string    `db:"hmac_secret"  json:"-"`
	CallbackURL *string   `db:"callback_url" json:"callback_url"`
	Config      JSON      `db:"config"       json:"config"`
	Status      string    `db:"status"       json:"status"`
	CreatedAt   time.Time `db:"created_at"   json:"created_at"`
	UpdatedAt   time.Time `db:"updated_at"   json:"updated_at"`
}
