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
	AlertSourceID *int64     `db:"alert_source_id" json:"alert_source_id"`
	SourceType    string     `db:"source_type"    json:"source_type"`
	TicketTypeID  *int64     `db:"ticket_type_id" json:"ticket_type_id"`
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
	Status       string `json:"status"        form:"status"`
	ClientID     int64  `json:"client_id"     form:"client_id"`
	Severity     string `json:"severity"      form:"severity"`
	Keyword      string `json:"keyword"       form:"keyword"`
	TicketTypeID int64  `json:"ticket_type_id" form:"ticket_type_id"`
	Page         int    `json:"page"          form:"page"`
	PageSize     int    `json:"page_size"     form:"page_size"`
}
