package model

import "time"

// Ticket status constants.
const (
	TicketStatusPending    = "pending"
	TicketStatusInProgress = "in_progress"
	TicketStatusCompleted  = "completed"
	TicketStatusFailed     = "failed"
	TicketStatusCancelled  = "cancelled"
	TicketStatusRejected   = "rejected"
)

// Ticket represents a network ticket record.
type Ticket struct {
	ID            int64      `db:"id"             json:"id"`
	TicketNo      string     `db:"ticket_no"      json:"ticket_no"`
	AlertSourceID int64      `db:"alert_source_id" json:"alert_source_id"`
	SourceType    string     `db:"source_type"    json:"source_type"`
	AlertRaw      JSON       `db:"alert_raw"      json:"alert_raw"`
	AlertParsed   JSON       `db:"alert_parsed"   json:"alert_parsed"`
	Title         string     `db:"title"          json:"title"`
	Description   *string    `db:"description"    json:"description"`
	Severity      string     `db:"severity"       json:"severity"`
	Status        string     `db:"status"         json:"status"`
	ClientID      *int64     `db:"client_id"      json:"client_id"`
	ExternalID    *string    `db:"external_id"    json:"external_id"`
	CallbackData  JSON       `db:"callback_data"  json:"callback_data"`
	Fingerprint   *string    `db:"fingerprint"    json:"fingerprint"`
	TimeoutAt     *time.Time `db:"timeout_at"     json:"timeout_at"`
	CreatedAt     time.Time  `db:"created_at"     json:"created_at"`
	UpdatedAt     time.Time  `db:"updated_at"     json:"updated_at"`
}

// TicketFilter holds parameters for listing tickets.
type TicketFilter struct {
	Status   string `json:"status"`
	ClientID int64  `json:"client_id"`
	Severity string `json:"severity"`
	Keyword  string `json:"keyword"`
	Page     int    `json:"page"`
	PageSize int    `json:"page_size"`
}
