package model

import "time"

// TicketLog represents an audit trail entry for ticket state changes.
type TicketLog struct {
	ID        int64     `db:"id"         json:"id"`
	TicketID  int64     `db:"ticket_id"  json:"ticket_id"`
	Action    string    `db:"action"     json:"action"`
	FromState *string   `db:"from_state" json:"from_state"`
	ToState   *string   `db:"to_state"   json:"to_state"`
	Operator  *string   `db:"operator"   json:"operator"`
	Detail    JSON      `db:"detail"     json:"detail"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
}
