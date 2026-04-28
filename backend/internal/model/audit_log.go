package model

import "time"

// AuditLog represents a system-wide audit log entry.
type AuditLog struct {
	ID           int64     `db:"id"            json:"id"`
	Actor        string    `db:"actor"         json:"actor"`
	Action       string    `db:"action"        json:"action"`
	ResourceType string    `db:"resource_type" json:"resource_type"`
	ResourceID   *int64    `db:"resource_id"   json:"resource_id"`
	Detail       JSON      `db:"detail"        json:"detail"`
	IPAddress    *string   `db:"ip_address"    json:"ip_address"`
	CreatedAt    time.Time `db:"created_at"    json:"created_at"`
}
