package model

import "time"

// AlertRecord represents a raw alert appended to a ticket (for deduplication).
type AlertRecord struct {
	ID         int64     `db:"id"          json:"id"`
	TicketID   int64     `db:"ticket_id"   json:"ticket_id"`
	AlertRaw   JSON      `db:"alert_raw"   json:"alert_raw"`
	AlertParsed JSON     `db:"alert_parsed" json:"alert_parsed"`
	ReceivedAt time.Time `db:"received_at" json:"received_at"`
}
